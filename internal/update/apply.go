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
	"strconv"
	"strings"
	"time"
)

// ResultFile is written by the Windows update script into the updates dir
// and read (then deleted) on the next start.
const ResultFile = "result.json"

// scriptName is the PowerShell script written next to the download;
// markerName is the file its first line writes, proving it was allowed to run;
// cancelName is written by Apply when it stopped waiting, so a script that
// starts late exits without installing.
const (
	scriptName = "apply-update.ps1"
	markerName = "apply.started"
	cancelName = "apply.cancel"
)

// Errors shown to the user (the UI translates them).
var (
	ErrApplyBlocked = errors.New("O Windows não deixou o atualizador rodar. Baixe a nova versão pela página de releases.")
	ErrNoXdgOpen    = errors.New("xdg-open não encontrado; abra a pasta da nova versão manualmente.")
)

// How long Apply waits for the script's start marker, and how often it looks.
const (
	applyStartTimeout = 15 * time.Second
	applyStartPoll    = 100 * time.Millisecond
)

// zoneIdentifier is the Mark of the Web ("downloaded from the Internet"), so
// SmartScreen evaluates the downloaded exe like any browser download.
const zoneIdentifier = "[ZoneTransfer]\r\nZoneId=3\r\n"

// macQuarantine is the com.apple.quarantine value a browser would set, so
// Gatekeeper evaluates the DMG as a download.
func macQuarantine(now time.Time) string {
	return fmt.Sprintf("0081;%08x;OPTIC;", now.Unix())
}

// Result is the outcome of the last Windows update.
type Result struct {
	Version string `json:"version"`
	OK      bool   `json:"ok"`
	Error   string `json:"error"`
}

var versionRe = regexp.MustCompile(`^v?[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$`)

// validVersion reports whether v looks like a semver (optional "v").
func validVersion(v string) bool { return versionRe.MatchString(v) }

// compareVersions orders two valid versions (-1, 0, 1). Build metadata is
// ignored; a prerelease sorts before its release and prereleases compare
// as text.
func compareVersions(a, b string) int {
	split := func(v string) ([3]int, string) {
		v, _, _ = strings.Cut(strings.TrimPrefix(v, "v"), "+")
		core, pre, _ := strings.Cut(v, "-")
		var n [3]int
		for i, f := range strings.SplitN(core, ".", 3) {
			n[i], _ = strconv.Atoi(f)
		}
		return n, pre
	}
	na, pa := split(a)
	nb, pb := split(b)
	for i := range na {
		if na[i] != nb[i] {
			if na[i] < nb[i] {
				return -1
			}
			return 1
		}
	}
	switch {
	case pa == pb:
		return 0
	case pa == "":
		return 1
	case pb == "":
		return -1
	}
	return comparePrerelease(pa, pb)
}

// comparePrerelease orders prerelease tags per semver: dot-separated
// fields, numeric ones by value and below alphanumeric ones, and a tag that
// runs out of fields first is lower.
func comparePrerelease(a, b string) int {
	fa, fb := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(fa) && i < len(fb); i++ {
		na, ea := strconv.Atoi(fa[i])
		nb, eb := strconv.Atoi(fb[i])
		switch {
		case ea == nil && eb == nil:
			if na != nb {
				if na < nb {
					return -1
				}
				return 1
			}
		case ea == nil:
			return -1
		case eb == nil:
			return 1
		default:
			if c := strings.Compare(fa[i], fb[i]); c != 0 {
				return c
			}
		}
	}
	switch {
	case len(fa) < len(fb):
		return -1
	case len(fa) > len(fb):
		return 1
	}
	return 0
}

// IsNewer reports whether latest is a valid version newer than current
// (either may start with "v"). Invalid input is never newer.
func IsNewer(current, latest string) bool {
	return validVersion(current) && validVersion(latest) && compareVersions(latest, current) > 0
}

var sha256Re = regexp.MustCompile(`^[0-9a-f]{64}$`)

// validSHA256 reports whether s is a lowercase hex SHA-256, as Fetch verifies.
func validSHA256(s string) bool { return sha256Re.MatchString(s) }

// validateApply checks the Windows Apply arguments before anything is written.
func validateApply(t Target, path, sha256, exe string, pid int, dir, version string) error {
	switch {
	case t.Kind != KindInstaller && t.Kind != KindPortable:
		return ErrUnsupported
	case !filepath.IsAbs(path) || !filepath.IsAbs(exe) || !filepath.IsAbs(dir):
		return errors.New("update: paths must be absolute")
	case pid <= 0:
		return errors.New("update: invalid pid")
	case !validVersion(version):
		return errors.New("update: invalid version")
	case !validSHA256(sha256):
		return errors.New("update: invalid sha256")
	}
	return nil
}

