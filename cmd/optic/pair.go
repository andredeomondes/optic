package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"optic/pkg/atvremote"
)

func runPair(ctx context.Context, args []string) int {
	fs := flag.NewFlagSet("pair", flag.ExitOnError)
	verbose := fs.Bool("v", false, "dump every protobuf frame (reverse engineering)")
	fs.Parse(args)
	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "uso: optic pair [-v] <ip>")
		return 2
	}
	host := fs.Arg(0)

	dir, err := atvremote.DefaultConfigDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
		return 1
	}
	cert, err := atvremote.LoadOrCreateCert(dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "erro no certificado:", err)
		return 1
	}

	var trace atvremote.Tracer
	if *verbose {
		trace = func(dir string, msg []byte) {
			label := "OPTIC → TV"
			if dir == "<" {
				label = "TV → OPTIC"
			}
			fmt.Printf("\n── %s (%d bytes) ──\n%s", label, len(msg), atvremote.DumpRaw(msg))
		}
	}

	fmt.Printf("Conectando em %s:%d...\n", host, atvremote.PairingPort)
	p, err := atvremote.StartPairing(ctx, host, cert, "OPTIC", trace)
	if err != nil {
		fmt.Fprintln(os.Stderr, "erro no pairing:", err)
		return 1
	}
	defer p.Close()

	if p.ServerName != "" {
		fmt.Printf("\nTV respondeu: %s\n", p.ServerName)
	}
	fmt.Println("Olhe a tela da TV: ela está mostrando um código de 6 caracteres.")

	in := bufio.NewScanner(os.Stdin)
	for attempt := 1; ; attempt++ {
		fmt.Print("\nCódigo: ")
		if !in.Scan() {
			fmt.Fprintln(os.Stderr, "\ncancelado")
			return 1
		}
		err := p.Finish(strings.TrimSpace(in.Text()))
		if err == nil {
			break
		}
		// Checksum failures are caught locally, so the session is still alive.
		if errors.Is(err, atvremote.ErrCodeTypo) && attempt < 3 {
			fmt.Println("✗", err, "— tente de novo")
			continue
		}
		fmt.Fprintln(os.Stderr, "✗ erro:", err)
		return 1
	}
	fmt.Println("\n✓ Pareado! A TV agora confia no certificado do OPTIC.")

	fmt.Printf("\nTestando porta de controle %d...\n", atvremote.ControlPort)
	first, err := atvremote.ProbeControl(ctx, host, cert)
	if err != nil {
		fmt.Fprintln(os.Stderr, "✗ conexão de controle falhou:", err)
		return 1
	}
	fmt.Printf("✓ Conexão de controle aceita. Primeira mensagem da TV (%d bytes):\n%s", len(first), atvremote.DumpRaw(first))
	fmt.Println("\nCertificado salvo em:", dir)
	return 0
}
