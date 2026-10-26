package pkgmeta

import (
	"encoding/json"
	"encoding/xml"
	"flag"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Regenerate goldens with: go test ./internal/pkgmeta -update
var update = flag.Bool("update", false, "regenerate golden files (removes stale ones)")

const (
	hexSetup    = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	hexPortable = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	hexDMG      = "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	hexTar41    = "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
	hexTar      = "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
)

func fakeSums() map[string]string {
	return map[string]string{
		AssetSetup:     hexSetup,
		AssetPortable:  hexPortable,
		AssetDMG:       hexDMG,
		AssetTarball:   hexTar,
		AssetTarball41: hexTar41,
	}
}

func fakeInput() Input {
	return Input{Version: "v1.2.3", Sums: fakeSums(), Date: "2026-01-02"}
}

func TestParseSums(t *testing.T) {
	text := hexSetup + "  OPTIC-windows-amd64-setup.exe\n" +
		"\n" +
		strings.ToUpper(hexDMG) + " *OPTIC-macos-universal.dmg\r\n" +
		"   \n"
	got, err := ParseSums(strings.NewReader(text))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 entries, got %v", got)
	}
	if got[AssetSetup] != hexSetup {
		t.Errorf("text-mode entry: %q", got[AssetSetup])
	}
	if got[AssetDMG] != hexDMG {
		t.Errorf("binary-mode entry must be lowercased: %q", got[AssetDMG])
	}
}

