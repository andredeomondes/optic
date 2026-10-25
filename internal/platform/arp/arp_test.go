package arp

import "testing"

func TestParseTable(t *testing.T) {
	win := "Interface: 192.168.0.10 --- 0x7\n  Endereço IP    Endereço físico    Tipo\n  192.168.0.1    11-22-33-44-55-66  dinâmico\n  192.168.0.18   aa-bb-cc-dd-ee-ff  dinâmico\n  192.168.0.255  ff-ff-ff-ff-ff-ff  estático\n"
	mac := "? (192.168.0.18) at aa:bb:cc:dd:ee:ff on en0 ifscope [ethernet]\n? (192.168.0.50) at (incomplete) on en0\n? (192.168.0.255) at ff:ff:ff:ff:ff:ff on en0\n"
	linux := "IP address       HW type     Flags       HW address            Mask     Device\n192.168.0.18     0x1         0x2         aa:bb:cc:dd:ee:ff     *        wlan0\n192.168.0.50     0x1         0x0         00:00:00:00:00:00     *        wlan0\n"
	for name, out := range map[string]string{"windows": win, "macos": mac, "linux": linux} {
		tbl := ParseTable(out)
		if tbl["192.168.0.18"] != "aa:bb:cc:dd:ee:ff" {
			t.Errorf("%s: got %v", name, tbl)
		}
		if _, ok := tbl["192.168.0.255"]; ok {
			t.Errorf("%s: broadcast kept", name)
		}
		if _, ok := tbl["192.168.0.50"]; ok {
			t.Errorf("%s: incomplete kept", name)
		}
	}
}

func TestParseTableNormalizesCase(t *testing.T) {
	tbl := ParseTable("  192.168.0.18   AA-BB-CC-DD-EE-FF   dynamic\n")
	if tbl["192.168.0.18"] != "aa:bb:cc:dd:ee:ff" {
		t.Fatalf("got %v", tbl)
	}
}
