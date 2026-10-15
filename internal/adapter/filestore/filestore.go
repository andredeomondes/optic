// Package filestore keeps OPTIC's small local state as files in the config
// directory (%APPDATA%\optic, ~/Library/Application Support/optic,
// ~/.config/optic): the last TV used and window preferences.
package filestore

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

const (
	lastHostFile = "last_host"
	prefsFile    = "prefs.json"
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
