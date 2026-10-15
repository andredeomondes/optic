package atvremote

import (
	"testing"

	"google.golang.org/protobuf/encoding/protowire"
)

func TestImeBatchEdit(t *testing.T) {
	b := imeBatchEdit(imeState{known: true, imeCounter: 4, fieldCounter: 9}, "olá")
	f, err := parseFields(b)
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := f.uint(1); v != 4 {
		t.Errorf("ime_counter = %d", v)
	}
	if v, _ := f.uint(2); v != 9 {
		t.Errorf("field_counter = %d", v)
	}
	edit, _ := f.bytes(3)
	ef, _ := parseFields(edit)
	if v, _ := ef.uint(1); v != 1 {
		t.Errorf("insert = %d", v)
	}
	obj, _ := ef.bytes(2)
	of, _ := parseFields(obj)
	// cursor counts runes, not bytes: "olá" is 3 runes → index 2
	if v, _ := of.uint(1); v != 2 {
		t.Errorf("start = %d", v)
	}
	if v, _ := of.bytes(3); string(v) != "olá" {
		t.Errorf("value = %q", v)
	}
}

func TestHandleImeShow(t *testing.T) {
	r := &Remote{}
	status := appendString(appendUint(nil, 1, 7), 2, "")
	show := appendMsg(nil, 2, status)
	f, _ := parseFields(appendMsg(nil, rmImeShow, show))
	r.handleIme(f)
	if !r.ime.known || r.ime.fieldCounter != 7 {
		t.Fatalf("ime = %+v", r.ime)
	}
}

func TestCharKey(t *testing.T) {
	cases := map[rune]int{'a': 29, 'z': 54, '0': 7, '9': 16, ' ': 62, '.': 56, '@': 77}
	for ch, want := range cases {
		if got, ok := charKey(ch); !ok || got != want {
			t.Errorf("charKey(%q) = %d, %v; want %d", ch, got, ok, want)
		}
	}
	if _, ok := charKey('ç'); ok {
		t.Error("ç should be unsupported")
	}
}

// Frames captured from a TCL Google TV opening the Google search keyboard.
func TestHandleImeCapturedFlow(t *testing.T) {
	r := &Remote{}
	feed := func(field protowire.Number, payload []byte) {
		f, err := parseFields(appendMsg(nil, field, payload))
		if err != nil {
			t.Fatal(err)
		}
		r.handleIme(f)
	}
	appInfo := func(pkg string) []byte { return appendString(nil, 12, pkg) }

	// On connect: only the foreground app.
	feed(rmImeKeyInject, appendMsg(nil, 1, appInfo("com.netflix.ninja")))
	if r.state.App != "com.netflix.ninja" || r.state.Keyboard || r.ime.known {
		t.Fatalf("after connect: state=%+v ime=%+v", r.state, r.ime)
	}

	// Keyboard opens: app info + text field status, then counters.
	status := appendUint(appendString(appendUint(nil, 1, 30), 2, ""), 5, 1)
	feed(rmImeKeyInject, appendMsg(appendMsg(nil, 1, appInfo("com.google.android.katniss")), 2, status))
	feed(rmImeBatchEdit, appendUint(appendUint(nil, 1, 1), 2, 3))
	if !r.state.Keyboard || r.state.App != "com.google.android.katniss" {
		t.Fatalf("keyboard open: state=%+v", r.state)
	}
	if !r.ime.known || r.ime.imeCounter != 1 || r.ime.fieldCounter != 3 {
		t.Fatalf("counters: %+v", r.ime)
	}

	// Focus leaves the field: counters reset.
	feed(rmImeKeyInject, appendMsg(nil, 1, appInfo("com.google.android.katniss")))
	if r.state.Keyboard || r.ime.known {
		t.Fatalf("after close: state=%+v ime=%+v", r.state, r.ime)
	}
}
