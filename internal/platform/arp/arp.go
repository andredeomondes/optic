// Package arp reads the OS ARP table to learn a device's MAC from its IP.
package arp

import (
	"net"
	"net/netip"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
)

var (
	ipv4Re = regexp.MustCompile(`\b\d{1,3}(?:\.\d{1,3}){3}\b`)
	macRe  = regexp.MustCompile(`(?i)\b[0-9a-f]{2}(?:[:-][0-9a-f]{2}){5}\b`)
)

// ParseTable reads `arp -a` (Windows), `arp -an` (macOS) or /proc/net/arp
// (Linux) into IP → lowercase MAC with colons. It matches IPv4 and MAC tokens
// by pattern, never by header or type words, because Windows localizes them.
// Broadcast, all-zero and incomplete entries are dropped.
func ParseTable(out string) map[string]string {
	table := map[string]string{}
	for _, line := range strings.Split(out, "\n") {
		ip := ipv4Re.FindString(line)
		mac := macRe.FindString(line)
		if ip == "" || mac == "" {
			continue
		}
		if _, err := netip.ParseAddr(ip); err != nil {
			continue
		}
		hw, err := net.ParseMAC(mac)
		if err != nil || isPlaceholder(hw) {
			continue
		}
		table[ip] = strings.ToLower(hw.String())
	}
	return table
}

// isPlaceholder reports all-ones (broadcast) and all-zeros (incomplete) MACs.
func isPlaceholder(hw net.HardwareAddr) bool {
	first := hw[0]
	if first != 0x00 && first != 0xff {
		return false
	}
	for _, b := range hw {
		if b != first {
			return false
		}
	}
	return true
}

// LookupMAC finds ip in the OS ARP table. The TV must have talked to this
// machine recently, which is true right after connecting to it.
func LookupMAC(ip string) (string, bool) {
	out, err := readTable()
	if err != nil {
		return "", false
	}
	mac, ok := ParseTable(out)[ip]
	return mac, ok
}

func readTable() (string, error) {
	switch runtime.GOOS {
	case "linux":
		b, err := os.ReadFile("/proc/net/arp")
		return string(b), err
	case "darwin":
		return run("arp", "-an")
	default:
		return run("arp", "-a")
	}
}

func run(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	hideWindow(cmd)
	b, err := cmd.Output()
	return string(b), err
}

// Resolver is the concrete adapter for domain.MACResolver.
type Resolver struct{}

func (Resolver) LookupMAC(ip string) (string, bool) { return LookupMAC(ip) }
