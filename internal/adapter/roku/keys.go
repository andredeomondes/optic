package roku

import "strings"

// rokuKeys maps OPTIC's (Android-style) key names to ECP key names.
var rokuKeys = map[string]string{
	"HOME": "Home", "BACK": "Back",
	"DPAD_UP": "Up", "DPAD_DOWN": "Down", "DPAD_LEFT": "Left", "DPAD_RIGHT": "Right",
	"DPAD_CENTER": "Select", "ENTER": "Enter", "DEL": "Backspace", "SEARCH": "Search",
	"VOLUME_UP": "VolumeUp", "VOLUME_DOWN": "VolumeDown", "MUTE": "VolumeMute",
	"POWER":            "Power",
	"MEDIA_PLAY_PAUSE": "Play", "MEDIA_REWIND": "Rev", "MEDIA_FAST_FORWARD": "Fwd",
	"MEDIA_PREVIOUS": "InstantReplay",
	"MENU":           "Info", "SETTINGS": "Info",
	"CHANNEL_UP": "ChannelUp", "CHANNEL_DOWN": "ChannelDown",
}

// rokuKey translates a key name, ignoring case and a KEYCODE_ prefix.
func rokuKey(name string) (string, bool) {
	n := strings.ToUpper(strings.TrimSpace(name))
	n = strings.TrimPrefix(n, "KEYCODE_")
	k, ok := rokuKeys[n]
	return k, ok
}
