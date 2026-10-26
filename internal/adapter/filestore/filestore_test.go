package filestore

import (
	"fmt"
	"reflect"
	"sync"

	"optic/internal/domain"
	"os"
	"path/filepath"
	"testing"
)

func TestLastHostRoundTrip(t *testing.T) {
	s := Store{Dir: t.TempDir()}
	if got := s.LastHost(); got != "" {
		t.Fatalf("empty store returned %q", got)
	}
	if err := s.SaveLastHost("192.168.0.18"); err != nil {
		t.Fatal(err)
	}
	if got := s.LastHost(); got != "192.168.0.18" {
		t.Fatalf("got %q", got)
	}
}

func TestPrefsDefaultsAndRoundTrip(t *testing.T) {
	s := Store{Dir: t.TempDir()}
	if got := s.LoadPrefs(); !reflect.DeepEqual(got, DefaultPrefs) {
		t.Fatalf("defaults: got %+v", got)
	}
	if err := s.SavePrefs(Prefs{CloseToTray: false}); err != nil {
		t.Fatal(err)
	}
	if got := s.LoadPrefs(); got.CloseToTray {
		t.Fatalf("saved false, loaded %+v", got)
	}
}

func TestCorruptPrefsFallBackToDefaults(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, prefsFile), []byte("{not json"), 0o600)
	if got := (Store{Dir: dir}).LoadPrefs(); !reflect.DeepEqual(got, DefaultPrefs) {
		t.Fatalf("got %+v", got)
	}
}

func TestKnownTVsMerge(t *testing.T) {
	s := Store{Dir: t.TempDir()}
	_ = s.Put(domain.KnownTV{IP: "192.168.0.18", Platform: domain.PlatformAndroidTV, MAC: "aa:bb:cc:dd:ee:ff"})
	_ = s.Put(domain.KnownTV{IP: "192.168.0.18", Name: "UnionTV"})
	got, ok := s.Get("192.168.0.18")
	if !ok || got.MAC != "aa:bb:cc:dd:ee:ff" || got.Name != "UnionTV" || got.Platform != domain.PlatformAndroidTV {
		t.Fatalf("got %+v", got)
	}
	if _, ok := s.Get("10.0.0.1"); ok {
		t.Fatal("unknown ip reported as known")
	}
}

func TestKnownTVsConcurrentPut(t *testing.T) {
	s := Store{Dir: t.TempDir()}
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = s.Put(domain.KnownTV{IP: fmt.Sprintf("10.0.0.%d", i)})
		}()
	}
	wg.Wait()
	for i := 0; i < 20; i++ {
		if _, ok := s.Get(fmt.Sprintf("10.0.0.%d", i)); !ok {
			t.Fatalf("lost write for 10.0.0.%d", i)
		}
	}
}

func TestKnownTVsList(t *testing.T) {
	s := Store{Dir: t.TempDir()}
	_ = s.Put(domain.KnownTV{IP: "10.0.0.2", MAC: "aa:bb:cc:dd:ee:ff"})
	_ = s.Put(domain.KnownTV{IP: "10.0.0.1", Name: "Sala"})
	got := s.List()
	if len(got) != 2 || got[0].IP != "10.0.0.1" || got[1].MAC != "aa:bb:cc:dd:ee:ff" {
		t.Fatalf("got %+v", got)
	}
	if entries, _ := os.ReadDir(s.Dir); len(entries) != 1 {
		t.Fatalf("temp file left behind: %v", entries)
	}
}

func TestHotkeyDefaults(t *testing.T) {
	h := (Store{Dir: t.TempDir()}).LoadPrefs().Hotkeys
	if h.Enabled || h.Media {
		t.Fatalf("hotkeys must start off: %+v", h)
	}
	want := map[string]string{"volume_up": "Ctrl+Alt+Up", "volume_down": "Ctrl+Alt+Down", "mute": "Ctrl+Alt+M", "home": "Ctrl+Alt+H", "palette": "Ctrl+Alt+K"}
	if !reflect.DeepEqual(h.Bindings, want) {
		t.Fatalf("bindings: %+v", h.Bindings)
	}
}

func TestOldPrefsFileGetsHotkeyDefaults(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, prefsFile), []byte(`{"closeToTray":false}`), 0o600)
	p := (Store{Dir: dir}).LoadPrefs()
	if p.CloseToTray || p.Hotkeys.Bindings["home"] != "Ctrl+Alt+H" || len(p.Hotkeys.Bindings) != 5 {
		t.Fatalf("got %+v", p)
	}
}

func TestPartialBindingsKeepSavedAndFillMissing(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, prefsFile), []byte(`{"hotkeys":{"enabled":true,"bindings":{"home":"Ctrl+Alt+J"}}}`), 0o600)
	p := (Store{Dir: dir}).LoadPrefs()
	if !p.Hotkeys.Enabled || p.Hotkeys.Bindings["home"] != "Ctrl+Alt+J" || p.Hotkeys.Bindings["mute"] != "Ctrl+Alt+M" {
		t.Fatalf("got %+v", p.Hotkeys)
	}
}

func TestHotkeysRoundTripAndNoAliasing(t *testing.T) {
	s := Store{Dir: t.TempDir()}
	p := s.LoadPrefs()
	p.Hotkeys.Enabled, p.Hotkeys.Media = true, true
	p.Hotkeys.Bindings["home"] = "Ctrl+Shift+H"
	if err := s.SavePrefs(p); err != nil {
		t.Fatal(err)
	}
	// load → modify → save of another field keeps hotkeys
	q := s.LoadPrefs()
	q.CloseToTray = false
	_ = s.SavePrefs(q)
	got := s.LoadPrefs()
	if got.CloseToTray || !got.Hotkeys.Enabled || !got.Hotkeys.Media || got.Hotkeys.Bindings["home"] != "Ctrl+Shift+H" {
		t.Fatalf("got %+v", got)
	}
	if DefaultPrefs.Hotkeys.Bindings["home"] != "Ctrl+Alt+H" {
		t.Fatal("defaults were mutated")
	}
}
