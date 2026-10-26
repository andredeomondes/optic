package webos

import (
	"errors"
	"net/url"
	"strings"

	"optic/internal/domain"
)

// ErrAppNotInstalled means the requested app is not on the LG TV.
var ErrAppNotInstalled = errors.New("Este app não está instalado na TV LG.")

type knownApp struct{ ID, Name string }

// knownApps maps the Android packages OPTIC's catalog uses to webOS app IDs.
// An empty ID means the app is matched by name only.
var knownApps = map[string]knownApp{
	"com.google.android.youtube.tv":      {"youtube.leanback.v4", "YouTube"},
	"com.netflix.ninja":                  {"netflix", "Netflix"},
	"com.amazon.amazonvideo.livingroom":  {"amazon", "Prime Video"},
	"com.disney.disneyplus":              {"com.disney.disneyplus-prod", "Disney+"},
	"com.globo.globotv":                  {"", "Globoplay"},
	"com.spotify.tv.android":             {"spotify-beehive", "Spotify"},
	"com.wbd.stream":                     {"", "Max"},
	"tv.twitch.android.app":              {"", "Twitch"},
	"com.crunchyroll.crunchyroid":        {"", "Crunchyroll"},
	"com.plexapp.android":                {"", "Plex"},
	"org.xbmc.kodi":                      {"", "Kodi"},
	"org.videolan.vlc":                   {"", "VLC"},
	"com.android.vending":                {"", "Play Store"},
	"com.cbs.ott":                        {"", "Paramount+"},
	"com.apple.atve.androidtv.appletv":   {"", "Apple TV"},
	"deezer.android.tv":                  {"", "Deezer"},
	"com.google.android.youtube.tvmusic": {"", "YouTube Music"},
	"com.google.android.youtube.tvkids":  {"", "YouTube Kids"},
	"com.valvesoftware.steamlink":        {"", "Steam Link"},
}

// isLink reports whether target is a web link, opened in the TV browser.
func isLink(target string) bool {
	lt := strings.ToLower(strings.TrimSpace(target))
	return strings.HasPrefix(lt, "http://") || strings.HasPrefix(lt, "https://")
}

// resolveApp turns a market://launch?id=<pkg> link, an app ID or an app name
// into the ID of an installed app.
func resolveApp(target string, installed []domain.App) (string, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return "", ErrAppNotInstalled
	}
	if strings.HasPrefix(strings.ToLower(target), "market://") {
		return resolveMarket(target, installed)
	}
	for _, a := range installed {
		if a.ID == target {
			return a.ID, nil
		}
	}
	for _, a := range installed {
		if strings.EqualFold(a.Name, target) {
			return a.ID, nil
		}
	}
	// A prefix can match several apps: the first in the TV's order wins.
	lt := strings.ToLower(target)
	for _, a := range installed {
		if strings.HasPrefix(strings.ToLower(a.Name), lt) {
			return a.ID, nil
		}
	}
	return "", ErrAppNotInstalled
}

func resolveMarket(target string, installed []domain.App) (string, error) {
	u, err := url.Parse(target)
	if err != nil {
		return "", ErrAppNotInstalled
	}
	pkg := u.Query().Get("id")
	k, ok := knownApps[pkg]
	if !ok {
		// Not in the catalog: it may be a webOS app ID (com.webos.app.browser).
		for _, a := range installed {
			if pkg != "" && a.ID == pkg {
				return a.ID, nil
			}
		}
		return "", ErrAppNotInstalled
	}
	for _, a := range installed {
		if strings.EqualFold(a.Name, k.Name) {
			return a.ID, nil
		}
	}
	if k.ID != "" {
		for _, a := range installed {
			if a.ID == k.ID {
				return a.ID, nil
			}
		}
	}
	return "", ErrAppNotInstalled
}
