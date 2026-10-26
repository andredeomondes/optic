// Package hotkey registers system-wide keyboard shortcuts (Windows RegisterHotKey).
package hotkey

import "sync"

// Binding ties an action ID to a key combination.
type Binding struct {
	ID    string
	Accel Accel
}

// Result reports whether one binding was registered.
type Result struct {
	ID  string
	OK  bool
	Err string
}

// Manager owns the process's global hotkeys. Calls are serialized.
type Manager interface {
	// Set replaces every registration with bindings and reports each one.
	Set(bindings []Binding) []Result
	// Close unregisters everything and stops the manager. Safe to call twice.
	Close()
}

const (
	errBusy        = "Em uso por outro programa"
	errUnsupported = "Atalhos globais indisponíveis neste sistema"
	errClosed      = "Atalhos globais desligados"
	errDupID       = "Ação repetida"
	errDupAccel    = "Atalho já usado em outra ação"
)

// maxID is the top of the application range for RegisterHotKey ids.
const maxID = 0xBFFF

// registrar is the thin OS layer; it only ever runs on the registration thread.
type registrar interface {
	register(id int, mods, vk uint32) bool
	unregister(id int)
}

// manager holds the portable logic. run executes a function on the
// registration thread (false if that thread is gone); stop ends the thread.
type manager struct {
	mu     sync.Mutex // serializes Set/Close
	closed bool
	run    func(func()) bool
	stop   func()

	// Touched only on the registration thread.
	reg    registrar
	onFire func(id string)
	next   int
	live   map[int]string // registrar id -> binding ID
}

func newManager(reg registrar, onFire func(string), run func(func()) bool, stop func()) *manager {
	return &manager{reg: reg, onFire: onFire, run: run, stop: stop, live: map[int]string{}}
}

func failAll(bindings []Binding, msg string) []Result {
	out := make([]Result, len(bindings))
	for i, b := range bindings {
		out[i] = Result{ID: b.ID, Err: msg}
	}
	return out
}

func (m *manager) Set(bindings []Binding) []Result {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return failAll(bindings, errClosed)
	}
	var out []Result
	if !m.run(func() { out = m.replace(bindings) }) {
		return failAll(bindings, errUnsupported)
	}
	return out
}

func (m *manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return
	}
	m.closed = true
	m.run(m.clear)
	m.stop()
}

// replace unregisters everything, then registers bindings in order.
func (m *manager) replace(bindings []Binding) []Result {
	m.clear()
	out := make([]Result, len(bindings))
	ids := map[string]bool{}
	accels := map[Accel]bool{}
	for i, b := range bindings {
		out[i] = Result{ID: b.ID}
		switch {
		case ids[b.ID]:
			out[i].Err = errDupID
			continue
		case accels[b.Accel]:
			out[i].Err = errDupAccel
			continue
		}
		if err := b.Accel.validate(); err != nil {
			out[i].Err = err.Error()
			continue
		}
		vk, _ := vkOf(b.Accel.Key)
		id := m.nextID()
		if !m.reg.register(id, modsOf(b.Accel), vk) {
			out[i].Err = errBusy
			continue
		}
		m.live[id] = b.ID
		ids[b.ID] = true
		accels[b.Accel] = true
		out[i].OK = true
	}
	return out
}

func (m *manager) clear() {
	for id := range m.live {
		m.reg.unregister(id)
		delete(m.live, id)
	}
}

// nextID never reuses a recent id, so a WM_HOTKEY queued before a
// replace cannot fire the binding that took its place.
func (m *manager) nextID() int {
	m.next++
	if m.next > maxID {
		m.next = 1
	}
	return m.next
}

// fire handles a WM_HOTKEY for registrar id n.
func (m *manager) fire(n int) {
	id, ok := m.live[n]
	if !ok || m.onFire == nil {
		return
	}
	go m.onFire(id)
}
