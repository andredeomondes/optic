package hotkey

import (
	"errors"
	"fmt"
	"strings"
)

// Accel is a key combination such as Ctrl+Alt+Up.
type Accel struct {
	Ctrl, Alt, Shift, Win bool
	Key                   string // canonical name, e.g. "Up", "F5", "MediaNext"
}

var (
	errNeedModifier = errors.New("Use Ctrl ou Alt junto com a tecla")
	errWinKey       = errors.New("A tecla Windows é reservada pelo sistema")
	errReserved     = errors.New("Atalho reservado pelo sistema")
)

// Windows modifier flags for RegisterHotKey.
const (
	modAlt      = 0x0001
	modControl  = 0x0002
	modShift    = 0x0004
	modNoRepeat = 0x4000
)

// keys maps canonical key names to Windows virtual-key codes.
var keys = map[string]uint32{
	"Up": 0x26, "Down": 0x28, "Left": 0x25, "Right": 0x27,
	"Home": 0x24, "End": 0x23, "PageUp": 0x21, "PageDown": 0x22, "Space": 0x20,
	"MediaPlayPause": 0xB3, "MediaNext": 0xB0, "MediaPrev": 0xB1, "MediaStop": 0xB2,
	"VolumeUp": 0xAF, "VolumeDown": 0xAE, "VolumeMute": 0xAD,
}

// media keys work alone; everything else needs Ctrl or Alt.
var media = map[string]bool{
	"MediaPlayPause": true, "MediaNext": true, "MediaPrev": true, "MediaStop": true,
	"VolumeUp": true, "VolumeDown": true, "VolumeMute": true,
}

// systemOnly keys are recognized only to name the reserved combos below.
var systemOnly = map[string]bool{"Tab": true, "Escape": true, "Delete": true}

var reserved = map[Accel]bool{
	{Alt: true, Key: "Tab"}:                  true,
	{Alt: true, Key: "F4"}:                   true,
	{Ctrl: true, Alt: true, Key: "Delete"}:   true,
	{Ctrl: true, Shift: true, Key: "Escape"}: true,
	{Alt: true, Key: "Escape"}:               true,
	{Alt: true, Key: "Space"}:                true,
}

// byLower finds a canonical key name case-insensitively.
var byLower = map[string]string{}

func init() {
	for c := 'A'; c <= 'Z'; c++ {
		keys[string(c)] = uint32(c)
	}
	for c := '0'; c <= '9'; c++ {
		keys[string(c)] = uint32(c)
	}
	for n := 1; n <= 12; n++ {
		keys[fmt.Sprintf("F%d", n)] = 0x70 + uint32(n-1)
	}
	for k := range keys {
		byLower[strings.ToLower(k)] = k
	}
	for k := range systemOnly {
		byLower[strings.ToLower(k)] = k
	}
}

func unknownKey(k string) error { return fmt.Errorf("Tecla desconhecida: %s", k) }

// Parse reads a "+"-separated accelerator like "ctrl+alt+up" (case-insensitive).
func Parse(s string) (Accel, error) {
	var a Accel
	for _, part := range strings.Split(s, "+") {
		tok := strings.TrimSpace(part)
		switch strings.ToLower(tok) {
		case "ctrl", "control":
			a.Ctrl = true
		case "alt":
			a.Alt = true
		case "shift":
			a.Shift = true
		case "win", "super", "meta":
			a.Win = true
		default:
			name, ok := byLower[strings.ToLower(tok)]
			if !ok || a.Key != "" {
				return Accel{}, unknownKey(tok)
			}
			a.Key = name
		}
	}
	if a.Key == "" {
		return Accel{}, unknownKey("")
	}
	if err := a.validate(); err != nil {
		return Accel{}, err
	}
	return a, nil
}

// validate applies the registration rules to an already-built Accel.
func (a Accel) validate() error {
	if !systemOnly[a.Key] {
		if _, ok := keys[a.Key]; !ok {
			return unknownKey(a.Key)
		}
	}
	if a.Win {
		return errWinKey
	}
	if reserved[a] {
		return errReserved
	}
	if systemOnly[a.Key] {
		return unknownKey(a.Key)
	}
	if !media[a.Key] && !a.Ctrl && !a.Alt {
		return errNeedModifier
	}
	return nil
}

// String returns the canonical form, e.g. "Ctrl+Alt+Shift+Up".
func (a Accel) String() string {
	var b strings.Builder
	for _, m := range []struct {
		on   bool
		name string
	}{{a.Ctrl, "Ctrl"}, {a.Alt, "Alt"}, {a.Shift, "Shift"}, {a.Win, "Win"}} {
		if m.on {
			b.WriteString(m.name)
			b.WriteByte('+')
		}
	}
	b.WriteString(a.Key)
	return b.String()
}

func vkOf(key string) (uint32, bool) {
	vk, ok := keys[key]
	return vk, ok
}

// modsOf returns RegisterHotKey modifiers; MOD_NOREPEAT is always set.
func modsOf(a Accel) uint32 {
	m := uint32(modNoRepeat)
	if a.Alt {
		m |= modAlt
	}
	if a.Ctrl {
		m |= modControl
	}
	if a.Shift {
		m |= modShift
	}
	return m
}
