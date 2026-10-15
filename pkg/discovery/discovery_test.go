package discovery

import (
	"net"
	"testing"
)

func TestDedupeJoinsEntriesSharingAnAddress(t *testing.T) {
	v6 := net.ParseIP("2804:14d:72c1:897b:e972:f103:81ab:84ae")
	devices := map[string]*Device{
		"192.168.0.18": {
			Name: "UnionTV", IPs: []net.IP{net.ParseIP("192.168.0.18"), v6},
			Services: map[string]int{ServiceRemoteV2: 6466}, TXT: map[string]string{},
		},
		v6.String(): {
			Name: "UnionTV", Model: "UnionTV", IPs: []net.IP{v6},
			Services: map[string]int{ServiceCast: 8009}, TXT: map[string]string{"fn": "UnionTV"},
		},
		"192.168.0.71": {
			Name: "Quarto", IPs: []net.IP{net.ParseIP("192.168.0.71")},
			Services: map[string]int{ServiceRemoteV2: 6466}, TXT: map[string]string{},
		},
	}

	out := dedupe(devices)
	if len(out) != 2 {
		t.Fatalf("got %d devices, want 2", len(out))
	}
	for _, d := range out {
		if d.Name != "UnionTV" {
			continue
		}
		if d.Addr() != "192.168.0.18" {
			t.Errorf("Addr = %s, want the IPv4", d.Addr())
		}
		if len(d.Services) != 2 || d.Model != "UnionTV" {
			t.Errorf("not merged: %+v", d)
		}
	}
}
