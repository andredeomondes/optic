package domain

import (
	"errors"
	"testing"
)

func TestPlatformLabel(t *testing.T) {
	cases := map[Platform]string{PlatformAndroidTV: "Android TV", PlatformTizen: "Samsung", PlatformWebOS: "LG webOS", PlatformRoku: "Roku", PlatformUnknown: "Desconhecida"}
	for p, want := range cases {
		if got := p.Label(); got != want {
			t.Errorf("%q.Label() = %q, want %q", p, got, want)
		}
	}
}

func TestErrUnsupported(t *testing.T) {
	err := error(ErrUnsupported{Platform: PlatformRoku})
	if err.Error() != "Roku ainda não é suportada" {
		t.Fatalf("got %q", err.Error())
	}
	var u ErrUnsupported
	if !errors.As(err, &u) || u.Platform != PlatformRoku {
		t.Fatal("errors.As failed")
	}
}
