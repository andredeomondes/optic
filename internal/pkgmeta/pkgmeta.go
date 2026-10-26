// Package pkgmeta renders ready-to-submit package manifests (winget, scoop,
// Homebrew, AUR, Flatpak) for an OPTIC release from its version and checksums.
package pkgmeta

import (
	"bufio"
	"bytes"
	"embed"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"
)

// Release asset names, as uploaded by the release workflow.
const (
	AssetSetup     = "OPTIC-windows-amd64-setup.exe"
	AssetPortable  = "OPTIC-windows-amd64-portable.exe"
	AssetDMG       = "OPTIC-macos-universal.dmg"
	AssetTarball   = "OPTIC-linux-amd64.tar.gz"
	AssetTarball41 = "OPTIC-linux-amd64-webkit41.tar.gz" // built with -tags webkit2_41
)

const (
	repoURL     = "https://github.com/andredeomondes/optic"
	appID       = "io.github.andredeomondes.optic"
	wingetID    = "andredeomondes.OPTIC"
	publisher   = "andredeomondes"
	descEN      = "Control your TV from your PC"
	descPT      = "Controle sua TV pelo computador"
	sumsAsset   = "SHA256SUMS"
	licenseName = "MIT"
)

//go:embed all:templates
var templatesFS embed.FS

// Input is everything needed to render one release.
type Input struct {
	Version string            // "1.2.3" or "v1.2.3"
	Sums    map[string]string // asset name -> sha256 hex
	Date    string            // YYYY-MM-DD, used in the Flatpak metainfo
}

// output maps a template to its path inside the output directory.
type output struct{ tmpl, path string }

var outputs = []output{
	{"winget/version.yaml.tmpl", "winget/andredeomondes.OPTIC.yaml"},
	{"winget/installer.yaml.tmpl", "winget/andredeomondes.OPTIC.installer.yaml"},
	{"winget/locale.en-US.yaml.tmpl", "winget/andredeomondes.OPTIC.locale.en-US.yaml"},
	{"winget/locale.pt-BR.yaml.tmpl", "winget/andredeomondes.OPTIC.locale.pt-BR.yaml"},
	{"scoop/optic.json.tmpl", "scoop/optic.json"},
	{"homebrew/optic.rb.tmpl", "homebrew/optic.rb"},
	{"aur/PKGBUILD.tmpl", "aur/PKGBUILD"},
	{"aur/SRCINFO.tmpl", "aur/.SRCINFO"},
	{"flatpak/app.yml.tmpl", "flatpak/" + appID + ".yml"},
	{"flatpak/metainfo.xml.tmpl", "flatpak/" + appID + ".metainfo.xml"},
}

var (
	versionRe = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)
	hexRe     = regexp.MustCompile(`^[0-9a-fA-F]{64}$`)
)

// ParseSums reads sha256sum output ("<hex>  <name>" or "<hex> *<name>").
// Hashes are returned lowercase. Blank lines are ignored.
func ParseSums(r io.Reader) (map[string]string, error) {
	sums := map[string]string{}
	sc := bufio.NewScanner(r)
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		sum, rest, ok := strings.Cut(line, " ")
		if !ok {
			return nil, fmt.Errorf("sums line %d: expected \"<sha256>  <name>\"", n)
		}
		if !hexRe.MatchString(sum) {
			return nil, fmt.Errorf("sums line %d: %q is not a 64-character hex sha256", n, sum)
		}
		if rest == "" || (rest[0] != ' ' && rest[0] != '*') {
			return nil, fmt.Errorf("sums line %d: expected \"<sha256>  <name>\"", n)
		}
		name := strings.TrimSpace(rest[1:])
		if name == "" {
			return nil, fmt.Errorf("sums line %d: missing file name", n)
		}
		sums[name] = strings.ToLower(sum)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return sums, nil
}

// data is what the templates see.
type data struct {
	Version     string
	Tag         string
	Date        string
	AppID       string
	WingetID    string
	Publisher   string
	License     string
	Homepage    string
	DescEN      string
	DescPT      string
	DownloadURL string // https://.../releases/download/vX.Y.Z
	SumsURL     string
	Setup       asset
	Portable    asset
	DMG         asset
	Tarball     asset
	Tarball41   asset
}

type asset struct {
	Name   string
	URL    string
	SHA256 string // lowercase
}

func (a asset) SHA256Upper() string { return strings.ToUpper(a.SHA256) }

func newData(in Input) (*data, error) {
	v := strings.TrimPrefix(in.Version, "v")
	if !versionRe.MatchString(v) {
		return nil, fmt.Errorf("invalid version %q: want X.Y.Z or vX.Y.Z", in.Version)
	}
	if _, err := time.Parse("2006-01-02", in.Date); err != nil {
		return nil, fmt.Errorf("invalid date %q: want YYYY-MM-DD", in.Date)
	}
	base := repoURL + "/releases/download/v" + v
	d := &data{
		Version: v, Tag: "v" + v, Date: in.Date,
		AppID: appID, WingetID: wingetID, Publisher: publisher, License: licenseName,
		Homepage: repoURL, DescEN: descEN, DescPT: descPT,
		DownloadURL: base, SumsURL: base + "/" + sumsAsset,
	}
	for _, a := range []struct {
		dst  *asset
		name string
	}{{&d.Setup, AssetSetup}, {&d.Portable, AssetPortable}, {&d.DMG, AssetDMG}, {&d.Tarball, AssetTarball}, {&d.Tarball41, AssetTarball41}} {
		sum, ok := in.Sums[a.name]
		if !ok {
			return nil, fmt.Errorf("missing checksum for %s", a.name)
		}
		if !hexRe.MatchString(sum) {
			return nil, fmt.Errorf("checksum for %s is not a 64-character hex sha256", a.name)
		}
		*a.dst = asset{Name: a.name, URL: base + "/" + a.name, SHA256: strings.ToLower(sum)}
	}
	return d, nil
}

// Render writes every manifest under outDir and returns the written paths.
// Everything is rendered in memory first; on any error nothing is written.
func Render(in Input, outDir string) ([]string, error) {
	d, err := newData(in)
	if err != nil {
		return nil, err
	}
	rendered := make([][]byte, len(outputs))
	for i, o := range outputs {
		src, err := templatesFS.ReadFile(path.Join("templates", o.tmpl))
		if err != nil {
			return nil, err
		}
		t, err := template.New(o.tmpl).Parse(strings.ReplaceAll(string(src), "\r\n", "\n")) // checkouts may convert to CRLF
		if err != nil {
			return nil, fmt.Errorf("%s: %w", o.tmpl, err)
		}
		var buf bytes.Buffer
		if err := t.Execute(&buf, d); err != nil {
			return nil, fmt.Errorf("%s: %w", o.tmpl, err)
		}
		rendered[i] = buf.Bytes()
	}

	written := make([]string, 0, len(outputs))
	for i, o := range outputs {
		p := filepath.Join(outDir, filepath.FromSlash(o.path))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return written, err
		}
		if err := os.WriteFile(p, rendered[i], 0o644); err != nil {
			return written, err
		}
		written = append(written, p)
	}
	return written, nil
}
