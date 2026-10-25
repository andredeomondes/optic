// Package filestore keeps OPTIC's small local state as files in the config
// directory (%APPDATA%\optic, ~/Library/Application Support/optic,
// ~/.config/optic): the last TV used and window preferences.
package filestore

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"optic/internal/domain"
)

const (
	lastHostFile = "last_host"
	prefsFile    = "prefs.json"
	knownFile    = "known_tvs.json"
)

// Prefs are user preferences owned by the desktop shell.
type Prefs struct {
	// CloseToTray makes the window's close button hide to the tray.
	CloseToTray bool `json:"closeToTray"`
}

// DefaultPrefs apply when nothing was saved yet.
var DefaultPrefs = Prefs{CloseToTray: true}

// Store reads and writes files in Dir. Errors on read fall back to defaults:
// this is convenience state, never worth failing the app over.
type Store struct{ Dir string }

func (s Store) path(name string) string { return filepath.Join(s.Dir, name) }

func (s Store) LastHost() string {
	b, _ := os.ReadFile(s.path(lastHostFile))
	return strings.TrimSpace(string(b))
}

func (s Store) SaveLastHost(host string) error {
	return os.WriteFile(s.path(lastHostFile), []byte(host), 0o600)
}

func (s Store) LoadPrefs() Prefs {
	p := DefaultPrefs
	if b, err := os.ReadFile(s.path(prefsFile)); err == nil {
		_ = json.Unmarshal(b, &p)
	}
	return p
}

func (s Store) SavePrefs(p Prefs) error {
	b, err := json.Marshal(p)
	if err != nil {
		return err
	}
	return os.WriteFile(s.path(prefsFile), b, 0o600)
}

// knownMu serializes read-modify-write of known_tvs.json across Store copies.
var knownMu sync.Mutex

func (s Store) readKnown() map[string]domain.KnownTV {
	m := map[string]domain.KnownTV{}
	if b, err := os.ReadFile(s.path(knownFile)); err == nil {
		_ = json.Unmarshal(b, &m)
	}
	return m
}

// Get returns the TV remembered for ip.
func (s Store) Get(ip string) (domain.KnownTV, bool) {
	knownMu.Lock()
	defer knownMu.Unlock()
	tv, ok := s.readKnown()[ip]
	return tv, ok
}

// Put remembers tv by IP. Empty fields in tv keep what was stored before.
func (s Store) Put(tv domain.KnownTV) error {
	knownMu.Lock()
	defer knownMu.Unlock()
	m := s.readKnown()
	cur := m[tv.IP]
	cur.IP = tv.IP
	if tv.Name != "" {
		cur.Name = tv.Name
	}
	if tv.Platform != domain.PlatformUnknown {
		cur.Platform = tv.Platform
	}
	if tv.MAC != "" {
		cur.MAC = tv.MAC
	}
	m[tv.IP] = cur
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return os.WriteFile(s.path(knownFile), b, 0o600)
}
