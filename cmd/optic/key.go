package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"optic/pkg/atvremote"
)

func connect(ctx context.Context, host string, verbose bool) (*atvremote.Remote, error) {
	dir, err := atvremote.DefaultConfigDir()
	if err != nil {
		return nil, err
	}
	cert, err := atvremote.LoadOrCreateCert(dir)
	if err != nil {
		return nil, err
	}
	var trace atvremote.Tracer
	if verbose {
		trace = func(dir string, msg []byte) {
			fmt.Printf("── %s (%d bytes)\n%s", dir, len(msg), atvremote.DumpRaw(msg))
		}
	}
	return atvremote.Connect(ctx, host, cert, nil, trace)
}

func runKey(ctx context.Context, args []string) int {
	fs := flag.NewFlagSet("key", flag.ExitOnError)
	verbose := fs.Bool("v", false, "dump every protobuf frame")
	fs.Parse(args)
	if fs.NArg() < 2 {
		fmt.Fprintf(os.Stderr, "uso: optic key [-v] <ip> <KEY> [KEY...]\nteclas: %s\n", strings.Join(atvremote.KeyNames(), " "))
		return 2
	}

	var codes []int
	for _, name := range fs.Args()[1:] {
		code, ok := atvremote.LookupKey(name)
		if !ok {
			fmt.Fprintf(os.Stderr, "tecla desconhecida %q\n", name)
			return 2
		}
		codes = append(codes, code)
	}

	r, err := connect(ctx, fs.Arg(0), *verbose)
	if err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
		return 1
	}
	defer r.Close()

	for i, code := range codes {
		if err := r.SendKey(code); err != nil {
			fmt.Fprintln(os.Stderr, "erro:", err)
			return 1
		}
		fmt.Printf("✓ %s\n", strings.ToUpper(fs.Args()[i+1]))
		time.Sleep(150 * time.Millisecond)
	}
	return 0
}

func runApp(ctx context.Context, args []string) int {
	if len(args) != 2 {
		fmt.Fprintln(os.Stderr, "uso: optic app <ip> <youtube|netflix|prime|globoplay|uri>")
		return 2
	}
	link := atvremote.Apps[strings.ToLower(args[1])]
	if link == "" {
		link = args[1]
	}
	r, err := connect(ctx, args[0], false)
	if err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
		return 1
	}
	defer r.Close()
	if err := r.LaunchApp(link); err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
		return 1
	}
	time.Sleep(300 * time.Millisecond)
	fmt.Println("✓ abrindo", link)
	return 0
}

func runType(ctx context.Context, args []string) int {
	fs := flag.NewFlagSet("type", flag.ExitOnError)
	verbose := fs.Bool("v", false, "dump every protobuf frame")
	fs.Parse(args)
	if fs.NArg() < 2 {
		fmt.Fprintln(os.Stderr, "uso: optic type [-v] <ip> <texto...>")
		return 2
	}
	r, err := connect(ctx, fs.Arg(0), *verbose)
	if err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
		return 1
	}
	defer r.Close()
	// Give the TV a moment to announce IME state if a keyboard is open.
	time.Sleep(400 * time.Millisecond)
	text := strings.Join(fs.Args()[1:], " ")
	if err := r.SendText(text); err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
		return 1
	}
	time.Sleep(300 * time.Millisecond)
	fmt.Printf("✓ digitado %q\n", text)
	return 0
}

// runWatch keeps a control session open and dumps every frame the TV sends.
// Used to reverse engineer messages (e.g. what arrives when the keyboard opens).
func runWatch(ctx context.Context, args []string) int {
	fs := flag.NewFlagSet("watch", flag.ExitOnError)
	dur := fs.Duration("t", 60*time.Second, "how long to listen")
	fs.Parse(args)
	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "uso: optic watch [-t 60s] <ip>")
		return 2
	}
	r, err := connect(ctx, fs.Arg(0), true)
	if err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
		return 1
	}
	defer r.Close()
	fmt.Printf("\n▸ escutando por %s — use o controle físico da TV agora (Ctrl+C para parar)\n\n", *dur)
	select {
	case <-ctx.Done():
	case <-r.Done():
		fmt.Println("conexão encerrada:", r.Err())
	case <-time.After(*dur):
	}
	return 0
}

func runSearch(ctx context.Context, args []string) int {
	fs := flag.NewFlagSet("search", flag.ExitOnError)
	verbose := fs.Bool("v", false, "dump every protobuf frame")
	fs.Parse(args)
	if fs.NArg() < 2 {
		fmt.Fprintln(os.Stderr, "uso: optic search [-v] <ip> <busca...>")
		return 2
	}
	r, err := connect(ctx, fs.Arg(0), *verbose)
	if err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
		return 1
	}
	defer r.Close()
	q := strings.Join(fs.Args()[1:], " ")
	if err := r.Search(q); err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
		return 1
	}
	time.Sleep(500 * time.Millisecond)
	fmt.Printf("✓ buscando %q na TV\n", q)
	return 0
}
