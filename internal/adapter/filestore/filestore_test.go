package filestore

import (
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
	if got := s.LoadPrefs(); got != DefaultPrefs {
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
	if got := (Store{Dir: dir}).LoadPrefs(); got != DefaultPrefs {
		t.Fatalf("got %+v", got)
	}
}
