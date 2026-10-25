// Package ssdp discovers TVs that advertise via SSDP (Samsung, LG, Roku).
package ssdp

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"net/textproto"
	"strings"
	"time"
)

const multicastAddr = "239.255.255.250:1900"

// SearchTargets are the M-SEARCH targets used to find supported TVs.
var SearchTargets = []string{
	"roku:ecp",
	"urn:lge-com:service:webos-second-screen:1",
	"urn:samsung.com:device:RemoteControlReceiver:1",
}

// Response is one SSDP answer.
type Response struct {
	IP, ST, USN, Server, Location string
}

// Parse reads an "HTTP/1.1 200 OK" SSDP response; anything else is rejected.
func Parse(raw []byte, from net.IP) (Response, bool) {
	tp := textproto.NewReader(bufio.NewReader(bytes.NewReader(raw)))
	status, err := tp.ReadLine()
	if err != nil || !strings.HasPrefix(status, "HTTP/1.1 200") {
		return Response{}, false
	}
	h, _ := tp.ReadMIMEHeader() // partial headers are still useful
	r := Response{
		IP:       from.String(),
		ST:       h.Get("ST"),
		USN:      h.Get("USN"),
		Server:   h.Get("Server"),
		Location: h.Get("Location"),
	}
	if r.ST == "" {
		r.ST = h.Get("NT")
	}
	if r.ST == "" {
		return Response{}, false
	}
	return r, true
}

// Classify maps a response to a platform ("roku", "webos" or "tizen").
func Classify(r Response) (string, bool) {
	id := r.ST + " " + r.USN
	switch {
	case strings.Contains(id, "roku:ecp"):
		return "roku", true
	case strings.Contains(id, "urn:lge-com:service:webos-second-screen"):
		return "webos", true
	case strings.Contains(id, "urn:samsung.com:device:RemoteControlReceiver"),
		strings.Contains(r.Server, "Samsung") && strings.Contains(r.Server, "Tizen"):
		return "tizen", true
	}
	return "", false
}

// Search multicasts one M-SEARCH per target and collects answers until timeout
// or ctx is done. Results are deduplicated by IP+ST.
func Search(ctx context.Context, timeout time.Duration) ([]Response, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	conn, err := net.ListenUDP("udp4", nil)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	// Unblocks ReadFromUDP as soon as ctx ends.
	go func() {
		<-ctx.Done()
		conn.SetReadDeadline(time.Now())
	}()

	dst, err := net.ResolveUDPAddr("udp4", multicastAddr)
	if err != nil {
		return nil, err
	}
	for _, st := range SearchTargets {
		msg := fmt.Sprintf("M-SEARCH * HTTP/1.1\r\nHOST: %s\r\nMAN: \"ssdp:discover\"\r\nMX: 2\r\nST: %s\r\n\r\n", multicastAddr, st)
		if _, err := conn.WriteToUDP([]byte(msg), dst); err != nil {
			return nil, err
		}
	}

	var out []Response
	seen := map[string]bool{}
	buf := make([]byte, 4096)
	for {
		n, from, err := conn.ReadFromUDP(buf)
		if err != nil {
			if stop, ret := readErrAction(ctx, err); stop {
				return out, ret
			}
			continue // transient (e.g. connection reset): keep listening
		}
		r, ok := Parse(buf[:n], from.IP)
		if !ok || seen[r.IP+r.ST] {
			continue
		}
		seen[r.IP+r.ST] = true
		out = append(out, r)
	}
}

// readErrAction decides what the read loop does with a ReadFromUDP error:
// stop with ret, or keep reading. Deadline/cancellation end the search cleanly;
// a closed socket is fatal; anything else is treated as transient.
func readErrAction(ctx context.Context, err error) (stop bool, ret error) {
	var ne net.Error
	switch {
	case ctx.Err() != nil, errors.As(err, &ne) && ne.Timeout():
		return true, nil
	case errors.Is(err, net.ErrClosed):
		return true, err
	}
	return false, nil
}
