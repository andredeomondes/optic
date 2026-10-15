// Package discovery finds controllable TVs on the local network via mDNS.
package discovery

import (
	"context"
	"net"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/grandcat/zeroconf"
)

// Known mDNS service types advertised by Android TV / Google TV devices.
const (
	ServiceRemoteV2 = "_androidtvremote2._tcp" // Android TV Remote protocol v2 (pairing 6467, control 6466)
	ServiceCast     = "_googlecast._tcp"       // Google Cast; TXT carries friendly name and model
	ServiceADBTLS   = "_adb-tls-connect._tcp"  // Wireless debugging (Android 11+)
)

// DefaultServices is the set browsed when none is given.
var DefaultServices = []string{ServiceRemoteV2, ServiceCast, ServiceADBTLS}

// Device is one physical TV, merged across every service it advertises.
type Device struct {
	Name     string
	Model    string
	IPs      []net.IP
	Services map[string]int    // service type -> port
	TXT      map[string]string // merged TXT records
}

// Addr returns the preferred IP (first IPv4) as a string.
func (d *Device) Addr() string {
	for _, ip := range d.IPs {
		if ip.To4() != nil {
			return ip.String()
		}
	}
	if len(d.IPs) > 0 {
		return d.IPs[0].String()
	}
	return ""
}

// Discover browses the given services for timeout and returns devices merged by IP.
func Discover(ctx context.Context, timeout time.Duration, services ...string) ([]*Device, error) {
	if len(services) == 0 {
		services = DefaultServices
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var (
		mu      sync.Mutex
		devices = map[string]*Device{}
		wg      sync.WaitGroup
		errOnce error
	)

	for _, svc := range services {
		resolver, err := zeroconf.NewResolver(nil)
		if err != nil {
			return nil, err
		}
		entries := make(chan *zeroconf.ServiceEntry)

		wg.Add(1)
		go func(svc string) {
			defer wg.Done()
			for e := range entries {
				mu.Lock()
				merge(devices, svc, e)
				mu.Unlock()
			}
		}(svc)

		if err := resolver.Browse(ctx, svc, "local.", entries); err != nil {
			mu.Lock()
			errOnce = err
			mu.Unlock()
		}
	}

	<-ctx.Done()
	wg.Wait()

	out := make([]*Device, 0, len(devices))
	for _, d := range devices {
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Addr() < out[j].Addr() })
	if len(out) == 0 && errOnce != nil {
		return nil, errOnce
	}
	return out, nil
}

func merge(devices map[string]*Device, svc string, e *zeroconf.ServiceEntry) {
	ips := append(append([]net.IP{}, e.AddrIPv4...), e.AddrIPv6...)
	if len(ips) == 0 {
		return
	}
	key := ips[0].String()
	for _, ip := range e.AddrIPv4 {
		key = ip.String()
		break
	}

	d, ok := devices[key]
	if !ok {
		d = &Device{IPs: ips, Services: map[string]int{}, TXT: map[string]string{}}
		devices[key] = d
	}
	d.Services[svc] = e.Port

	for _, kv := range e.Text {
		k, v, _ := strings.Cut(kv, "=")
		d.TXT[k] = v
	}

	// Cast TXT has the nicest names: fn = friendly name, md = model.
	if fn := d.TXT["fn"]; fn != "" {
		d.Name = fn
	} else if d.Name == "" {
		d.Name = unescape(e.Instance)
	}
	if md := d.TXT["md"]; md != "" {
		d.Model = md
	}
}

// unescape turns mDNS instance escapes like "TCL\ 55" into "TCL 55".
func unescape(s string) string {
	return strings.ReplaceAll(s, `\ `, " ")
}