// clearStale removes the marker and cancel files of an earlier run. A file
// that exists but cannot be removed would fake (or block) the handshake, so
// that refuses the update.
func clearStale(paths ...string) error {
	for _, p := range paths {
		if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%w (%v)", ErrApplyBlocked, err)
		}
	}
	return nil
}

// cancelStart tells a script that has not started yet to exit without
// installing: Apply gave up on it and the app stays open.
func cancelStart(cancelPath string) error {
	return os.WriteFile(cancelPath, []byte("cancel"), 0o600)
}

// awaitStart waits until the script's marker exists (started → nil) or the
// process exits or timeout passes without it (→ ErrApplyBlocked), e.g. when
// policy, AppLocker or Smart App Control refuses the script.
func awaitStart(markerExists func() bool, exited <-chan struct{}, timeout, poll time.Duration) error {
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	tick := time.NewTicker(poll)
	defer tick.Stop()
	for {
		if markerExists() {
			return nil
		}
		select {
		case <-exited:
		case <-deadline.C:
		case <-tick.C:
			continue
		}
		// Exited or timed out: one last look covers a marker written just now.
		if markerExists() {
			return nil
		}
		return ErrApplyBlocked
	}
}

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

// Cleanup removes the version folders in dir older than current. Newer ones
// (an update not installed yet) and current itself stay, minus leftover
// *.part downloads and, in current's folder, the installer already used.
// Only direct subfolders named like a version are touched; symlinks are
// skipped. An invalid current removes nothing. Call it only when no
// download is running.
func Cleanup(dir, current string) {
	if !validVersion(current) {
		return
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		name := e.Name()
		if !e.IsDir() || !validVersion(name) {
			continue
		}
		folder := filepath.Join(dir, name)
		if compareVersions(name, current) < 0 {
			os.RemoveAll(folder)
			continue
		}
		cleanFolder(folder, compareVersions(name, current) == 0)
	}
}

// cleanFolder removes *.part files and, when installed is true, the
// release assets in folder. Only regular files directly inside are touched.
func cleanFolder(folder string, installed bool) {
	files, err := os.ReadDir(folder)
	if err != nil {
		return
	}
	for _, f := range files {
		n := f.Name()
		if !f.Type().IsRegular() {
			continue
		}
		if strings.HasSuffix(n, ".part") || installed && knownAsset(n) {
			os.Remove(filepath.Join(folder, n))
		}
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
// starts the app again, new or old (unless it never exited). The first line
// writes markerPath so Apply knows the script was allowed to run; if
// cancelPath exists right after, Apply already gave up and the script exits
// without installing. src is
// re-checked against sha256 right before use. Other kinds or a bad sha256
// give "".
func windowsScript(kind Kind, src, sha256, exe string, pid int, resultPath, markerPath, cancelPath, version string) string {
	if kind != KindInstaller && kind != KindPortable || !validSHA256(sha256) {
		return ""
	}
	var b strings.Builder
	w := func(format string, a ...any) { fmt.Fprintf(&b, format+"\n", a...) }
	hashCheck := fmt.Sprintf("if ((Get-FileHash -LiteralPath %s -Algorithm SHA256 -ErrorAction Stop).Hash -ne '%s') { throw 'checksum' }",
		psQuote(src), strings.ToUpper(sha256))
	w("Set-Content -LiteralPath %s -Value 'started' -ErrorAction SilentlyContinue", psQuote(markerPath))
	w("if (Test-Path -LiteralPath %s) { exit }", psQuote(cancelPath))
	w("$result = %s", psQuote(resultPath))
	w("$version = %s", psQuote(version))
	w("function Write-Result([bool]$ok, [string]$msg) {")
	w("  if ($msg.Length -gt 200) { $msg = $msg.Substring(0, 200) }")
	w("  try {")
	w("    [ordered]@{ version = $version; ok = $ok; error = $msg } | ConvertTo-Json | Set-Content -LiteralPath $result -Encoding UTF8 -ErrorAction Stop")
	w("  } catch {}")
	w("}")
	w("Wait-Process -Id %d -Timeout 30 -ErrorAction SilentlyContinue", pid)
	// Still running: install nothing and start nothing (it is open already).
	w("if (Get-Process -Id %d -ErrorAction SilentlyContinue) {", pid)
	w("  Write-Result $false 'O OPTIC não fechou a tempo.'")
	w("  exit")
	w("}")
	if kind == KindInstaller {
		// Installed apps run from Program Files, so exe is also the path the
		// installer writes to. UAC is shown by Windows; a refusal or a
		// Smart App Control block lands in catch.
		w("try {")
		w("  %s", hashCheck)
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
	w("  %s", hashCheck)
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
	w("if (-not (Test-Path -LiteralPath $exe)) { $exe = $old }")
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
	entries := 0 // files and folders, capped by lim.maxFiles
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
		if entries++; entries > lim.maxFiles {
			return errors.New("tarball: too many entries")
		}
		switch h.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			total += h.Size
			if h.Size < 0 || total > lim.maxBytes {
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
