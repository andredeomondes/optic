// Package filestore keeps OPTIC's small local state as files in the config
// directory (%APPDATA%\optic, ~/Library/Application Support/optic,
// ~/.config/optic): the last TV used and window preferences.
package filestore

import (
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"slices"
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
	// Hotkeys are the global keyboard shortcuts.
	Hotkeys HotkeyPrefs `json:"hotkeys"`
}

// HotkeyPrefs configures global shortcuts. Bindings maps an action id to an
// accelerator such as "Ctrl+Alt+H".
type HotkeyPrefs struct {
	Enabled  bool              `json:"enabled"`
	Media    bool              `json:"media"`
	Bindings map[string]string `json:"bindings"`
}

// DefaultPrefs apply when nothing was saved yet. Do not mutate its map;
// LoadPrefs returns a copy.
var DefaultPrefs = Prefs{
	CloseToTray: true,
	Hotkeys: HotkeyPrefs{Bindings: map[string]string{
		"volume_up":   "Ctrl+Alt+Up",
		"volume_down": "Ctrl+Alt+Down",
		"mute":        "Ctrl+Alt+M",
		"home":        "Ctrl+Alt+H",
		"palette":     "Ctrl+Alt+K",
	}},
}

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
	p.Hotkeys.Bindings = maps.Clone(DefaultPrefs.Hotkeys.Bindings)
	if b, err := os.ReadFile(s.path(prefsFile)); err == nil {
		var saved Prefs
		saved.CloseToTray = p.CloseToTray
		if json.Unmarshal(b, &saved) == nil {
			p.CloseToTray = saved.CloseToTray
			p.Hotkeys.Enabled = saved.Hotkeys.Enabled
			p.Hotkeys.Media = saved.Hotkeys.Media
			// Saved bindings win; actions missing from the file keep defaults.
			maps.Copy(p.Hotkeys.Bindings, saved.Hotkeys.Bindings)
		}
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
	return writeAtomic(s.path(knownFile), b)
}

// List returns every remembered TV, ordered by IP.
func (s Store) List() []domain.KnownTV {
	knownMu.Lock()
	defer knownMu.Unlock()
	m := s.readKnown()
	out := make([]domain.KnownTV, 0, len(m))
	for _, tv := range m {
		out = append(out, tv)
	}
	slices.SortFunc(out, func(a, b domain.KnownTV) int { return strings.Compare(a.IP, b.IP) })
	return out
}

// writeAtomic writes via a temp file and rename so a crash never leaves a
// half-written file behind.
func writeAtomic(path string, b []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}
