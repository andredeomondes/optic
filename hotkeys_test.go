package main

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"optic/internal/adapter/filestore"
	"optic/internal/domain"
	"optic/internal/platform/hotkey"
)

func defaultHK() filestore.HotkeyPrefs { return filestore.DefaultPrefs.Hotkeys }

func TestHotkeyBindingsDisabledIsEmpty(t *testing.T) {
	if got := hotkeyBindings(defaultHK()); len(got) != 0 {
		t.Fatalf("got %v", got)
	}
}

func TestHotkeyBindingsEnabledAndMedia(t *testing.T) {
	p := defaultHK()
	p.Enabled = true
	if n := len(hotkeyBindings(p)); n != 5 {
		t.Fatalf("want 5, got %d", n)
	}
	p.Media = true
	got := hotkeyBindings(p)
	if len(got) != 9 {
		t.Fatalf("want 9, got %d", len(got))
	}
	if got[5].ID != "media_play_pause" || got[5].Accel.Key != "MediaPlayPause" {
		t.Fatalf("media binding: %+v", got[5])
	}
	p.Enabled = false
	if len(hotkeyBindings(p)) != 0 {
		t.Fatal("media keys must not register while disabled")
	}
}

func TestHotkeyBindingsSkipsUnparsable(t *testing.T) {
	p := defaultHK()
	p.Enabled = true
	p.Bindings = map[string]string{"home": "nonsense", "mute": "Ctrl+Alt+M"}
	got := hotkeyBindings(p)
	if len(got) != 1 || got[0].ID != "mute" {
		t.Fatalf("got %+v", got)
	}
}

func TestValidateHotkeys(t *testing.T) {
	cur := defaultHK().Bindings
	t.Run("canonicalizes, fills missing, ignores unknown", func(t *testing.T) {
		out, err := validateHotkeys(map[string]string{"home": "ctrl+alt+j", "bogus": "x"}, cur)
		if err != nil {
			t.Fatal(err)
		}
		if out["home"] != "Ctrl+Alt+J" || out["mute"] != "Ctrl+Alt+M" || len(out) != 5 {
			t.Fatalf("got %v", out)
		}
	})
	t.Run("invalid accel names the action", func(t *testing.T) {
		_, err := validateHotkeys(map[string]string{"mute": "M"}, cur)
		if err == nil || !strings.HasPrefix(err.Error(), "Mudo: ") {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("duplicate", func(t *testing.T) {
		_, err := validateHotkeys(map[string]string{"home": "ctrl+alt+m"}, cur)
		if err == nil || err.Error() != "Atalho repetido: Ctrl+Alt+M" {
			t.Fatalf("got %v", err)
		}
	})
}

func TestBuildHotkeyState(t *testing.T) {
	p := defaultHK()
	p.Enabled, p.Media = true, true
	st := buildHotkeyState(true, p, map[string]string{"home": "Em uso por outro programa"})
	if !st.Supported || len(st.Bindings) != 9 || st.Bindings[0].ID != "volume_up" {
		t.Fatalf("got %+v", st)
	}
	if st.Bindings[3].ID != "home" || st.Bindings[3].Error == "" || st.Bindings[0].Error != "" {
		t.Fatalf("errors: %+v", st.Bindings)
	}
	p.Media = false
	if n := len(buildHotkeyState(true, p, nil).Bindings); n != 5 {
		t.Fatalf("want 5, got %d", n)
	}
}

func TestHotkeyErrors(t *testing.T) {
	got := hotkeyErrors([]hotkey.Result{{ID: "a", OK: true}, {ID: "b", Err: "x"}})
	if !reflect.DeepEqual(got, map[string]string{"b": "x"}) {
		t.Fatalf("got %v", got)
	}
}

type emitted struct {
	event string
	data  []any
}

func runDispatch(id string, keysErr error, visible bool) (sent []string, shown bool, ev []emitted) {
	keys := func(k string) error { sent = append(sent, k); return keysErr }
	dispatchHotkey(id, keys, hotkeyEffects{
		visible: func() bool { return visible },
		show:    func() { shown = true },
		emit:    func(e string, d ...any) { ev = append(ev, emitted{e, d}) },
	})
	return
}

func TestDispatchMapping(t *testing.T) {
	want := map[string]string{
		"volume_up": "VOLUME_UP", "volume_down": "VOLUME_DOWN", "mute": "MUTE", "home": "HOME",
		"media_play_pause": "MEDIA_PLAY_PAUSE", "media_next": "MEDIA_NEXT",
		"media_prev": "MEDIA_PREVIOUS", "media_stop": "MEDIA_STOP",
	}
	for id, key := range want {
		sent, shown, ev := runDispatch(id, nil, true)
		if len(sent) != 1 || sent[0] != key || shown || len(ev) != 0 {
			t.Errorf("%s: sent=%v shown=%v ev=%v", id, sent, shown, ev)
		}
	}
}

func TestDispatchPalette(t *testing.T) {
	sent, shown, ev := runDispatch("palette", nil, false)
	if len(sent) != 0 || !shown || len(ev) != 1 || ev[0].event != "hotkey:palette" {
		t.Fatalf("sent=%v shown=%v ev=%v", sent, shown, ev)
	}
}

func TestDispatchNotConnected(t *testing.T) {
	_, _, ev := runDispatch("home", domain.ErrNotConnected, true)
	if len(ev) != 1 || ev[0].event != "hotkey:error" || ev[0].data[0] != "Conecte uma TV primeiro" {
		t.Fatalf("visible: %v", ev)
	}
	_, shown, ev := runDispatch("home", domain.ErrNotConnected, false)
	if len(ev) != 0 || shown {
		t.Fatalf("hidden must drop silently: %v", ev)
	}
}

func TestDispatchOtherErrorAndUnknown(t *testing.T) {
	_, _, ev := runDispatch("mute", errors.New("falhou"), true)
	if len(ev) != 1 || ev[0].data[0] != "falhou" {
		t.Fatalf("got %v", ev)
	}
	if sent, _, ev := runDispatch("nope", nil, true); len(sent) != 0 || len(ev) != 0 {
		t.Fatal("unknown id must do nothing")
	}
}
