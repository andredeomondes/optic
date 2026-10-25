package main

import (
	"bytes"
	"strings"
	"testing"

	"optic/internal/domain"
	"optic/pkg/discovery"
)

func TestPrintTVsShowsPlatform(t *testing.T) {
	var buf bytes.Buffer
	printTVs(&buf, []domain.TV{
		{Name: "UnionTV", Model: "TCL 55", IP: "192.168.0.18", Platform: domain.PlatformAndroidTV},
		{Name: "Sala", IP: "192.168.0.20", Platform: domain.PlatformRoku},
	}, nil)

	out := buf.String()
	for _, want := range []string{
		"▸ UnionTV  [Android TV]  (TCL 55)",
		"ip: 192.168.0.18",
		"▸ Sala  [Roku]",
		"ip: 192.168.0.20",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestPrintTVsVerboseAddsMDNSDetails(t *testing.T) {
	var buf bytes.Buffer
	details := map[string]*discovery.Device{
		"192.168.0.18": {
			Services: map[string]int{discovery.ServiceRemoteV2: 9009},
			TXT:      map[string]string{"md": "UnionTV"},
		},
	}
	printTVs(&buf, []domain.TV{
		{Name: "UnionTV", IP: "192.168.0.18", Platform: domain.PlatformAndroidTV},
		{Name: "Sala", IP: "192.168.0.20", Platform: domain.PlatformRoku},
	}, details)

	out := buf.String()
	for _, want := range []string{"txt md=UnionTV", discovery.ServiceRemoteV2} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	if strings.Count(out, "txt md=") != 1 {
		t.Errorf("TXT details should only appear for the TV with mDNS details:\n%s", out)
	}
}
