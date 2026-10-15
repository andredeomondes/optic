package atv

import (
	"errors"
	"testing"
)

func TestResolveAppLink(t *testing.T) {
	cases := []struct {
		in, want string
		err      error
	}{
		{"youtube", "https://www.youtube.com", nil},
		{"Netflix", "https://www.netflix.com/browse", nil},
		{"market://launch?id=com.spotify.tv.android", "spotify://", nil},
		{"spotify://", "spotify://", nil},
		{"market://launch?id=com.amazon.amazonvideo.livingroom", "https://app.primevideo.com", nil},
		{"com.disney.disneyplus", "https://www.disneyplus.com", nil},
		{"https://www.youtube.com/watch?v=abc", "https://www.youtube.com/watch?v=abc", nil},
		{"twitch://", "", ErrAppNotLaunchable},
		{"market://launch?id=com.unknown.app", "", ErrAppNotLaunchable},
		{"intent:#Intent;package=com.x;end", "", ErrAppNotLaunchable},
		{"android-app://com.netflix.ninja", "", ErrAppNotLaunchable},
		{"not a link", "", ErrAppNotLaunchable},
	}
	for _, tc := range cases {
		got, err := resolveAppLink(tc.in)
		if got != tc.want || !errors.Is(err, tc.err) {
			t.Errorf("resolveAppLink(%q) = %q, %v; want %q, %v", tc.in, got, err, tc.want, tc.err)
		}
	}
}
