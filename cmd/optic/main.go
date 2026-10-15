// Command optic controls TVs on the local network from the terminal.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"time"

	"optic/pkg/discovery"
)

const usage = `OPTIC — Open PC-to-TV Interface Controller

Usage:
  optic discover [-t 5s] [-v]   find TVs on the local network
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
	timeout := fs.Duration("t", 5*time.Second, "how long to listen for mDNS answers")
	verbose := fs.Bool("v", false, "show all TXT records")
	fs.Parse(args)

	fmt.Printf("Procurando TVs na rede por %s...\n\n", *timeout)
	devices, err := discovery.Discover(ctx, *timeout)
	if err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
		return 1
	}
	if len(devices) == 0 {
		fmt.Println("Nenhuma TV encontrada. A TV está ligada e na mesma rede?")
		return 1
	}

	for _, d := range devices {
		fmt.Printf("▸ %s", d.Name)
		if d.Model != "" {
			fmt.Printf("  (%s)", d.Model)
		}
		fmt.Printf("\n  ip: %s\n", d.Addr())

		svcs := make([]string, 0, len(d.Services))
		for s := range d.Services {
			svcs = append(svcs, s)
		}
		sort.Strings(svcs)
		for _, s := range svcs {
			fmt.Printf("  %-24s porta %d\n", s, d.Services[s])
		}

		if *verbose {
			keys := make([]string, 0, len(d.TXT))
			for k := range d.TXT {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				fmt.Printf("    txt %s=%s\n", k, d.TXT[k])
			}
		}
		fmt.Println()
	}
	return 0
}
