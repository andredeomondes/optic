package main

import (
	"errors"
	"fmt"

	"optic/internal/adapter/filestore"
	"optic/internal/domain"
	"optic/internal/platform/hotkey"
)

// Events for global shortcuts.
const (
	evHotkeyPalette = "hotkey:palette"
	evHotkeyError   = "hotkey:error"
)

// hotkeyAction is one configurable shortcut. Key is the TV key it sends;
// the palette action has none (it opens the quick command).
type hotkeyAction struct {
	ID, Label, Key string
}

// hotkeyActions is the user-configurable table, in display order.
var hotkeyActions = []hotkeyAction{
	{"volume_up", "Volume +", "VOLUME_UP"},
	{"volume_down", "Volume −", "VOLUME_DOWN"},
	{"mute", "Mudo", "MUTE"},
	{"home", "Início", "HOME"},
	{"palette", "Comandos rápidos", ""},
}

// mediaActions use the keyboard's own media keys; the accelerator is fixed.
var mediaActions = []struct {
	hotkeyAction
	Accel string
}{
	{hotkeyAction{"media_play_pause", "Mídia: play/pause", "MEDIA_PLAY_PAUSE"}, "MediaPlayPause"},
	{hotkeyAction{"media_next", "Mídia: próxima", "MEDIA_NEXT"}, "MediaNext"},
	{hotkeyAction{"media_prev", "Mídia: anterior", "MEDIA_PREVIOUS"}, "MediaPrev"},
	{hotkeyAction{"media_stop", "Mídia: parar", "MEDIA_STOP"}, "MediaStop"},
}

// HotkeyBinding is one shortcut as the UI sees it. Error is the reason the
// last registration failed, "" when active or disabled.
type HotkeyBinding struct {
	ID    string `json:"id"`
	Accel string `json:"accel"`
	Error string `json:"error"`
}

// HotkeyState is what GetHotkeys/SetHotkeys return.
type HotkeyState struct {
	Supported bool            `json:"supported"`
	Enabled   bool            `json:"enabled"`
	Media     bool            `json:"media"`
	Bindings  []HotkeyBinding `json:"bindings"`
}

// validateHotkeys checks the accelerators the UI sent. Actions missing from
// in keep their value from current; unknown ids are ignored. It returns the
// full table, or the first problem as "<ação>: <motivo>".
func validateHotkeys(in, current map[string]string) (map[string]string, error) {
	out := map[string]string{}
	used := map[hotkey.Accel]bool{}
	for _, a := range hotkeyActions {
		raw, ok := in[a.ID]
		if !ok {
			raw = current[a.ID]
		}
		acc, err := hotkey.Parse(raw)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", a.Label, err)
		}
		if used[acc] {
			return nil, fmt.Errorf("Atalho repetido: %s", acc)
		}
		used[acc] = true
		out[a.ID] = acc.String()
	}
	return out, nil
}

// hotkeyBindings turns prefs into registrations: the five actions only when
// Enabled, the media keys whenever Media is on. Saved values that no longer
// parse are skipped.
func hotkeyBindings(p filestore.HotkeyPrefs) []hotkey.Binding {
	var out []hotkey.Binding
	if p.Enabled {
		for _, a := range hotkeyActions {
			if acc, err := hotkey.Parse(p.Bindings[a.ID]); err == nil {
				out = append(out, hotkey.Binding{ID: a.ID, Accel: acc})
			}
		}
	}
	if p.Media {
		for _, m := range mediaActions {
			acc, _ := hotkey.Parse(m.Accel)
			out = append(out, hotkey.Binding{ID: m.ID, Accel: acc})
		}
	}
	return out
}

// buildHotkeyState merges prefs with the per-binding errors of the last Set.
func buildHotkeyState(supported bool, p filestore.HotkeyPrefs, errs map[string]string) HotkeyState {
	st := HotkeyState{Supported: supported, Enabled: p.Enabled, Media: p.Media, Bindings: []HotkeyBinding{}}
	for _, a := range hotkeyActions {
		st.Bindings = append(st.Bindings, HotkeyBinding{ID: a.ID, Accel: p.Bindings[a.ID], Error: errs[a.ID]})
	}
	if p.Media {
		for _, m := range mediaActions {
			st.Bindings = append(st.Bindings, HotkeyBinding{ID: m.ID, Accel: m.Accel, Error: errs[m.ID]})
		}
	}
	return st
}

// hotkeyErrors collects the failed results by binding id.
func hotkeyErrors(rs []hotkey.Result) map[string]string {
	m := map[string]string{}
	for _, r := range rs {
		if !r.OK && r.Err != "" {
			m[r.ID] = r.Err
		}
	}
	return m
}

// hotkeyTVKey maps an action id to the TV key it sends.
func hotkeyTVKey(id string) (string, bool) {
	for _, a := range hotkeyActions {
		if a.ID == id && a.Key != "" {
			return a.Key, true
		}
	}
	for _, m := range mediaActions {
		if m.ID == id {
			return m.Key, true
		}
	}
	return "", false
}

// hotkeyEffects are the window side effects of a fired shortcut.
type hotkeyEffects struct {
	visible func() bool
	show    func()
	emit    func(event string, data ...any)
}

// dispatchHotkey runs the action behind a fired shortcut. Failures are
// only shown while the window is visible; a hidden app stays quiet.
func dispatchHotkey(id string, keys func(string) error, fx hotkeyEffects) {
	if id == "palette" {
		fx.show()
		fx.emit(evHotkeyPalette)
		return
	}
	key, ok := hotkeyTVKey(id)
	if !ok || keys == nil {
		return
	}
	err := keys(key)
	if err == nil || !fx.visible() {
		return
	}
	if errors.Is(err, domain.ErrNotConnected) {
		fx.emit(evHotkeyError, "Conecte uma TV primeiro")
		return
	}
	fx.emit(evHotkeyError, err.Error())
}
