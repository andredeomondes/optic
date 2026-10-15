package atv

import (
	"errors"
	"net/url"
	"strings"
)

// ErrAppNotLaunchable means the TV cannot open this app through the remote
// protocol. We refuse it locally: the TV answers unsupported links with a
// RemoteError and then drops the whole control session.
var ErrAppNotLaunchable = errors.New("a TV não abre este app pelo controle remoto; abra pela tela inicial")

// webLinks maps Android packages to links the TV resolves to the app.
//
// Tested on a TCL Google TV (Android 9): "market://launch?id=", "intent:",
// "android-app://" and any scheme without an installed handler come back as
// RemoteError 7 and end the session; http(s) links are always accepted and
// open the app that claims them (or nothing). Verified to open the app:
// YouTube, Netflix, Prime Video, Disney+, Spotify. The others are the apps'
// public web links, accepted by the TV but not yet seen opening the app.
var webLinks = map[string]string{
	"com.google.android.youtube.tv":      "https://www.youtube.com",
	"com.google.android.youtube.tvmusic": "https://music.youtube.com",
	"com.netflix.ninja":                  "https://www.netflix.com/browse",
	"com.amazon.amazonvideo.livingroom":  "https://app.primevideo.com",
	"com.disney.disneyplus":              "https://www.disneyplus.com",
	"com.globo.globotv":                  "https://globoplay.globo.com",
	"com.spotify.tv.android":             "spotify://",
	"tv.twitch.android.app":              "https://www.twitch.tv",
	"com.crunchyroll.crunchyroid":        "https://www.crunchyroll.com",
	"com.google.android.youtube.tvkids":  "https://www.youtubekids.com",
	"com.hbo.hbonow":                     "https://play.max.com",
	"com.wbd.stream":                     "https://play.max.com",
	"com.apple.atve.androidtv.appletv":   "https://tv.apple.com",
	"com.cbs.ott":                        "https://www.paramountplus.com",
}

// shortcuts are the names the CLI and older callers use.
var shortcuts = map[string]string{
	"youtube":   "com.google.android.youtube.tv",
	"netflix":   "com.netflix.ninja",
	"prime":     "com.amazon.amazonvideo.livingroom",
	"globoplay": "com.globo.globotv",
	"disney":    "com.disney.disneyplus",
}

// resolveAppLink turns a shortcut, a package, a "market://launch?id=<pkg>"
// URI or an http(s) link into something the TV accepts, or
// ErrAppNotLaunchable. Other schemes are refused: an unknown one would make
// the TV drop the connection.
func resolveAppLink(nameOrURI string) (string, error) {
	s := strings.TrimSpace(nameOrURI)
	if pkg, ok := shortcuts[strings.ToLower(s)]; ok {
		s = pkg
	}
	if link, ok := webLinks[s]; ok { // bare package name
		return link, nil
	}
	u, err := url.Parse(s)
	if err != nil || u.Scheme == "" {
		return "", ErrAppNotLaunchable
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		return s, nil
	case "market":
		pkg := u.Query().Get("id")
		if link, ok := webLinks[pkg]; ok {
			return link, nil
		}
	}
	for _, link := range webLinks { // a known app scheme, e.g. "spotify://"
		if s == link {
			return s, nil
		}
	}
	return "", ErrAppNotLaunchable
}
