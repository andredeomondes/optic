package roku

import (
	"testing"

	"optic/pkg/roku"
)

func TestResolveAppExactIDBeforeName(t *testing.T) {
	installed := []roku.App{
		{ID: "tvinput.hdmi1", Name: "HDMI 1", Type: "tvin"},
		{ID: "abc.def", Name: "Outro"},
	}
	got, err := resolveApp("abc.def", installed)
	if err != nil || got != "abc.def" {
		t.Fatalf("got %q, err %v", got, err)
	}
	got, err = resolveApp("tvinput.hdmi1", installed)
	if err != nil || got != "tvinput.hdmi1" {
		t.Fatalf("got %q, err %v", got, err)
	}
}
