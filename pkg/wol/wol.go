// Package wol wakes a machine with a Wake-on-LAN magic packet.
package wol

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"strings"
)

// MagicPacket builds 6×0xFF followed by the MAC 16 times (102 bytes).
// It accepts "aa:bb:..", "AA-BB-.." and the other forms net.ParseMAC knows.
func MagicPacket(mac string) ([]byte, error) {
	hw, err := net.ParseMAC(strings.TrimSpace(mac))
	if err != nil || len(hw) != 6 {
		return nil, fmt.Errorf("MAC inválido %q", mac)
	}
	return append(bytes.Repeat([]byte{0xff}, 6), bytes.Repeat(hw, 16)...), nil
}

// Targets are where the packet is sent: the global broadcast plus the TV's
// /24 broadcast (some routers drop the global one). The latter only for IPv4.
func Targets(tvIP string) []string {
	out := []string{"255.255.255.255:9"}
	if ip := net.ParseIP(tvIP).To4(); ip != nil {
		out = append(out, fmt.Sprintf("%d.%d.%d.255:9", ip[0], ip[1], ip[2]))
	}
	return out
}

// Send broadcasts the magic packet for mac. It succeeds when at least one
// target accepted it.
func Send(mac, tvIP string) error {
	pkt, err := MagicPacket(mac)
	if err != nil {
		return err
	}
	var errs []error
	sent := false
	for _, target := range Targets(tvIP) {
		if err := sendTo(target, pkt); err != nil {
			errs = append(errs, err)
			continue
		}
		sent = true
	}
	if !sent {
		return errors.Join(errs...)
	}
	return nil
}

func sendTo(target string, pkt []byte) error {
	conn, err := net.Dial("udp4", target)
	if err != nil {
		return err
	}
	defer conn.Close()
	_, err = conn.Write(pkt)
	return err
}

// Sender is the concrete adapter for domain.WakeSender.
type Sender struct{}

func (Sender) Send(mac, ip string) error { return Send(mac, ip) }
