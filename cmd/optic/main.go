// Command optic controls TVs on the local network from the terminal.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sort"
	"time"

	"optic/internal/adapter/compose"
	"optic/internal/adapter/mdns"
	"optic/internal/adapter/ssdp"
	"optic/internal/domain"
	"optic/pkg/discovery"
)

const usage = `OPTIC — Open PC-to-TV Interface Controller

Usage:
  optic discover [-t 5s] [-v] [-ssdp=false]  find TVs on the local network
  optic pair [-v] <ip>          pair with a TV (Remote protocol v2)
  optic key [-v] <ip> <KEY>...  send keys (HOME, BACK, VOLUME_UP...)
  optic app <ip> <name|uri>     open an app (youtube, netflix, prime, globoplay)
  optic type [-v] <ip> <text>   type text into the focused field
  optic search [-v] <ip> <query>  search with the TV's global search
  optic watch [-t 60s] <ip>     dump every message the TV sends
`

func main() {
	if len(os.Args) < 2 {
		fmt.Print(usage)
		os.Exit(2)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	switch os.Args[1] {
	case "discover":
		os.Exit(runDiscover(ctx, os.Args[2:]))
	case "pair":
		os.Exit(runPair(ctx, os.Args[2:]))
	case "key":
		os.Exit(runKey(ctx, os.Args[2:]))
	case "search":
		os.Exit(runSearch(ctx, os.Args[2:]))
	case "watch":
		os.Exit(runWatch(ctx, os.Args[2:]))
	case "type":
		os.Exit(runType(ctx, os.Args[2:]))
	case "app":
		os.Exit(runApp(ctx, os.Args[2:]))
	case "-h", "--help", "help":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", os.Args[1], usage)
		os.Exit(2)
	}
}

func runDiscover(ctx context.Context, args []string) int {
	fs := flag.NewFlagSet("discover", flag.ExitOnError)
	timeout := fs.Duration("t", 5*time.Second, "how long to listen for answers")
	verbose := fs.Bool("v", false, "show mDNS services and TXT records")
	useSSDP := fs.Bool("ssdp", true, "also search with SSDP (UPnP)")
	fs.Parse(args)

	sources := []domain.Discoverer{mdns.Discoverer{}}
	if *useSSDP {
		sources = append(sources, ssdp.Discoverer{})
	}

	// The mDNS details for -v run next to the combined scan, so the wait
	// does not double.
	var detailsCh chan map[string]*discovery.Device
	if *verbose {
		detailsCh = make(chan map[string]*discovery.Device, 1)
		go func() { detailsCh <- mdnsDetails(ctx, *timeout) }()
	}

	fmt.Printf("Procurando TVs na rede por %s...\n\n", *timeout)
	tvs, err := compose.Discoverer{Sources: sources}.Discover(ctx, *timeout)
	if err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
		return 1
	}
	if len(tvs) == 0 {
		fmt.Println("Nenhuma TV encontrada. A TV está ligada e na mesma rede?")
		return 1
	}

	var details map[string]*discovery.Device
	if detailsCh != nil {
		details = <-detailsCh
	}
	printTVs(os.Stdout, tvs, details)
	return 0
}

// mdnsDetails indexes the mDNS devices by IP. Errors are ignored: the details
// only add to the list that the combined scan already printed.
func mdnsDetails(ctx context.Context, timeout time.Duration) map[string]*discovery.Device {
	devices, _ := discovery.Discover(ctx, timeout)
	out := make(map[string]*discovery.Device, len(devices))
	for _, d := range devices {
		out[d.Addr()] = d
	}
	return out
}

// printTVs writes one block per TV. details is optional and keyed by IP.
func printTVs(w io.Writer, tvs []domain.TV, details map[string]*discovery.Device) {
	for _, tv := range tvs {
		fmt.Fprintf(w, "▸ %s  [%s]", tv.Name, tv.Platform.Label())
		if tv.Model != "" {
			fmt.Fprintf(w, "  (%s)", tv.Model)
		}
		fmt.Fprintf(w, "\n  ip: %s\n", tv.IP)

		if d := details[tv.IP]; d != nil {
			svcs := make([]string, 0, len(d.Services))
			for s := range d.Services {
				svcs = append(svcs, s)
			}
			sort.Strings(svcs)
			for _, s := range svcs {
				fmt.Fprintf(w, "  %-24s porta %d\n", s, d.Services[s])
			}

			keys := make([]string, 0, len(d.TXT))
			for k := range d.TXT {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				fmt.Fprintf(w, "    txt %s=%s\n", k, d.TXT[k])
			}
		}
		fmt.Fprintln(w)
	}
}
