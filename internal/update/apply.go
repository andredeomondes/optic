package update

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ResultFile is written by the Windows update script into the updates dir
// and read (then deleted) on the next start.
const ResultFile = "result.json"

// scriptName is the PowerShell script written next to the download.
const scriptName = "apply-update.ps1"

// Result is the outcome of the last Windows update.
type Result struct {
	Version string `json:"version"`
	OK      bool   `json:"ok"`
	Error   string `json:"error"`
}

var versionRe = regexp.MustCompile(`^v?[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$`)

// validVersion reports whether v looks like a semver (optional "v").
func validVersion(v string) bool { return versionRe.MatchString(v) }

// ReadResult reads and deletes <dir>/result.json. A missing file gives
// (nil, nil); an unreadable one is deleted and reported as an error.
func ReadResult(dir string) (*Result, error) {
	p := filepath.Join(dir, ResultFile)
	f, err := os.Open(p)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	b, err := io.ReadAll(io.LimitReader(f, 64<<10))
	f.Close()
	os.Remove(p)
	if err != nil {
		return nil, err
	}
	// Windows PowerShell 5.1 writes UTF-8 with a BOM.
	b = []byte(strings.TrimPrefix(string(b), "\uFEFF"))
	var r Result
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, fmt.Errorf("update result: %w", err)
	}
	return &r, nil
}

// Cleanup removes the version folders in dir other than keepVersion. Only
// direct subfolders named like a version are touched; symlinks are skipped.
func Cleanup(dir, keepVersion string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	keep := strings.TrimPrefix(keepVersion, "v")
	for _, e := range entries {
		name := e.Name()
		if !e.IsDir() || !validVersion(name) || strings.TrimPrefix(name, "v") == keep {
			continue
		}
		os.RemoveAll(filepath.Join(dir, name))
	}
}

// psQuote returns s as a single-quoted PowerShell literal. PowerShell also
// treats the typographic quotes U+2018..U+201B as single quotes, so those
// are doubled too.
func psQuote(s string) string {
	var b strings.Builder
	b.WriteByte('\'')
	for _, r := range s {
		b.WriteRune(r)
		switch r {
		case '\'', '\u2018', '\u2019', '\u201a', '\u201b':
			b.WriteRune(r)
		}
	}
	b.WriteByte('\'')
	return b.String()
}

// windowsScript builds the PowerShell script that installs src and relaunches
// exe once the app (pid) has exited. It writes the outcome to resultPath and
// always starts the app again, new or old. Other kinds give "".
func windowsScript(kind Kind, src, exe string, pid int, resultPath, version string) string {
	if kind != KindInstaller && kind != KindPortable {
		return ""
	}
	var b strings.Builder
	w := func(format string, a ...any) { fmt.Fprintf(&b, format+"\n", a...) }
	w("$result = %s", psQuote(resultPath))
	w("$version = %s", psQuote(version))
	w("function Write-Result([bool]$ok, [string]$msg) {")
	w("  if ($msg.Length -gt 200) { $msg = $msg.Substring(0, 200) }")
	w("  try {")
	w("    [ordered]@{ version = $version; ok = $ok; error = $msg } | ConvertTo-Json | Set-Content -LiteralPath $result -Encoding UTF8 -ErrorAction Stop")
	w("  } catch {}")
	w("}")
	w("Wait-Process -Id %d -Timeout 30 -ErrorAction SilentlyContinue", pid)
	if kind == KindInstaller {
		// Installed apps run from Program Files, so exe is also the path the
		// installer writes to. UAC is shown by Windows; a refusal or a
		// Smart App Control block lands in catch.
		w("try {")
		w("  $p = Start-Process -FilePath %s -ArgumentList '/S' -Verb RunAs -Wait -PassThru -ErrorAction Stop", psQuote(src))
		w(`  if ($null -ne $p.ExitCode -and $p.ExitCode -ne 0) { throw "installer exit code $($p.ExitCode)" }`)
		w("  Write-Result $true ''")
		w("} catch {")
		w("  Write-Result $false $_.Exception.Message")
		w("}")
		w("Start-Process -FilePath %s", psQuote(exe))
		return b.String()
	}
	w("$exe = %s", psQuote(exe))
	w("$old = %s", psQuote(exe+".old"))
	w("$ok = $false")
	w("try {")
	w("  Move-Item -LiteralPath $exe -Destination $old -Force -ErrorAction Stop")
	w("  try {")
	w("    Move-Item -LiteralPath %s -Destination $exe -Force -ErrorAction Stop", psQuote(src))
	w("    $ok = $true")
	w("    Write-Result $true ''")
	w("  } catch {")
	w("    $msg = $_.Exception.Message")
	w("    Move-Item -LiteralPath $old -Destination $exe -Force -ErrorAction SilentlyContinue")
	w("    Write-Result $false $msg")
	w("  }")
	w("} catch {")
	w("  Write-Result $false $_.Exception.Message")
	w("}")
	w("Start-Process -FilePath $exe")
	w("if ($ok) { Remove-Item -LiteralPath $old -Force -ErrorAction SilentlyContinue }")
	return b.String()
}

