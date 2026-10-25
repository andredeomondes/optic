package wol

import (
	"bytes"
	"testing"
)

func TestMagicPacket(t *testing.T) {
	p, err := MagicPacket("AA-BB-CC-DD-EE-FF")
	if err != nil || len(p) != 102 || !bytes.Equal(p[:6], bytes.Repeat([]byte{0xff}, 6)) || !bytes.Equal(p[6:12], []byte{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}) {
		t.Fatalf("p=%x err=%v", p, err)
	}
	if _, err := MagicPacket("zz:11"); err == nil {
		t.Fatal("invalid mac accepted")
	}
}

func TestTargets(t *testing.T) {
	got := Targets("192.168.0.18")
	if len(got) != 2 || got[0] != "255.255.255.255:9" || got[1] != "192.168.0.255:9" {
		t.Fatalf("got %v", got)
	}
	if got := Targets("not-an-ip"); len(got) != 1 || got[0] != "255.255.255.255:9" {
		t.Fatalf("invalid ip: got %v", got)
	}
}
