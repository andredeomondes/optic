package filestore

import (
	"fmt"
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
