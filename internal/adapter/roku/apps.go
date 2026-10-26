package roku

import (
	"errors"
	"net/url"
	"strings"

	"optic/pkg/roku"
)

// ErrAppNotInstalled means the requested app is not on the Roku.
var ErrAppNotInstalled = errors.New("Este app não está instalado na TV Roku.")

// ErrLinkUnsupported means a web link was given to a Roku, which cannot open them.
var ErrLinkUnsupported = errors.New("A Roku não abre links; abra o app pela lista.")

type knownApp struct{ ID, Name string }

// knownApps maps the Android packages OPTIC's catalog uses to their Roku
// channels. An empty ID means the channel is matched by name only.
var knownApps = map[string]knownApp{
	"com.google.android.youtube.tv":      {"837", "YouTube"},
	"com.netflix.ninja":                  {"12", "Netflix"},
	"com.amazon.amazonvideo.livingroom":  {"13", "Prime Video"},
	"com.disney.disneyplus":              {"291097", "Disney+"},
	"com.globo.globotv":                  {"", "Globoplay"},
	"com.spotify.tv.android":             {"22297", "Spotify"},
	"com.wbd.stream":                     {"61322", "Max"},
	"com.cbs.ott":                        {"31440", "Paramount+"},
	"com.apple.atve.androidtv.appletv":   {"551012", "Apple TV"},
	"com.plexapp.android":                {"13535", "Plex"},
	"com.crunchyroll.crunchyroid":        {"2595", "Crunchyroll"},
	"com.google.android.youtube.tvkids":  {"", "YouTube Kids"},
	"com.google.android.youtube.tvmusic": {"", "YouTube Music"},
	"deezer.android.tv":                  {"", "Deezer"},
}

// resolveApp turns a digit ID, a market://launch?id=<pkg> link or an app name
// into the ID of an installed channel.
func resolveApp(target string, installed []roku.App) (string, error) {
	target = strings.TrimSpace(target)
	if isDigits(target) {
		for _, a := range installed {
			if a.ID == target {
				return a.ID, nil
			}
		}
		return "", ErrAppNotInstalled
	}
	if strings.HasPrefix(strings.ToLower(target), "market://") {
		return resolveMarket(target, installed)
	}
	if lt := strings.ToLower(target); strings.HasPrefix(lt, "http://") || strings.HasPrefix(lt, "https://") {
		if u, err := url.Parse(target); err == nil && strings.Contains(strings.ToLower(u.Hostname()), "youtube") {
			return resolveApp("YouTube", installed)
		}
		return "", ErrLinkUnsupported
	}
	if target == "" {
		return "", ErrAppNotInstalled
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
	lt := strings.ToLower(target)
	for _, a := range installed {
		if strings.HasPrefix(strings.ToLower(a.Name), lt) {
			return a.ID, nil
		}
	}
	return "", ErrAppNotInstalled
}

func resolveMarket(target string, installed []roku.App) (string, error) {
	u, err := url.Parse(target)
	if err != nil {
		return "", ErrAppNotInstalled
	}
	k, ok := knownApps[u.Query().Get("id")]
	if !ok {
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

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
