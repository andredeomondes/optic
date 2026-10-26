package tizen

import (
	"errors"
	"net/url"
	"strings"

	"optic/internal/domain"
)

// ErrAppNotInstalled means the requested app is not on the Samsung TV.
var ErrAppNotInstalled = errors.New("Este app não está instalado na TV Samsung.")

// ErrAppsUnavailable means the TV did not send its app list.
var ErrAppsUnavailable = errors.New("A TV Samsung não enviou a lista de apps. Tente de novo.")

type knownApp struct{ ID, Name string }

// knownApps maps the Android packages OPTIC's catalog uses to Samsung app IDs.
// The IDs are best-effort from public lists, so the installed title is tried
// first. An empty ID means the app is matched by name only.
var knownApps = map[string]knownApp{
	"com.google.android.youtube.tv":      {"111299001912", "YouTube"},
	"com.netflix.ninja":                  {"3201907018807", "Netflix"},
	"com.amazon.amazonvideo.livingroom":  {"3201910019365", "Prime Video"},
	"com.disney.disneyplus":              {"3201901017640", "Disney+"},
	"com.globo.globotv":                  {"3201908019041", "Globoplay"},
	"com.spotify.tv.android":             {"3201606009684", "Spotify"},
	"com.wbd.stream":                     {"3202301029760", "Max"},
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
		// Not in the catalog: it may be a Samsung app ID picked from the TV list.
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

// knownID returns the Samsung app ID of a market:// link in the catalog, or "".
func knownID(target string) string {
	target = strings.TrimSpace(target)
	if !strings.HasPrefix(strings.ToLower(target), "market://") {
		return ""
	}
	u, err := url.Parse(target)
	if err != nil {
		return ""
	}
	return knownApps[u.Query().Get("id")].ID
}
