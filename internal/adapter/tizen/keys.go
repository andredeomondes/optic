package tizen

import "strings"

// samsungKeys maps OPTIC's (Android-style) key names to Samsung remote keys.
// MEDIA_PLAY_PAUSE is handled apart: it alternates KEY_PAUSE and KEY_PLAY.
var samsungKeys = map[string]string{
	"HOME": "KEY_HOME", "BACK": "KEY_RETURN",
	"DPAD_UP": "KEY_UP", "DPAD_DOWN": "KEY_DOWN", "DPAD_LEFT": "KEY_LEFT", "DPAD_RIGHT": "KEY_RIGHT",
	"DPAD_CENTER": "KEY_ENTER", "ENTER": "KEY_ENTER",
	"VOLUME_UP": "KEY_VOLUP", "VOLUME_DOWN": "KEY_VOLDOWN", "MUTE": "KEY_MUTE",
	"POWER":      "KEY_POWER",
	"CHANNEL_UP": "KEY_CHUP", "CHANNEL_DOWN": "KEY_CHDOWN",
	"MEDIA_STOP": "KEY_STOP", "MEDIA_REWIND": "KEY_REWIND", "MEDIA_FAST_FORWARD": "KEY_FF",
	"MENU": "KEY_MENU", "SETTINGS": "KEY_MENU", "GUIDE": "KEY_GUIDE",
	"TV_INPUT": "KEY_SOURCE",
}

// keyName normalizes a key name: upper case, no KEYCODE_ prefix.
func keyName(name string) string {
	return strings.TrimPrefix(strings.ToUpper(strings.TrimSpace(name)), "KEYCODE_")
}
