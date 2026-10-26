package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sums = `aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa  OPTIC-windows-amd64-setup.exe
bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb  OPTIC-windows-amd64-portable.exe
cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc  OPTIC-macos-universal.dmg
dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd  OPTIC-linux-amd64.tar.gz
eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee  OPTIC-linux-amd64-webkit41.tar.gz
`

func TestRunWritesFiles(t *testing.T) {
	dir := t.TempDir()
	sumsPath := filepath.Join(dir, "SHA256SUMS")
	if err := os.WriteFile(sumsPath, []byte(sums), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out")
	var stdout bytes.Buffer
	err := run([]string{"-version", "v1.2.3", "-sums", sumsPath, "-out", out, "-date", "2026-01-02"}, &stdout)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(out, "scoop", "optic.json")); err != nil {
		t.Error(err)
	}
	if !strings.Contains(stdout.String(), "optic.json") {
		t.Errorf("written files not printed: %q", stdout.String())
	}
}

func TestRunErrors(t *testing.T) {
	if err := run([]string{"-sums", "x"}, &bytes.Buffer{}); err == nil {
		t.Error("missing -version must fail")
	}
	if err := run([]string{"-version", "1.2.3"}, &bytes.Buffer{}); err == nil {
		t.Error("missing -sums must fail")
	}
	if err := run([]string{"-version", "1.2.3", "-sums", filepath.Join(t.TempDir(), "nope")}, &bytes.Buffer{}); err == nil {
		t.Error("unreadable sums must fail")
	}
}
