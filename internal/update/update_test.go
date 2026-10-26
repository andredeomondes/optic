package update

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"optic/internal/domain"
)

func TestChoose(t *testing.T) {
	pf := []string{`C:\Program Files`, `C:\Program Files (x86)`}
	cases := []struct {
		goos, exe string
		pf        []string
		wk41      bool
		want      Target
	}{
		{"windows", `C:\Program Files\OPTIC\OPTIC.exe`, pf, false, Target{KindInstaller, AssetWindowsSetup}},
		{"windows", `c:\PROGRAM FILES (X86)\optic\optic.exe`, pf, false, Target{KindInstaller, AssetWindowsSetup}},
		{"windows", `C:/Program Files/OPTIC/OPTIC.exe`, []string{`C:\Program Files\`}, false, Target{KindInstaller, AssetWindowsSetup}},
		{"windows", `C:\Program Files Extra\OPTIC.exe`, pf, false, Target{KindPortable, AssetWindowsPortable}},
		{"windows", `C:\Program Files\..\Users\me\OPTIC.exe`, pf, false, Target{KindPortable, AssetWindowsPortable}},
		{"windows", `C:\Program Files\OPTIC\.\OPTIC.exe`, pf, false, Target{KindInstaller, AssetWindowsSetup}},
		{"windows", `C:\Users\..\Program Files\OPTIC\OPTIC.exe`, pf, false, Target{KindInstaller, AssetWindowsSetup}},
		{"windows", `C:\Users\me\Downloads\OPTIC.exe`, pf, false, Target{KindPortable, AssetWindowsPortable}},
		{"windows", `C:\Users\me\OPTIC.exe`, []string{"", " "}, false, Target{KindPortable, AssetWindowsPortable}},
		{"darwin", "/Applications/OPTIC.app/Contents/MacOS/OPTIC", nil, false, Target{KindDMG, AssetMacOS}},
		{"linux", "/usr/local/bin/optic", nil, false, Target{KindTarball, AssetLinux}},
		{"linux", "/usr/local/bin/optic", nil, true, Target{KindTarball, AssetLinuxWebkit41}},
	}
	for _, c := range cases {
		got, err := Choose(c.goos, "amd64", c.exe, c.pf, c.wk41)
		if err != nil || got != c.want {
			t.Errorf("Choose(%q, %q, %v) = %v, %v; want %v", c.goos, c.exe, c.wk41, got, err, c.want)
		}
	}
	if _, err := Choose("freebsd", "amd64", "/usr/bin/optic", nil, false); !errors.Is(err, ErrUnsupported) {
		t.Errorf("freebsd: err = %v, want ErrUnsupported", err)
	}
	for _, c := range []struct{ goos, goarch string }{{"windows", "arm64"}, {"windows", "386"}, {"linux", "arm64"}} {
		if _, err := Choose(c.goos, c.goarch, "/x/optic", nil, false); !errors.Is(err, ErrUnsupported) {
			t.Errorf("%s/%s: err = %v, want ErrUnsupported", c.goos, c.goarch, err)
		}
	}
	if got, err := Choose("darwin", "arm64", "/Applications/OPTIC.app/Contents/MacOS/OPTIC", nil, false); err != nil || got.Asset != AssetMacOS {
		t.Errorf("darwin/arm64 = %v, %v; want the universal dmg", got, err)
	}
}

const testAsset = AssetLinux

// fixture is a local TLS "GitHub" serving one release.
type fixture struct {
	srv     *httptest.Server
	payload []byte
	sums    string
	dl      Downloader
	rel     domain.Release
	dir     string
	prefix  string
	handler map[string]http.HandlerFunc // extra routes
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	fx := &fixture{payload: bytes.Repeat([]byte("optic"), 50_000), handler: map[string]http.HandlerFunc{}}
	sum := sha256.Sum256(fx.payload)
	fx.sums = hex.EncodeToString(sum[:]) + "  " + testAsset + "\n" + strings.Repeat("0", 64) + "  OPTIC-macos-universal.dmg\n"
	base := "/andredeomondes/optic/releases/download/v1.1.0/"
	fx.srv = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h, ok := fx.handler[r.URL.Path]; ok {
			h(w, r)
			return
		}
		switch r.URL.Path {
		case base + AssetSums:
			fmt.Fprint(w, fx.sums)
		case base + testAsset:
			// GitHub redirects assets to a storage host with a signed query.
			http.Redirect(w, r, "/storage/blob?sig=secret", http.StatusFound)
		case "/storage/blob":
			w.Write(fx.payload)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(fx.srv.Close)
	fx.prefix = fx.srv.URL + "/andredeomondes/optic/releases/download/"
	fx.dl = Downloader{HTTP: fx.srv.Client(), AllowedHosts: []string{"127.0.0.1"}, Prefix: fx.prefix}
	fx.rel = domain.Release{Tag: "v1.1.0", Assets: []domain.Asset{
		{Name: testAsset, URL: fx.srv.URL + base + testAsset, Size: int64(len(fx.payload))},
		{Name: AssetSums, URL: fx.srv.URL + base + AssetSums, Size: int64(len(fx.sums))},
	}}
	fx.dir = filepath.Join(t.TempDir(), "updates", "1.1.0")
	return fx
}

func (fx *fixture) fetch(ctx context.Context, progress func(got, total int64)) (string, error) {
	return fx.dl.Fetch(ctx, fx.rel, testAsset, fx.dir, progress)
}

// assertNoFiles checks that neither the asset nor any temp .part file is left behind.
func (fx *fixture) assertNoFiles(t *testing.T) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(fx.dir, testAsset)); !os.IsNotExist(err) {
		t.Errorf("%s left behind (stat err %v)", testAsset, err)
	}
	if left, _ := filepath.Glob(filepath.Join(fx.dir, "*.part")); len(left) != 0 {
		t.Errorf("temp files left behind: %v", left)
	}
}

func TestFetchHappyPath(t *testing.T) {
	fx := newFixture(t)
	var last, total int64
	calls := 0
	path, err := fx.fetch(context.Background(), func(got, tot int64) {
		if got < last {
			t.Errorf("progress went backwards: %d after %d", got, last)
		}
		last, total = got, tot
		calls++
	})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if path != filepath.Join(fx.dir, testAsset) {
		t.Errorf("path = %q", path)
	}
	b, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(b, fx.payload) {
		t.Fatalf("file content wrong (err %v, %d bytes)", err, len(b))
	}
	if calls == 0 || last != int64(len(fx.payload)) || total != int64(len(fx.payload)) {
		t.Errorf("progress calls=%d last=%d total=%d, want last=total=%d", calls, last, total, len(fx.payload))
	}
	if left, _ := filepath.Glob(filepath.Join(fx.dir, "*.part")); len(left) != 0 {
		t.Errorf("temp files left behind: %v", left)
	}
}

func TestFetchChecksumMismatchDeletes(t *testing.T) {
	fx := newFixture(t)
	fx.sums = strings.Repeat("ab", 32) + "  " + testAsset + "\n"
	if _, err := fx.fetch(context.Background(), nil); !errors.Is(err, ErrChecksum) {
		t.Fatalf("err = %v, want ErrChecksum", err)
	}
	fx.assertNoFiles(t)
}

func TestFetchMissingSumsLine(t *testing.T) {
	fx := newFixture(t)
	fx.sums = strings.Repeat("0", 64) + "  OPTIC-macos-universal.dmg\n"
	if _, err := fx.fetch(context.Background(), nil); !errors.Is(err, ErrNoChecksum) {
		t.Fatalf("err = %v, want ErrNoChecksum", err)
	}
	fx.assertNoFiles(t)
}

func TestFetchMissingAssets(t *testing.T) {
	fx := newFixture(t)
	noSums := fx.rel
	noSums.Assets = fx.rel.Assets[:1]
	if _, err := fx.dl.Fetch(context.Background(), noSums, testAsset, fx.dir, nil); !errors.Is(err, ErrNoChecksum) {
		t.Errorf("no SHA256SUMS: err = %v, want ErrNoChecksum", err)
	}
	if _, err := fx.dl.Fetch(context.Background(), fx.rel, AssetMacOS, fx.dir, nil); !errors.Is(err, ErrNoAsset) {
		t.Errorf("missing asset: err = %v, want ErrNoAsset", err)
	}
	if _, err := fx.dl.Fetch(context.Background(), fx.rel, "../"+testAsset, fx.dir, nil); !errors.Is(err, ErrNoAsset) {
		t.Errorf("path in asset name: err = %v, want ErrNoAsset", err)
	}
}

func TestFetchSizeCap(t *testing.T) {
	fx := newFixture(t)
	fx.dl.MaxSize = 1024
	fx.rel.Assets[0].Size = 0 // only MaxSize applies
	if _, err := fx.fetch(context.Background(), nil); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("err = %v, want ErrTooLarge", err)
	}
	fx.assertNoFiles(t)

	// Without Content-Length the cap is enforced while streaming.
	fx.handler["/storage/blob"] = func(w http.ResponseWriter, r *http.Request) {
		for i := 0; i < 4; i++ {
			w.Write(fx.payload[:1000])
			w.(http.Flusher).Flush()
		}
	}
	if _, err := fx.fetch(context.Background(), nil); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("streamed: err = %v, want ErrTooLarge", err)
	}
	fx.assertNoFiles(t)
}

func TestFetchDeclaredSize(t *testing.T) {
	fx := newFixture(t)
	fx.rel.Assets[0].Size = int64(len(fx.payload)) - 1 // body is bigger than declared
	if _, err := fx.fetch(context.Background(), nil); !errors.Is(err, ErrTooLarge) {
		t.Errorf("bigger: err = %v, want ErrTooLarge", err)
	}
	fx.assertNoFiles(t)
	fx.rel.Assets[0].Size = int64(len(fx.payload)) + 1 // body is smaller than declared
	if _, err := fx.fetch(context.Background(), nil); !errors.Is(err, ErrChecksum) {
		t.Errorf("smaller: err = %v, want ErrChecksum", err)
	}
	fx.assertNoFiles(t)
}

func TestFetchSumsTooLarge(t *testing.T) {
	fx := newFixture(t)
	fx.sums = strings.Repeat("x", int(MaxSumsSize)+1)
	if _, err := fx.fetch(context.Background(), nil); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("err = %v, want ErrTooLarge", err)
	}
}

func TestFetchRedirectToForeignHostRefused(t *testing.T) {
	fx := newFixture(t)
	var hits atomic.Int32
	foreign := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Write(fx.payload)
	}))
	defer foreign.Close()
	// Same machine, different host name: only the name decides.
	to := strings.Replace(foreign.URL, "127.0.0.1", "localhost", 1) + "/evil?sig=secret"
	// The test cert does not cover "localhost"; skip verification so a TLS
	// failure cannot hide a missing host check.
	tr := fx.srv.Client().Transport.(*http.Transport).Clone()
	tr.TLSClientConfig.InsecureSkipVerify = true
	fx.dl.HTTP = &http.Client{Transport: tr}
	fx.handler["/storage/blob"] = func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, to, http.StatusFound)
	}
	_, err := fx.fetch(context.Background(), nil)
	if !errors.Is(err, ErrNetwork) {
		t.Fatalf("err = %v, want ErrNetwork", err)
	}
	if strings.Contains(err.Error(), "sig=") || strings.Contains(err.Error(), "http") {
		t.Errorf("error leaks a URL: %v", err)
	}
	if hits.Load() != 0 {
		t.Errorf("foreign host was contacted %d times", hits.Load())
	}
	fx.assertNoFiles(t)
}

func TestFetchPlainHTTPRefused(t *testing.T) {
	fx := newFixture(t)
	var hits atomic.Int32
	plain := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Write(fx.payload)
	}))
	defer plain.Close()

	// Redirect hop to http on an allowed host.
	fx.handler["/storage/blob"] = func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, plain.URL+"/blob", http.StatusFound)
	}
	if _, err := fx.fetch(context.Background(), nil); !errors.Is(err, ErrNetwork) {
		t.Errorf("http redirect: err = %v, want ErrNetwork", err)
	}
	fx.assertNoFiles(t)

	// Initial URL on http, even with a matching prefix.
	httpPrefix := plain.URL + "/andredeomondes/optic/releases/download/"
	dl := Downloader{HTTP: fx.srv.Client(), AllowedHosts: []string{"127.0.0.1"}, Prefix: httpPrefix}
	rel := domain.Release{Assets: []domain.Asset{
		{Name: testAsset, URL: httpPrefix + "v1.1.0/" + testAsset},
		{Name: AssetSums, URL: httpPrefix + "v1.1.0/" + AssetSums},
	}}
	if _, err := dl.Fetch(context.Background(), rel, testAsset, fx.dir, nil); !errors.Is(err, ErrNetwork) {
		t.Errorf("http start: err = %v, want ErrNetwork", err)
	}
	if hits.Load() != 0 {
		t.Errorf("plain http server was contacted %d times", hits.Load())
	}
}

func TestFetchRefusesURLOutsidePrefix(t *testing.T) {
	fx := newFixture(t)
	fx.rel.Assets[0].URL = fx.srv.URL + "/someone/else/releases/download/v1/" + testAsset
	if _, err := fx.fetch(context.Background(), nil); !errors.Is(err, ErrNetwork) {
		t.Errorf("err = %v, want ErrNetwork", err)
	}
	// The real defaults only accept the project's GitHub release path.
	var d Downloader
	if d.allowedStart("https://github.com/someone/optic/releases/download/v1/x") ||
		d.allowedStart("http://github.com/andredeomondes/optic/releases/download/v1/x") ||
		!d.allowedStart(ReleasePrefix+"v1.1.0/"+testAsset) {
		t.Errorf("default prefix check is wrong")
	}
}

func TestFetchCancelRemovesPart(t *testing.T) {
	fx := newFixture(t)
	fx.handler["/storage/blob"] = func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprint(len(fx.payload)))
		w.Write(fx.payload[:4096])
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, err := fx.fetch(ctx, func(got, total int64) { cancel() })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	fx.assertNoFiles(t)
}

func TestSumFor(t *testing.T) {
	h := strings.Repeat("Ab", 32)
	list := []byte("junk line\n" + h + " *" + testAsset + "\r\n")
	if got, ok := sumFor(list, testAsset); !ok || got != strings.ToLower(h) {
		t.Errorf("binary-mode line: %q %v", got, ok)
	}
	if _, ok := sumFor(list, AssetMacOS); ok {
		t.Errorf("found a sum for a missing name")
	}
}

func TestFetchConcurrentNeverPublishesUnverifiedBytes(t *testing.T) {
	fx := newFixture(t)
	bad := bytes.Repeat([]byte("evil!"), len(fx.payload)/5)
	badPath := "/andredeomondes/optic/releases/download/v1.1.0-bad/" + testAsset
	goodDone := make(chan struct{})
	fx.handler[badPath] = func(w http.ResponseWriter, r *http.Request) {
		w.Write(bad[:len(bad)/2])
		w.(http.Flusher).Flush()
		<-goodDone
		w.Write(bad[len(bad)/2:])
	}
	badRel := fx.rel
	badRel.Assets = []domain.Asset{
		{Name: testAsset, URL: fx.srv.URL + badPath, Size: int64(len(bad))},
		fx.rel.Assets[1],
	}

	started := make(chan struct{})
	var once sync.Once
	badErr := make(chan error, 1)
	go func() {
		_, err := fx.dl.Fetch(context.Background(), badRel, testAsset, fx.dir, func(got, total int64) {
			once.Do(func() { close(started) })
		})
		badErr <- err
	}()
	<-started
	path, err := fx.fetch(context.Background(), nil)
	close(goodDone)
	if err != nil {
		t.Fatalf("good Fetch: %v", err)
	}
	if err := <-badErr; !errors.Is(err, ErrChecksum) {
		t.Errorf("bad Fetch: err = %v, want ErrChecksum", err)
	}
	b, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(b, fx.payload) {
		t.Fatalf("final file holds unverified bytes (err %v, %d bytes)", err, len(b))
	}
	if left, _ := filepath.Glob(filepath.Join(fx.dir, "*.part")); len(left) != 0 {
		t.Errorf("temp files left behind: %v", left)
	}
}

func TestFetchRefusesUnknownAssetNames(t *testing.T) {
	fx := newFixture(t)
	names := []string{"..", ".", "", testAsset + ":evil", "x/" + testAsset, `x\` + testAsset, AssetSums, "OPTIC.exe"}
	for _, n := range names {
		rel := fx.rel
		rel.Assets = append([]domain.Asset{{Name: n, URL: fx.rel.Assets[0].URL}}, fx.rel.Assets...)
		fx.sums += strings.Repeat("0", 64) + "  " + n + "\n"
		if _, err := fx.dl.Fetch(context.Background(), rel, n, fx.dir, nil); !errors.Is(err, ErrNoAsset) {
			t.Errorf("asset %q: err = %v, want ErrNoAsset", n, err)
		}
	}
}

func TestFetchConflictingSumsLines(t *testing.T) {
	fx := newFixture(t)
	good := fx.sums
	fx.sums = good + strings.Repeat("ab", 32) + "  " + testAsset + "\n"
	if _, err := fx.fetch(context.Background(), nil); !errors.Is(err, ErrNoChecksum) {
		t.Errorf("conflicting lines: err = %v, want ErrNoChecksum", err)
	}
	fx.assertNoFiles(t)
	fx.sums = good + good // identical duplicates are fine
	if _, err := fx.fetch(context.Background(), nil); err != nil {
		t.Errorf("identical duplicates: %v", err)
	}
}

func TestSumForExactName(t *testing.T) {
	list := []byte(strings.Repeat("1", 64) + "  " + testAsset + ".part\n")
	if _, ok := sumFor(list, testAsset); ok {
		t.Errorf("a .part line matched the asset")
	}
}

func TestRedirectHostVariants(t *testing.T) {
	var d Downloader
	cases := map[string]bool{
		"https://github.com/x":                               true,
		"https://GITHUB.COM/x":                               true,
		"https://Objects.GitHubUserContent.com/x?sig=1":      true,
		"https://release-assets.githubusercontent.com:443/x": true,
		"https://github.com.evil.com/x":                      false,
		"https://GITHUB.COM.EVIL.COM/x":                      false,
		"https://evilgithub.com/x":                           false,
		"https://github.com@evil.com/x":                      false,
		"http://github.com/x":                                false,
		"https://objects.githubusercontent.com.evil.com/x":   false,
	}
	for u, want := range cases {
		req, _ := http.NewRequest(http.MethodGet, u, nil)
		if got := d.checkRedirect(req, nil) == nil; got != want {
			t.Errorf("redirect to %s allowed=%v, want %v", u, got, want)
		}
	}
	for _, u := range []string{
		"https://github.com.evil.com/andredeomondes/optic/releases/download/v1/x",
		"https://github.com@evil.com/andredeomondes/optic/releases/download/v1/x",
		"https://GITHUB.COM/andredeomondes/optic/releases/download/v1/x",
	} {
		if d.allowedStart(u) {
			t.Errorf("start URL %s accepted", u)
		}
	}
}

func TestFetchStalledBodyAborts(t *testing.T) {
	fx := newFixture(t)
	fx.dl.stall = 200 * time.Millisecond
	fx.handler["/storage/blob"] = func(w http.ResponseWriter, r *http.Request) {
		w.Write(fx.payload[:1024])
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}
	done := make(chan error, 1)
	go func() { _, err := fx.fetch(context.Background(), nil); done <- err }()
	select {
	case err := <-done:
		if !errors.Is(err, ErrNetwork) {
			t.Errorf("err = %v, want ErrNetwork", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("stalled download was not aborted")
	}
	fx.assertNoFiles(t)
}

func TestFetchRefusesSymlinkFinal(t *testing.T) {
	fx := newFixture(t)
	if err := os.MkdirAll(fx.dir, 0o755); err != nil {
		t.Fatal(err)
	}
	victim := filepath.Join(t.TempDir(), "victim")
	os.WriteFile(victim, []byte("keep"), 0o644)
	if err := os.Symlink(victim, filepath.Join(fx.dir, testAsset)); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := fx.fetch(context.Background(), nil); err == nil {
		t.Fatal("Fetch wrote through a symlinked destination")
	}
	if b, _ := os.ReadFile(victim); string(b) != "keep" {
		t.Errorf("symlink target changed: %q", b)
	}
}

func TestFetchReplacesExistingFinal(t *testing.T) {
	fx := newFixture(t)
	os.MkdirAll(fx.dir, 0o755)
	os.WriteFile(filepath.Join(fx.dir, testAsset), []byte("old"), 0o644)
	path, err := fx.fetch(context.Background(), nil)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if b, _ := os.ReadFile(path); !bytes.Equal(b, fx.payload) {
		t.Errorf("old file not replaced")
	}
}

func TestFetchRefusesNonRegularFinal(t *testing.T) {
	fx := newFixture(t)
	if err := os.MkdirAll(filepath.Join(fx.dir, testAsset), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := fx.fetch(context.Background(), nil); err == nil {
		t.Fatal("Fetch replaced a directory at the destination")
	}
	if fi, err := os.Stat(filepath.Join(fx.dir, testAsset)); err != nil || !fi.IsDir() {
		t.Errorf("destination directory was touched (err %v)", err)
	}
	if left, _ := filepath.Glob(filepath.Join(fx.dir, "*.part")); len(left) != 0 {
		t.Errorf("temp files left behind: %v", left)
	}
}
