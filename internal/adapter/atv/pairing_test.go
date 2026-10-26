package atv

import (
	"testing"

	"optic/internal/domain"
)

func TestPairingModeCode(t *testing.T) {
	var p domain.PairingSession = pairing{}
	if p.Mode() != domain.PairingCode {
		t.Errorf("Mode() = %q", p.Mode())
	}
}
