// Command pkgmeta renders package manifests (winget, scoop, Homebrew, AUR,
// Flatpak) for an OPTIC release from its version and SHA256SUMS file.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"optic/internal/pkgmeta"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "pkgmeta:", err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("pkgmeta", flag.ContinueOnError)
	version := fs.String("version", "", "release version, e.g. v1.0.10 (required)")
	sums := fs.String("sums", "", "path to the SHA256SUMS file (required)")
	out := fs.String("out", "dist/pkgmeta", "output directory")
	date := fs.String("date", time.Now().UTC().Format("2006-01-02"), "release date, YYYY-MM-DD")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *version == "" || *sums == "" {
		return fmt.Errorf("-version and -sums are required")
	}

	f, err := os.Open(*sums)
	if err != nil {
		return err
	}
	defer f.Close()
	parsed, err := pkgmeta.ParseSums(f)
	if err != nil {
		return fmt.Errorf("%s: %w", *sums, err)
	}

	written, err := pkgmeta.Render(pkgmeta.Input{Version: *version, Sums: parsed, Date: *date}, *out)
	if err != nil {
		return err
	}
	for _, p := range written {
		fmt.Fprintln(stdout, p)
	}
	return nil
}