func TestParseSumsErrors(t *testing.T) {
	bad := map[string]string{
		"short hex":  "abc  file.exe\n",
		"non hex":    strings.Repeat("z", 64) + "  file.exe\n",
		"no name":    hexSetup + "\n",
		"no spacing": hexSetup + "file.exe\n",
	}
	for name, in := range bad {
		if _, err := ParseSums(strings.NewReader(in)); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}

func TestRenderGolden(t *testing.T) {
	out := t.TempDir()
	written, err := Render(fakeInput(), out)
	if err != nil {
		t.Fatal(err)
	}
	if len(written) != 10 {
		t.Fatalf("want 10 files, got %d: %v", len(written), written)
	}
	if *update {
		if err := os.RemoveAll(filepath.Join("testdata", "golden")); err != nil {
			t.Fatal(err)
		}
	}
	for _, p := range written {
		rel, err := filepath.Rel(out, p)
		if err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		golden := filepath.Join("testdata", "golden", rel)
		if *update {
			if err := os.MkdirAll(filepath.Dir(golden), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(golden, got, 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		want, err := os.ReadFile(golden)
		if err != nil {
			t.Fatalf("%s: %v (run with -update)", rel, err)
		}
		if string(got) != strings.ReplaceAll(string(want), "\r\n", "\n") { // goldens may be CRLF after checkout
			t.Errorf("%s differs from golden\n--- got ---\n%s\n--- want ---\n%s", rel, got, want)
		}
	}
}

func TestGoldenSetMatchesRenderedSet(t *testing.T) {
	out := renderTemp(t)
	list := func(root string) []string {
		var files []string
		err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err == nil && !d.IsDir() {
				rel, _ := filepath.Rel(root, p)
				files = append(files, filepath.ToSlash(rel))
			}
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		return files
	}
	got, want := list(out), list(filepath.Join("testdata", "golden"))
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("golden set differs from rendered set:\nrendered: %v\ngolden:   %v", got, want)
	}
}

func TestWingetScopeIsMachine(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(renderTemp(t), "winget", "andredeomondes.OPTIC.installer.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "Scope: machine") {
		t.Errorf("installer is per-machine:\n%s", data)
	}
}

func TestWingetInstallerEntries(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(renderTemp(t), "winget", "andredeomondes.OPTIC.installer.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"ProductCode: OPTICOPTIC", "AppsAndFeaturesEntries:", "Publisher: OPTIC", "DisplayName: OPTIC", "ElevationRequirement: elevatesSelf"} {
		if !strings.Contains(string(data), want) {
			t.Errorf("installer missing %q: %s", want, data)
		}
	}
}

func TestLinuxManifestsUseWebkit41(t *testing.T) {
	out := renderTemp(t)
	for rel, want := range map[string]string{
		"aur/PKGBUILD": "webkit2gtk-4.1",
		"aur/.SRCINFO": "webkit2gtk-4.1",
	} {
		data, _ := os.ReadFile(filepath.Join(out, filepath.FromSlash(rel)))
		if !strings.Contains(string(data), want) || !strings.Contains(string(data), AssetTarball41) {
			t.Errorf("%s must use the webkit 4.1 tarball and dependency", rel)
		}
	}
	data, _ := os.ReadFile(filepath.Join(out, "aur", "PKGBUILD"))
	if !strings.Contains(string(data), "options=('!strip')") {
		t.Error("PKGBUILD needs options=('!strip')")
	}
	fp, _ := os.ReadFile(filepath.Join(out, "flatpak", "io.github.andredeomondes.optic.yml"))
	if !strings.Contains(string(fp), AssetTarball41) || !strings.Contains(string(fp), "runtime-version: '50'") {
		t.Errorf("flatpak must use the webkit41 tarball and runtime 50: %s", fp)
	}
}

func TestRenderMissingAssetWritesNothing(t *testing.T) {
	for _, asset := range []string{AssetSetup, AssetPortable, AssetDMG, AssetTarball, AssetTarball41} {
		in := fakeInput()
		delete(in.Sums, asset)
		out := filepath.Join(t.TempDir(), "out")
		if _, err := Render(in, out); err == nil || !strings.Contains(err.Error(), asset) {
			t.Fatalf("missing %s: want error naming the asset, got %v", asset, err)
		}
		if _, err := os.Stat(out); !os.IsNotExist(err) {
			t.Errorf("missing %s: outDir must be untouched, stat err=%v", asset, err)
		}
	}
}

func TestRenderInvalidVersion(t *testing.T) {
	for _, v := range []string{"", "v", "1.2", "1.2.3.4", "v1.2.x", "1.2.3-beta", "../1.2.3", "1.2.3 "} {
		in := fakeInput()
		in.Version = v
		out := filepath.Join(t.TempDir(), "out")
		if _, err := Render(in, out); err == nil {
			t.Errorf("version %q: expected error", v)
		}
		if _, err := os.Stat(out); !os.IsNotExist(err) {
			t.Errorf("version %q: outDir must be untouched", v)
		}
	}
}

func TestRenderInvalidDate(t *testing.T) {
	in := fakeInput()
	in.Date = "02/01/2026"
	if _, err := Render(in, filepath.Join(t.TempDir(), "out")); err == nil {
		t.Fatal("expected error for bad date")
	}
}

func TestVersionWithoutPrefix(t *testing.T) {
	a, b := t.TempDir(), t.TempDir()
	in := fakeInput()
	if _, err := Render(in, a); err != nil {
		t.Fatal(err)
	}
	in.Version = "1.2.3"
	if _, err := Render(in, b); err != nil {
		t.Fatal(err)
	}
	x, _ := os.ReadFile(filepath.Join(a, "scoop", "optic.json"))
	y, _ := os.ReadFile(filepath.Join(b, "scoop", "optic.json"))
	if string(x) != string(y) {
		t.Error("v-prefixed and bare versions must render identically")
	}
}

func renderTemp(t *testing.T) string {
	t.Helper()
	out := t.TempDir()
	if _, err := Render(fakeInput(), out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestScoopIsValidJSON(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(renderTemp(t), "scoop", "optic.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("scoop manifest is not valid JSON: %v", err)
	}
	if m["version"] != "1.2.3" {
		t.Errorf("version = %v", m["version"])
	}
}

func TestMetainfoIsValidXML(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(renderTemp(t), "flatpak", "io.github.andredeomondes.optic.metainfo.xml"))
	if err != nil {
		t.Fatal(err)
	}
	var c struct {
		ID      string `xml:"id"`
		DevName string `xml:"developer>name"`
		Shots   []struct {
			Type    string `xml:"type,attr"`
			Caption string `xml:"caption"`
			Image   string `xml:"image"`
		} `xml:"screenshots>screenshot"`
		Releases []struct {
			Version string `xml:"version,attr"`
			Date    string `xml:"date,attr"`
		} `xml:"releases>release"`
	}
	if err := xml.Unmarshal(data, &c); err != nil {
		t.Fatalf("metainfo is not valid XML: %v", err)
	}
	if c.ID != "io.github.andredeomondes.optic" {
		t.Errorf("id = %q", c.ID)
	}
	if c.DevName != "andredeomondes" {
		t.Errorf("developer name = %q", c.DevName)
	}
	if len(c.Shots) != 4 || c.Shots[0].Type != "default" || c.Shots[0].Caption == "" ||
		c.Shots[0].Image != "https://raw.githubusercontent.com/andredeomondes/optic/v1.2.3/docs/screenshots/main.png" {
		t.Errorf("screenshots = %+v", c.Shots)
	}
	if len(c.Releases) != 1 || c.Releases[0].Version != "1.2.3" || c.Releases[0].Date != "2026-01-02" {
		t.Errorf("releases = %+v", c.Releases)
	}
}

func TestWingetHashUppercase(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(renderTemp(t), "winget", "andredeomondes.OPTIC.installer.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !strings.Contains(s, "InstallerSha256: "+strings.ToUpper(hexSetup)) {
		t.Errorf("installer hash must be uppercase:\n%s", s)
	}
	if strings.Contains(s, hexSetup) {
		t.Error("lowercase hash leaked into winget installer manifest")
	}
}

func TestChecksumsLandInManifests(t *testing.T) {
	out := renderTemp(t)
	want := map[string]string{
		"scoop/optic.json":  hexPortable,
		"homebrew/optic.rb": hexDMG,
		"aur/PKGBUILD":      hexTar41,
		"aur/.SRCINFO":      hexTar41,
		"flatpak/io.github.andredeomondes.optic.yml": hexTar41,
	}
	for rel, hash := range want {
		data, err := os.ReadFile(filepath.Join(out, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), hash) {
			t.Errorf("%s does not contain its sha256", rel)
		}
	}
}
