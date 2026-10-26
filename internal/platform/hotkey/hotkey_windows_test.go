//go:build windows

package hotkey

import (
	"testing"
	"time"
)

func TestWindowsStartSetClose(t *testing.T) {
	m := Start(func(string) {})
	done := make(chan struct{})
	defer func() {
		go func() { m.Close(); close(done) }()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("Close did not return")
		}
	}()

	a := mustParse(t, "Ctrl+Alt+Shift+F12")
	res := m.Set([]Binding{{ID: "probe", Accel: a}})
	if len(res) != 1 || res[0].ID != "probe" {
		t.Fatalf("Set = %+v", res)
	}
	if !res[0].OK {
		t.Skipf("combo unavailable on this machine: %s", res[0].Err)
	}
	// Replacing with the same combo must work: the old one is unregistered first.
	if res = m.Set([]Binding{{ID: "probe", Accel: a}}); !res[0].OK {
		t.Fatalf("re-Set = %+v", res[0])
	}
}
