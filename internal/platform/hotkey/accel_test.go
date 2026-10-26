package hotkey

import "testing"

func TestParseValid(t *testing.T) {
	cases := []struct {
		in   string
		want Accel
	}{
		{"Ctrl+Alt+Up", Accel{Ctrl: true, Alt: true, Key: "Up"}},
		{"ctrl+alt+up", Accel{Ctrl: true, Alt: true, Key: "Up"}},
		{" Control + Shift + pagedown ", Accel{Ctrl: true, Shift: true, Key: "PageDown"}},
		{"Alt+F12", Accel{Alt: true, Key: "F12"}},
		{"Ctrl+a", Accel{Ctrl: true, Key: "A"}},
		{"Alt+0", Accel{Alt: true, Key: "0"}},
		{"Shift+Alt+Ctrl+Home", Accel{Ctrl: true, Alt: true, Shift: true, Key: "Home"}},
		{"Ctrl+Space", Accel{Ctrl: true, Key: "Space"}},
		{"MediaPlayPause", Accel{Key: "MediaPlayPause"}},
		{"medianext", Accel{Key: "MediaNext"}},
		{"MediaPrev", Accel{Key: "MediaPrev"}},
		{"MediaStop", Accel{Key: "MediaStop"}},
		{"VolumeUp", Accel{Key: "VolumeUp"}},
		{"Shift+VolumeDown", Accel{Shift: true, Key: "VolumeDown"}},
		{"Ctrl+VolumeMute", Accel{Ctrl: true, Key: "VolumeMute"}},
	}
	for _, c := range cases {
		got, err := Parse(c.in)
		if err != nil {
			t.Errorf("Parse(%q) error: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("Parse(%q) = %+v, want %+v", c.in, got, c.want)
		}
	}
}

func TestParseErrors(t *testing.T) {
	const reserved = "Atalho reservado pelo sistema"
	const needMod = "Use Ctrl ou Alt junto com a tecla"
	const win = "A tecla Windows é reservada pelo sistema"
	cases := []struct{ in, want string }{
		{"", "Tecla desconhecida: "},
		{"Ctrl+", "Tecla desconhecida: "},
		{"Ctrl+Alt+Foo", "Tecla desconhecida: Foo"},
		{"Ctrl+A+B", "Tecla desconhecida: B"},
		{"Ctrl+F13", "Tecla desconhecida: F13"},
		{"Ctrl+Tab", "Tecla desconhecida: Tab"},
		{"Up", needMod},
		{"Shift+Up", needMod},
		{"A", needMod},
		{"Win+Up", win},
		{"Super+Ctrl+Up", win},
		{"Meta+MediaNext", win},
		{"Alt+Tab", reserved},
		{"alt+tab", reserved},
		{"Alt+F4", reserved},
		{"Ctrl+Alt+Delete", reserved},
		{"Ctrl+Shift+Escape", reserved},
		{"Alt+Escape", reserved},
		{"Alt+Space", reserved},
	}
	for _, c := range cases {
		_, err := Parse(c.in)
		if err == nil {
			t.Errorf("Parse(%q) = nil error, want %q", c.in, c.want)
			continue
		}
		if err.Error() != c.want {
			t.Errorf("Parse(%q) error = %q, want %q", c.in, err.Error(), c.want)
		}
	}
}

func TestStringRoundTrip(t *testing.T) {
	ins := []string{
		"shift+alt+ctrl+up", "Ctrl+Z", "Alt+9", "Ctrl+Shift+F1", "Alt+PageUp",
		"Ctrl+Alt+Right", "MediaPlayPause", "Shift+VolumeUp", "Ctrl+End",
	}
	for _, in := range ins {
		a, err := Parse(in)
		if err != nil {
			t.Fatalf("Parse(%q): %v", in, err)
		}
		back, err := Parse(a.String())
		if err != nil || back != a {
			t.Errorf("round trip %q -> %q -> %+v (%v)", in, a.String(), back, err)
		}
	}
	if s := (Accel{Shift: true, Alt: true, Ctrl: true, Key: "Up"}).String(); s != "Ctrl+Alt+Shift+Up" {
		t.Errorf("String() = %q", s)
	}
}

func TestVirtualKeys(t *testing.T) {
	cases := map[string]uint32{
		"A": 'A', "Z": 'Z', "0": '0', "9": '9', "F1": 0x70, "F12": 0x7B,
		"Up": 0x26, "Down": 0x28, "Left": 0x25, "Right": 0x27,
		"Home": 0x24, "End": 0x23, "PageUp": 0x21, "PageDown": 0x22, "Space": 0x20,
		"MediaPlayPause": 0xB3, "MediaNext": 0xB0, "MediaPrev": 0xB1, "MediaStop": 0xB2,
		"VolumeUp": 0xAF, "VolumeDown": 0xAE, "VolumeMute": 0xAD,
	}
	for k, want := range cases {
		got, ok := vkOf(k)
		if !ok || got != want {
			t.Errorf("vkOf(%q) = %#x,%v want %#x", k, got, ok, want)
		}
	}
	if got := modsOf(Accel{Ctrl: true, Alt: true, Shift: true}); got != 0x4007 {
		t.Errorf("modsOf = %#x", got)
	}
	if got := modsOf(Accel{}); got != 0x4000 {
		t.Errorf("modsOf(none) = %#x", got)
	}
}