// scriptBytes prefixes a UTF-8 BOM so Windows PowerShell 5.1 reads non-ASCII
// paths as UTF-8 instead of the ANSI code page.
func scriptBytes(script string) []byte {
	return append([]byte{0xEF, 0xBB, 0xBF}, script...)
}

// powerShellPath is Windows PowerShell under systemRoot ("" if unknown), so
// PATH is never consulted.
func powerShellPath(systemRoot string) string {
	if systemRoot == "" {
		return ""
	}
	return filepath.Join(systemRoot, "System32", "WindowsPowerShell", "v1.0", "powershell.exe")
}

// powerShellArgs runs one script file. -ExecutionPolicy Bypass applies only
// to this one invocation; it changes no system or user setting.
func powerShellArgs(script string) []string {
	return []string{"-NoProfile", "-ExecutionPolicy", "Bypass", "-WindowStyle", "Hidden", "-File", script}
}

// extractLimits caps what a tarball may unpack to.
type extractLimits struct {
	maxBytes int64
	maxFiles int
}

var defaultExtractLimits = extractLimits{maxBytes: 400 << 20, maxFiles: 1000}

// tarRoot is the single top-level folder of the Linux release tarball.
const tarRoot = "OPTIC"

// extractTarball unpacks the release tarball into dest/OPTIC, replacing any
// earlier copy. Only regular files and folders under OPTIC/ are accepted:
// absolute paths, "..", links and devices fail the whole extraction, and
// nothing is left behind on failure.
func extractTarball(src, dest string, lim extractLimits) (err error) {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("tarball: %w", err)
	}
	defer gz.Close()

	root := filepath.Join(dest, tarRoot)
	if err := os.RemoveAll(root); err != nil {
		return err
	}
	defer func() {
		if err != nil {
			os.RemoveAll(root)
		}
	}()

	tr := tar.NewReader(gz)
	var total int64
	files := 0
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tarball: %w", err)
		}
		rel, err := tarPath(h.Name)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, rel)
		switch h.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			files++
			total += h.Size
			if files > lim.maxFiles || h.Size < 0 || total > lim.maxBytes {
				return errors.New("tarball: too large")
			}
			if err := writeTarFile(target, tr, h); err != nil {
				return err
			}
		default:
			return fmt.Errorf("tarball: unsupported entry %q", h.Name)
		}
	}
	if fi, err := os.Stat(root); err != nil || !fi.IsDir() {
		return errors.New("tarball: no OPTIC folder")
	}
	return nil
}

// tarPath checks an entry name and returns it as a relative OS path that
// starts with OPTIC.
func tarPath(name string) (string, error) {
	bad := fmt.Errorf("tarball: unsafe path %q", name)
	if strings.ContainsAny(name, "\\:\x00") || strings.HasPrefix(name, "/") {
		return "", bad
	}
	parts := strings.Split(strings.TrimSuffix(name, "/"), "/")
	if parts[0] != tarRoot {
		return "", bad
	}
	for _, p := range parts {
		if p == "" || p == "." || p == ".." {
			return "", bad
		}
	}
	return filepath.Join(parts...), nil
}

func writeTarFile(target string, r io.Reader, h *tar.Header) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	mode := os.FileMode(0o644)
	if h.Mode&0o111 != 0 {
		mode = 0o755
	}
	// O_EXCL: a name seen twice is an error, never an overwrite.
	out, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, io.LimitReader(r, h.Size))
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	return err
}
