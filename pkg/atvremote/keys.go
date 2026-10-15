package atvremote

import (
	"sort"
	"strings"
)

// Keys maps friendly names to Android KeyEvent codes.
var Keys = map[string]int{
	"HOME":               3,
	"BACK":               4,
	"DPAD_UP":            19,
	"DPAD_DOWN":          20,
	"DPAD_LEFT":          21,
	"DPAD_RIGHT":         22,
	"DPAD_CENTER":        23,
	"VOLUME_UP":          24,
	"VOLUME_DOWN":        25,
	"POWER":              26,
	"ENTER":              66,
	"DEL":                67,
	"MENU":               82,
	"SEARCH":             84,
	"MEDIA_PLAY_PAUSE":   85,
	"MEDIA_STOP":         86,
	"MEDIA_NEXT":         87,
	"MEDIA_PREVIOUS":     88,
	"MEDIA_REWIND":       89,
	"MEDIA_FAST_FORWARD": 90,
	"MUTE":               164,
	"CHANNEL_UP":         166,
	"CHANNEL_DOWN":       167,
	"GUIDE":              172,
	"SETTINGS":           176,
	"TV_INPUT":           178,
}

// LookupKey accepts "HOME", "home" or "KEYCODE_HOME".
func LookupKey(name string) (int, bool) {
	n := strings.TrimPrefix(strings.ToUpper(strings.TrimSpace(name)), "KEYCODE_")
	code, ok := Keys[n]
	return code, ok
}

// KeyNames returns all supported key names, sorted.
func KeyNames() []string {
	out := make([]string, 0, len(Keys))
	for k := range Keys {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Apps maps shortcut names to launch URIs for common Brazilian streaming apps.
var Apps = map[string]string{
	"youtube":   "market://launch?id=com.google.android.youtube.tv",
	"netflix":   "market://launch?id=com.netflix.ninja",
	"prime":     "market://launch?id=com.amazon.amazonvideo.livingroom",
	"globoplay": "market://launch?id=com.globo.globotv",
}
