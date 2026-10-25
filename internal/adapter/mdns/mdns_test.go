package mdns

import (
	"net"
	"testing"

	"optic/internal/domain"
	"optic/pkg/discovery"
)

func TestToTVsMarksRemoteV2AsAndroidTV(t *testing.T) {
	devices := []*discovery.Device{
		{
			Name:     "UnionTV",
			Model:    "TCL",
			IPs:      []net.IP{net.ParseIP("192.168.0.18").To4()},
			Services: map[string]int{discovery.ServiceRemoteV2: 9009},
		},
		{
			Name:     "Cast only",
			IPs:      []net.IP{net.ParseIP("192.168.0.20").To4()},
			Services: map[string]int{discovery.ServiceCast: 8009},
		},
	}

	got := ToTVs(devices)
	if len(got) != 2 {
		t.Fatalf("got %d TVs, want 2", len(got))
	}
	want := domain.TV{Name: "UnionTV", Model: "TCL", IP: "192.168.0.18", Remotev2: true, Platform: domain.PlatformAndroidTV}
	if got[0] != want {
		t.Errorf("got[0] = %+v, want %+v", got[0], want)
	}
	if got[1].Platform != domain.PlatformUnknown || got[1].Remotev2 || got[1].IP != "192.168.0.20" {
		t.Errorf("got[1] = %+v, want unknown platform without remotev2", got[1])
	}
}
