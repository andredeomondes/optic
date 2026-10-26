package hotkey

import (
	"testing"
	"time"
)

// fakeReg records registrations; combos in busy fail like a hotkey owned by another program.
type fakeReg struct {
	live          map[int]uint64
	busy          map[uint64]bool
	registers     int
	unregisters   int
	badUnregister int
}

func newFakeReg() *fakeReg {
	return &fakeReg{live: map[int]uint64{}, busy: map[uint64]bool{}}
}

func combo(mods, vk uint32) uint64 { return uint64(mods)<<32 | uint64(vk) }

func (f *fakeReg) register(id int, mods, vk uint32) bool {
	f.registers++
	c := combo(mods, vk)
	if f.busy[c] {
		return false
	}
	for _, v := range f.live {
		if v == c {
			return false
		}
	}
	f.live[id] = c
	return true
}

func (f *fakeReg) unregister(id int) {
	f.unregisters++
	if _, ok := f.live[id]; !ok {
		f.badUnregister++
	}
	delete(f.live, id)
}

func (f *fakeReg) idFor(t *testing.T, a Accel) int {
	t.Helper()
	vk, _ := vkOf(a.Key)
	c := combo(modsOf(a), vk)
	for id, v := range f.live {
		if v == c {
			return id
		}
	}
	t.Fatalf("%s not registered", a)
	return 0
}

type harness struct {
	reg   *fakeReg
	m     *manager
	fired chan string
	stops int
}

func newHarness() *harness {
	h := &harness{reg: newFakeReg(), fired: make(chan string, 8)}
	h.m = newManager(h.reg, func(id string) { h.fired <- id },
		func(f func()) bool { f(); return true },
		func() { h.stops++ })
	return h
}

func mustParse(t *testing.T, s string) Accel {
	t.Helper()
	a, err := Parse(s)
	if err != nil {
		t.Fatalf("Parse(%q): %v", s, err)
	}
	return a
}

func TestSetReportsEachBinding(t *testing.T) {
	h := newHarness()
	up, down := mustParse(t, "Ctrl+Alt+Up"), mustParse(t, "Ctrl+Alt+Down")
	vk, _ := vkOf("Down")
	h.reg.busy[combo(modsOf(down), vk)] = true

	res := h.m.Set([]Binding{
		{ID: "volUp", Accel: up},
		{ID: "volDown", Accel: down},
		{ID: "bad", Accel: Accel{Ctrl: true, Key: "Nope"}},
		{ID: "shiftOnly", Accel: Accel{Shift: true, Key: "Up"}},
	})
	want := []Result{
		{ID: "volUp", OK: true},
		{ID: "volDown", Err: "Em uso por outro programa"},
		{ID: "bad", Err: "Tecla desconhecida: Nope"},
		{ID: "shiftOnly", Err: "Use Ctrl ou Alt junto com a tecla"},
	}
	if len(res) != len(want) {
		t.Fatalf("got %d results", len(res))
	}
	for i := range want {
		if res[i] != want[i] {
			t.Errorf("result %d = %+v, want %+v", i, res[i], want[i])
		}
	}
	if len(h.reg.live) != 1 {
		t.Errorf("live = %d, want 1", len(h.reg.live))
	}
}

func TestSetDuplicates(t *testing.T) {
	h := newHarness()
	a := mustParse(t, "Ctrl+Alt+Up")
	res := h.m.Set([]Binding{
		{ID: "x", Accel: a},
		{ID: "x", Accel: mustParse(t, "Ctrl+Alt+Down")},
		{ID: "y", Accel: a},
	})
	if !res[0].OK {
		t.Errorf("first = %+v", res[0])
	}
	if res[1].OK || res[1].Err != "Ação repetida" {
		t.Errorf("duplicate id = %+v", res[1])
	}
	if res[2].OK || res[2].Err != "Atalho já usado em outra ação" {
		t.Errorf("duplicate accel = %+v", res[2])
	}
	if len(h.reg.live) != 1 {
		t.Errorf("live = %d, want 1", len(h.reg.live))
	}
}

func TestSetReplacesAll(t *testing.T) {
	h := newHarness()
	h.m.Set([]Binding{{ID: "a", Accel: mustParse(t, "Ctrl+Alt+Up")}, {ID: "b", Accel: mustParse(t, "Ctrl+Alt+Down")}})
	res := h.m.Set([]Binding{{ID: "a", Accel: mustParse(t, "Ctrl+Alt+Up")}})
	if !res[0].OK {
		t.Fatalf("re-register same combo failed: %+v", res[0])
	}
	if len(h.reg.live) != 1 {
		t.Errorf("live = %d, want 1", len(h.reg.live))
	}
	h.m.Set(nil)
	if len(h.reg.live) != 0 {
		t.Errorf("live after empty Set = %d", len(h.reg.live))
	}
	if h.reg.badUnregister != 0 {
		t.Errorf("unregistered unknown ids %d times", h.reg.badUnregister)
	}
}

func TestCloseUnregistersAndIsIdempotent(t *testing.T) {
	h := newHarness()
	h.m.Set([]Binding{{ID: "a", Accel: mustParse(t, "Ctrl+Alt+Up")}, {ID: "b", Accel: mustAccel("MediaNext")}})
	h.m.Close()
	h.m.Close()
	if len(h.reg.live) != 0 {
		t.Errorf("live after Close = %d", len(h.reg.live))
	}
	if h.stops != 1 {
		t.Errorf("stop called %d times", h.stops)
	}
	regs := h.reg.registers
	res := h.m.Set([]Binding{{ID: "a", Accel: mustParse(t, "Ctrl+Alt+Up")}})
	if res[0].OK || h.reg.registers != regs {
		t.Errorf("Set after Close = %+v, registers %d -> %d", res[0], regs, h.reg.registers)
	}
}

func TestRepeatedSetDoesNotLeak(t *testing.T) {
	h := newHarness()
	bs := []Binding{{ID: "a", Accel: mustParse(t, "Ctrl+Alt+Up")}, {ID: "b", Accel: mustParse(t, "Ctrl+Alt+Down")}}
	for i := 0; i < 50; i++ {
		h.m.Set(bs)
	}
	h.m.Close()
	if len(h.reg.live) != 0 || h.reg.registers != h.reg.unregisters || h.reg.badUnregister != 0 {
		t.Errorf("leak: live=%d reg=%d unreg=%d bad=%d", len(h.reg.live), h.reg.registers, h.reg.unregisters, h.reg.badUnregister)
	}
}

func TestFireCallsOnFireWithID(t *testing.T) {
	h := newHarness()
	up := mustParse(t, "Ctrl+Alt+Up")
	h.m.Set([]Binding{{ID: "volUp", Accel: up}})
	old := h.reg.idFor(t, up)
	h.m.fire(old)
	select {
	case id := <-h.fired:
		if id != "volUp" {
			t.Errorf("fired %q", id)
		}
	case <-time.After(time.Second):
		t.Fatal("onFire not called")
	}

	// A hotkey queued before a replace must not fire the new binding.
	h.m.Set([]Binding{{ID: "other", Accel: up}})
	h.m.fire(old)
	h.m.fire(9999)
	select {
	case id := <-h.fired:
		t.Errorf("stale id fired %q", id)
	case <-time.After(50 * time.Millisecond):
	}
}

// mustAccel parses or panics.
func mustAccel(s string) Accel {
	a, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return a
}
