package atvremote

import (
	"errors"
	"strings"
	"time"
	"unicode"

	"google.golang.org/protobuf/encoding/protowire"
)

// Text input. When a text field on the TV has the on-screen keyboard open, the
// TV announces IME counters; we then replace the field's content in one
// RemoteImeBatchEdit. Without counters we fall back to typing key by key.
//
//	message RemoteImeBatchEdit  { int32 ime_counter = 1; int32 field_counter = 2; repeated RemoteEditInfo edit_info = 3; }
//	message RemoteEditInfo      { int32 insert = 1; RemoteImeObject text_field_status = 2; }
//	message RemoteImeObject     { int32 start = 1; int32 end = 2; string value = 3; }
//	message RemoteImeShowRequest { RemoteTextFieldStatus remote_text_field_status = 2; }
//	message RemoteTextFieldStatus { int32 counter_field = 1; string value = 2; ... }
const (
	rmImeKeyInject protowire.Number = 20
	rmImeBatchEdit protowire.Number = 21
	rmImeShow      protowire.Number = 22

	keyTypingGap = 30 * time.Millisecond
)

type imeState struct {
	known        bool
	imeCounter   uint64
	fieldCounter uint64
}

// handleIme tracks the foreground app and IME counters.
//
// Observed on a TCL Google TV (Android 9): on connect the TV sends
// RemoteImeKeyInject (20) with only app_info. When a system keyboard opens
// it sends 20 again, now with text_field_status, followed by
// RemoteImeBatchEdit (21) carrying the counters. Apps with their own
// on-screen keyboard (Netflix) send nothing.
func (r *Remote) handleIme(f fields) {
	if b, ok := f.bytes(rmImeKeyInject); ok {
		kf, err := parseFields(b)
		if err == nil {
			var pkg string
			if info, ok := kf.bytes(1); ok {
				af, _ := parseFields(info)
				if p, ok := af.bytes(12); ok {
					pkg = string(p)
				}
			}
			_, field := kf[2]
			if !field {
				// Focus moved without a text field: counters are stale.
				r.mu.Lock()
				r.ime = imeState{}
				r.mu.Unlock()
			}
			r.updateState(func(s *State) {
				if pkg != "" {
					s.App = pkg
				}
				s.Keyboard = field
			})
		}
	}
	if b, ok := f.bytes(rmImeBatchEdit); ok {
		bf, err := parseFields(b)
		if err != nil {
			return
		}
		ime, _ := bf.uint(1)
		field, _ := bf.uint(2)
		r.mu.Lock()
		r.ime = imeState{known: true, imeCounter: ime, fieldCounter: field}
		r.mu.Unlock()
	}
	if b, ok := f.bytes(rmImeShow); ok {
		sf, err := parseFields(b)
		if err != nil {
			return
		}
		status, ok := sf.bytes(2)
		if !ok {
			return
		}
		tf, err := parseFields(status)
		if err != nil {
			return
		}
		field, _ := tf.uint(1)
		r.mu.Lock()
		r.ime.known = true
		r.ime.fieldCounter = field
		r.mu.Unlock()
	}
}

// IsTextFrame reports whether a control frame carries text the user typed or a
// text field's content: ImeBatchEdit (21), ImeShow (22), or ImeKeyInject (20)
// with a text_field_status. Tracers must not log these.
func IsTextFrame(msg []byte) bool {
	f, err := parseFields(msg)
	if err != nil {
		return false
	}
	if _, ok := f[rmImeBatchEdit]; ok {
		return true
	}
	if _, ok := f[rmImeShow]; ok {
		return true
	}
	if b, ok := f.bytes(rmImeKeyInject); ok {
		kf, err := parseFields(b)
		if err != nil {
			// Cannot tell what it holds; fail closed.
			return true
		}
		_, field := kf[2]
		return field
	}
	return false
}

// SendText types text into the focused field on the TV.
// With the keyboard open, the field's content is replaced by text.
func (r *Remote) SendText(text string) error {
	r.mu.Lock()
	ime := r.ime
	r.mu.Unlock()

	if ime.known {
		return r.send(rmImeBatchEdit, imeBatchEdit(ime, text))
	}
	return r.typeKeys(text)
}

func imeBatchEdit(ime imeState, text string) []byte {
	cursor := uint64(len([]rune(text)))
	if cursor > 0 {
		cursor--
	}
	var obj []byte
	obj = appendUint(obj, 1, cursor)
	obj = appendUint(obj, 2, cursor)
	obj = appendString(obj, 3, text)

	var edit []byte
	edit = appendUint(edit, 1, 1)
	edit = appendMsg(edit, 2, obj)

	var p []byte
	p = appendUint(p, 1, ime.imeCounter)
	p = appendUint(p, 2, ime.fieldCounter)
	p = appendMsg(p, 3, edit)
	return p
}

// typeKeys sends one keycode per character; unsupported characters are skipped.
func (r *Remote) typeKeys(text string) error {
	for _, ch := range strings.ToLower(text) {
		code, ok := charKey(ch)
		if !ok {
			continue
		}
		if err := r.SendKey(code); err != nil {
			return err
		}
		time.Sleep(keyTypingGap)
	}
	return nil
}

var punctKeys = map[rune]int{
	' ': 62, '\n': 66, '\t': 61,
	',': 55, '.': 56, '`': 68, '-': 69, '=': 70, '[': 71, ']': 72,
	'\\': 73, ';': 74, '\'': 75, '/': 76, '@': 77, '+': 81, '*': 17, '#': 18,
}

// charKey maps a lowercase character to its Android keycode.
func charKey(ch rune) (int, bool) {
	switch {
	case ch >= 'a' && ch <= 'z':
		return 29 + int(ch-'a'), true
	case ch >= '0' && ch <= '9':
		return 7 + int(ch-'0'), true
	case unicode.IsSpace(ch) && ch != '\n' && ch != '\t':
		return 62, true
	}
	code, ok := punctKeys[ch]
	return code, ok
}

// ErrNoKeyboard means the TV opened search without the Android keyboard
// (e.g. straight into voice search), so there is no field to type into.
var ErrNoKeyboard = errors.New("a busca da TV abriu sem teclado (talvez por voz)")

// WaitKeyboard blocks until the TV reports an open Android keyboard.
func (r *Remote) WaitKeyboard(timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		r.mu.Lock()
		ok := r.ime.known
		r.mu.Unlock()
		if ok {
			return true
		}
		select {
		case <-r.done:
			return false
		case <-time.After(100 * time.Millisecond):
		}
	}
	return false
}

// Search opens the TV's global search (which uses the Android keyboard even
// when the foreground app has its own), types query and submits it.
// Results from Google TV link into apps like Netflix.
func (r *Remote) Search(query string) error {
	// Forget any open field so we only trust counters for the search box.
	r.mu.Lock()
	r.ime = imeState{}
	r.mu.Unlock()

	if err := r.SendKey(Keys["SEARCH"]); err != nil {
		return err
	}
	if !r.WaitKeyboard(4 * time.Second) {
		return ErrNoKeyboard
	}
	if err := r.SendText(query); err != nil {
		return err
	}
	time.Sleep(300 * time.Millisecond)
	return r.SendKey(Keys["ENTER"])
}
