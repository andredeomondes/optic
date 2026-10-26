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

func ids(bs []hotkey.Binding) []string {
	var out []string
	for _, b := range bs {
		out = append(out, b.ID)
	}
	return out
}

func TestHotkeyBindingsCombinations(t *testing.T) {
	actions := []string{"volume_up", "volume_down", "mute", "home", "palette"}
	media := []string{"media_play_pause", "media_next", "media_prev", "media_stop"}
	cases := []struct {
		name           string
		enabled, media bool
		want           []string
	}{
		{"none", false, false, nil},
		{"enabled only", true, false, actions},
		{"media only", false, true, media},
		{"both", true, true, append(append([]string{}, actions...), media...)},
	}
	for _, c := range cases {
		p := defaultHK()
		p.Enabled, p.Media = c.enabled, c.media
		got := hotkeyBindings(p)
		if !reflect.DeepEqual(ids(got), c.want) {
			t.Errorf("%s: got %v want %v", c.name, ids(got), c.want)
		}
	}
	p := defaultHK()
	p.Media = true
	if got := hotkeyBindings(p); got[0].Accel.Key != "MediaPlayPause" {
		t.Fatalf("media binding: %+v", got[0])
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
	p.Enabled, p.Media = false, true
	st = buildHotkeyState(true, p, map[string]string{"media_next": "Em uso por outro programa"})
	if len(st.Bindings) != 9 || st.Bindings[6].ID != "media_next" || st.Bindings[6].Error == "" {
		t.Fatalf("media without enabled: %+v", st.Bindings)
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

type fakeHK struct {
	sets   [][]hotkey.Binding
	closed bool
}

func (f *fakeHK) Set(b []hotkey.Binding) []hotkey.Result {
	f.sets = append(f.sets, b)
	return nil
}
func (f *fakeHK) Close() { f.closed = true }

func TestSuspendHotkeys(t *testing.T) {
	store := filestore.Store{Dir: t.TempDir()}
	s := newShell(store, nil)
	fake := &fakeHK{}
	s.hk = fake
	p := store.LoadPrefs()
	p.Hotkeys.Enabled = true
	if err := store.SavePrefs(p); err != nil {
		t.Fatal(err)
	}

	s.SuspendHotkeys(true)
	s.SuspendHotkeys(true)
	if len(fake.sets) != 1 || fake.sets[0] != nil {
		t.Fatalf("suspend must register nothing, once: %v", fake.sets)
	}
	s.SuspendHotkeys(false)
	s.SuspendHotkeys(false)
	if len(fake.sets) != 2 || len(fake.sets[1]) != 5 {
		t.Fatalf("resume must re-apply prefs, once: %v", fake.sets)
	}
	s.SuspendHotkeys(false) // never suspended again: no-op
	if len(fake.sets) != 2 {
		t.Fatalf("resume is idempotent: %v", fake.sets)
	}
}

func TestDispatchNotConnectedFollowsLanguage(t *testing.T) {
	for lang, want := range map[string]string{"en": "Connect a TV first", "pt-BR": "Conecte uma TV primeiro", "": "Conecte uma TV primeiro"} {
		var ev []emitted
		dispatchHotkey("home", func(string) error { return domain.ErrNotConnected }, hotkeyEffects{
			visible: func() bool { return true },
			show:    func() {},
			emit:    func(e string, d ...any) { ev = append(ev, emitted{e, d}) },
			lang:    func() string { return lang },
		})
		if len(ev) != 1 || ev[0].data[0] != want {
			t.Errorf("lang %q: got %v, want %q", lang, ev, want)
		}
	}
}

func TestShellSetLanguage(t *testing.T) {
	s := &Shell{}
	if s.language() != "" {
		t.Fatalf("default: %q", s.language())
	}
	s.SetLanguage("en")
	if s.language() != "en" {
		t.Fatalf("en: %q", s.language())
	}
	s.SetLanguage("fr")
	if s.language() != "pt-BR" {
		t.Fatalf("unknown must fall back to pt-BR: %q", s.language())
	}
}
