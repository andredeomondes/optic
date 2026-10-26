package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// Tricky Windows paths: spaces, a quote and a non-ASCII letter.
const (
	trickySrc    = `C:\Users\Jo'ão Conceição\AppData\Roaming\optic\updates\1.1.0\OPTIC-windows-amd64-setup.exe`
	trickyExe    = `C:\Program Files\OPTIC Ação\optic-desktop.exe`
	trickyResult = `C:\Users\Jo'ão Conceição\AppData\Roaming\optic\updates\result.json`
)

func TestPSQuote(t *testing.T) {
	cases := []struct{ in, want string }{
		{``, `''`},
		{`C:\a b\c.exe`, `'C:\a b\c.exe'`},
		{`it's`, `'it''s'`},
		{`''`, `''''''`},
		{`ação`, `'ação'`},
		{`$env:x`, `'$env:x'`},
		{"a`b", "'a`b'"},
		{"a\u2018b\u2019c\u201ad\u201be", "'a\u2018\u2018b\u2019\u2019c\u201a\u201ad\u201b\u201be'"},
	}
	for _, c := range cases {
		if got := psQuote(c.in); got != c.want {
			t.Errorf("psQuote(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

const scriptHead = `$result = 'C:\Users\Jo''ão Conceição\AppData\Roaming\optic\updates\result.json'
$version = '1.1.0'
function Write-Result([bool]$ok, [string]$msg) {
  if ($msg.Length -gt 200) { $msg = $msg.Substring(0, 200) }
  try {
    [ordered]@{ version = $version; ok = $ok; error = $msg } | ConvertTo-Json | Set-Content -LiteralPath $result -Encoding UTF8 -ErrorAction Stop
  } catch {}
}
Wait-Process -Id 4242 -Timeout 30 -ErrorAction SilentlyContinue
`

func TestWindowsScriptInstaller(t *testing.T) {
	want := scriptHead + `try {
  $p = Start-Process -FilePath 'C:\Users\Jo''ão Conceição\AppData\Roaming\optic\updates\1.1.0\OPTIC-windows-amd64-setup.exe' -ArgumentList '/S' -Verb RunAs -Wait -PassThru -ErrorAction Stop
  if ($null -ne $p.ExitCode -and $p.ExitCode -ne 0) { throw "installer exit code $($p.ExitCode)" }
  Write-Result $true ''
} catch {
  Write-Result $false $_.Exception.Message
}
Start-Process -FilePath 'C:\Program Files\OPTIC Ação\optic-desktop.exe'
`
	got := windowsScript(KindInstaller, trickySrc, trickyExe, 4242, trickyResult, "1.1.0")
	if got != want {
		t.Errorf("installer script mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestWindowsScriptPortable(t *testing.T) {
	src := `C:\Users\Jo'ão Conceição\AppData\Roaming\optic\updates\1.1.0\OPTIC-windows-amd64-portable.exe`
	exe := `D:\Meus Apps\Jo'ão\OPTIC ç.exe`
	want := scriptHead + `$exe = 'D:\Meus Apps\Jo''ão\OPTIC ç.exe'
$old = 'D:\Meus Apps\Jo''ão\OPTIC ç.exe.old'
$ok = $false
try {
  Move-Item -LiteralPath $exe -Destination $old -Force -ErrorAction Stop
  try {
    Move-Item -LiteralPath 'C:\Users\Jo''ão Conceição\AppData\Roaming\optic\updates\1.1.0\OPTIC-windows-amd64-portable.exe' -Destination $exe -Force -ErrorAction Stop
    $ok = $true
    Write-Result $true ''
  } catch {
    $msg = $_.Exception.Message
    Move-Item -LiteralPath $old -Destination $exe -Force -ErrorAction SilentlyContinue
    Write-Result $false $msg
  }
} catch {
  Write-Result $false $_.Exception.Message
}
Start-Process -FilePath $exe
if ($ok) { Remove-Item -LiteralPath $old -Force -ErrorAction SilentlyContinue }
`
	got := windowsScript(KindPortable, src, exe, 4242, trickyResult, "1.1.0")
	if got != want {
		t.Errorf("portable script mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestWindowsScriptOtherKinds(t *testing.T) {
	for _, k := range []Kind{KindDMG, KindTarball, ""} {
		if s := windowsScript(k, "a", "b", 1, "c", "1.0.0"); s != "" {
			t.Errorf("windowsScript(%q) = %q, want empty", k, s)
		}
	}
}

func TestScriptBytesHasBOM(t *testing.T) {
	b := scriptBytes("Write-Host 'ação'\n")
	if !bytes.HasPrefix(b, []byte{0xEF, 0xBB, 0xBF}) {
		t.Fatalf("missing UTF-8 BOM: % x", b[:3])
	}
	if string(b[3:]) != "Write-Host 'ação'\n" {
		t.Errorf("body = %q", b[3:])
	}
}

func TestPowerShellArgs(t *testing.T) {
	got := powerShellArgs(`C:\x y\apply-update.ps1`)
	want := []string{"-NoProfile", "-ExecutionPolicy", "Bypass", "-WindowStyle", "Hidden", "-File", `C:\x y\apply-update.ps1`}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("powerShellArgs = %q, want %q", got, want)
	}
	if p := powerShellPath(`C:\Windows`); p != filepath.Join(`C:\Windows`, "System32", "WindowsPowerShell", "v1.0", "powershell.exe") {
		t.Errorf("powerShellPath = %q", p)
	}
	if p := powerShellPath(""); p != "" {
		t.Errorf("powerShellPath(empty) = %q, want empty", p)
	}
}

func TestValidVersion(t *testing.T) {
	for _, v := range []string{"1.1.0", "v1.1.0", "10.20.30", "1.1.0-rc.1", "1.0.0+build.5"} {
		if !validVersion(v) {
			t.Errorf("validVersion(%q) = false", v)
		}
	}
	for _, v := range []string{"", "1.1", "1.1.0.0", "x1.1.0", "1.1.0'", "../1.0.0", "1.1.0\n", "latest"} {
		if validVersion(v) {
			t.Errorf("validVersion(%q) = true", v)
		}
	}
}

// --- tar extraction ---

type tarEntry struct {
	hdr  tar.Header
	body string
}

func makeTarGz(t *testing.T, entries []tarEntry) string {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, e := range entries {
		h := e.hdr
		if h.Typeflag == tar.TypeReg && h.Size == 0 {
			h.Size = int64(len(e.body))
		}
		if h.Mode == 0 {
			h.Mode = 0o644
		}
		if err := tw.WriteHeader(&h); err != nil {
			t.Fatal(err)
		}
		if h.Typeflag == tar.TypeReg {
			if _, err := tw.Write([]byte(e.body)); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "OPTIC-linux-amd64.tar.gz")
	if err := os.WriteFile(p, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func tarDir(name string) tarEntry {
	return tarEntry{hdr: tar.Header{Name: name, Typeflag: tar.TypeDir, Mode: 0o755}}
}
func tarFile(name, body string, mode int64) tarEntry {
	return tarEntry{hdr: tar.Header{Name: name, Typeflag: tar.TypeReg, Mode: mode}, body: body}
}

func TestExtractTarballHappyPath(t *testing.T) {
	src := makeTarGz(t, []tarEntry{
		tarDir("OPTIC/"),
		tarFile("OPTIC/optic-desktop", "ELF", 0o755),
		tarFile("OPTIC/install.sh", "#!/bin/sh", 0o755),
		tarFile("OPTIC/LICENSE", "MIT", 0o644),
		tarFile("OPTIC/sub/deep.txt", "x", 0o644), // parent dir not listed
	})
	dest := t.TempDir()
	// Leftovers from an earlier attempt are replaced.
	os.MkdirAll(filepath.Join(dest, "OPTIC"), 0o755)
	os.WriteFile(filepath.Join(dest, "OPTIC", "stale"), []byte("old"), 0o644)

	if err := extractTarball(src, dest, defaultExtractLimits); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{"optic-desktop": "ELF", "install.sh": "#!/bin/sh", "LICENSE": "MIT", "sub/deep.txt": "x"} {
		b, err := os.ReadFile(filepath.Join(dest, "OPTIC", filepath.FromSlash(name)))
		if err != nil || string(b) != body {
			t.Errorf("%s = %q, %v", name, b, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dest, "OPTIC", "stale")); !os.IsNotExist(err) {
		t.Errorf("stale file kept: %v", err)
	}
	if exec, _ := os.Stat(filepath.Join(dest, "OPTIC", "optic-desktop")); exec != nil && os.PathSeparator == '/' && exec.Mode().Perm()&0o100 == 0 {
		t.Errorf("executable bit lost: %v", exec.Mode())
	}
}

func TestExtractTarballRejects(t *testing.T) {
	cases := map[string][]tarEntry{
		"dotdot":         {tarDir("OPTIC/"), tarFile("OPTIC/../x", "x", 0o644)},
		"leading dotdot": {tarFile("../x", "x", 0o644)},
		"absolute":       {tarFile("/etc/x", "x", 0o644)},
		"drive":          {tarFile("C:/x", "x", 0o644)},
		"backslash":      {tarFile(`OPTIC\..\x`, "x", 0o644)},
		"outside OPTIC":  {tarFile("other/x", "x", 0o644)},
		"symlink":        {tarDir("OPTIC/"), {hdr: tar.Header{Name: "OPTIC/link", Typeflag: tar.TypeSymlink, Linkname: "/etc/passwd"}}},
		"symlink inside": {tarDir("OPTIC/"), {hdr: tar.Header{Name: "OPTIC/link", Typeflag: tar.TypeSymlink, Linkname: "install.sh"}}},
		"hardlink":       {tarDir("OPTIC/"), {hdr: tar.Header{Name: "OPTIC/hl", Typeflag: tar.TypeLink, Linkname: "OPTIC/x"}}},
		"char device":    {{hdr: tar.Header{Name: "OPTIC/tty", Typeflag: tar.TypeChar}}},
		"block device":   {{hdr: tar.Header{Name: "OPTIC/sda", Typeflag: tar.TypeBlock}}},
		"fifo":           {{hdr: tar.Header{Name: "OPTIC/fifo", Typeflag: tar.TypeFifo}}},
		"duplicate":      {tarFile("OPTIC/a", "1", 0o644), tarFile("OPTIC/a", "2", 0o644)},
		"empty":          {},
	}
	for name, entries := range cases {
		t.Run(name, func(t *testing.T) {
			src := makeTarGz(t, entries)
			dest := t.TempDir()
			if err := extractTarball(src, dest, defaultExtractLimits); err == nil {
				t.Fatal("want error")
			}
			if _, err := os.Stat(filepath.Join(dest, "OPTIC")); !os.IsNotExist(err) {
				t.Errorf("OPTIC folder left behind after failure: %v", err)
			}
			if _, err := os.Stat(filepath.Join(filepath.Dir(dest), "x")); !os.IsNotExist(err) {
				t.Errorf("file written outside dest")
			}
		})
	}
}

func TestExtractTarballCaps(t *testing.T) {
	big := makeTarGz(t, []tarEntry{tarFile("OPTIC/a", strings.Repeat("x", 600), 0o644), tarFile("OPTIC/b", strings.Repeat("x", 600), 0o644)})
	if err := extractTarball(big, t.TempDir(), extractLimits{maxBytes: 1000, maxFiles: 10}); err == nil {
		t.Error("size cap not enforced")
	}
	many := makeTarGz(t, []tarEntry{tarFile("OPTIC/a", "1", 0o644), tarFile("OPTIC/b", "2", 0o644), tarFile("OPTIC/c", "3", 0o644)})
	if err := extractTarball(many, t.TempDir(), extractLimits{maxBytes: 1000, maxFiles: 2}); err == nil {
		t.Error("file count cap not enforced")
	}
	if err := extractTarball(many, t.TempDir(), extractLimits{maxBytes: 1000, maxFiles: 3}); err != nil {
		t.Errorf("at the cap: %v", err)
	}
	if defaultExtractLimits.maxBytes != 400<<20 {
		t.Errorf("default size cap = %d", defaultExtractLimits.maxBytes)
	}
}

func TestExtractTarballNotGzip(t *testing.T) {
	p := filepath.Join(t.TempDir(), "x.tar.gz")
	os.WriteFile(p, []byte("not gzip"), 0o644)
	if err := extractTarball(p, t.TempDir(), defaultExtractLimits); err == nil {
		t.Error("want error")
	}
}

// --- result file ---

func TestReadResult(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, ResultFile)

	if r, err := ReadResult(d); r != nil || err != nil {
		t.Fatalf("missing: %v, %v", r, err)
	}

	// Windows PowerShell 5.1 writes UTF-8 with a BOM and CRLF.
	body := "\xEF\xBB\xBF{\r\n    \"version\":  \"1.1.0\",\r\n    \"ok\":  false,\r\n    \"error\":  \"A operação foi cancelada pelo usuário\"\r\n}\r\n"
	os.WriteFile(p, []byte(body), 0o644)
	r, err := ReadResult(d)
	if err != nil {
		t.Fatal(err)
	}
	want := &Result{Version: "1.1.0", OK: false, Error: "A operação foi cancelada pelo usuário"}
	if !reflect.DeepEqual(r, want) {
		t.Errorf("got %+v, want %+v", r, want)
	}
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Error("result file not deleted")
	}

	os.WriteFile(p, []byte(`{"version":"1.1.0","ok":true,"error":""}`), 0o644)
	if r, _ := ReadResult(d); r == nil || !r.OK {
		t.Errorf("ok result: %+v", r)
	}

	os.WriteFile(p, []byte("{garbage"), 0o644)
	if r, err := ReadResult(d); r != nil || err == nil {
		t.Errorf("corrupt: %v, %v", r, err)
	}
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Error("corrupt result file not deleted")
	}
}

// --- cleanup ---

func TestCleanup(t *testing.T) {
	root := t.TempDir()
	d := filepath.Join(root, "updates")
	for _, sub := range []string{"1.0.9", "1.1.0", "v1.0.8", "1.2.0-rc.1", "notes", "1.0"} {
		os.MkdirAll(filepath.Join(d, sub, "OPTIC"), 0o755)
		os.WriteFile(filepath.Join(d, sub, "f"), []byte("x"), 0o644)
	}
	os.WriteFile(filepath.Join(d, ResultFile), []byte("{}"), 0o644)
	os.WriteFile(filepath.Join(d, "2.0.0"), []byte("a file, not a folder"), 0o644)
	outside := filepath.Join(root, "1.0.7")
	os.MkdirAll(outside, 0o755)

	Cleanup(d, "v1.1.0")

	gone := []string{"1.0.9", "v1.0.8", "1.2.0-rc.1"}
	kept := []string{"1.1.0", "notes", "1.0", ResultFile, "2.0.0"}
	for _, g := range gone {
		if _, err := os.Stat(filepath.Join(d, g)); !os.IsNotExist(err) {
			t.Errorf("%s not removed", g)
		}
	}
	for _, k := range kept {
		if _, err := os.Stat(filepath.Join(d, k)); err != nil {
			t.Errorf("%s removed: %v", k, err)
		}
	}
	if _, err := os.Stat(outside); err != nil {
		t.Errorf("folder outside dir touched: %v", err)
	}

	Cleanup(filepath.Join(root, "missing"), "1.1.0") // no panic, no error
}
