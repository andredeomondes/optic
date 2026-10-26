package update

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"optic/internal/domain"
)

const (
	// ReleasePrefix is the only place downloads may start from.
	ReleasePrefix = "https://github.com/andredeomondes/optic/releases/download/"
	// DefaultMaxSize caps an asset download.
	DefaultMaxSize int64 = 300 << 20
	// MaxSumsSize caps the SHA256SUMS download.
	MaxSumsSize int64 = 64 << 10
)

// DefaultAllowedHosts are the hosts a download may be redirected to.
var DefaultAllowedHosts = []string{"github.com", "objects.githubusercontent.com", "release-assets.githubusercontent.com"}

// Downloader fetches a release asset and verifies it. The zero value is ready
// to use; tests override the hosts and prefix to point at local TLS servers.
type Downloader struct {
	HTTP         *http.Client // nil → default client
	AllowedHosts []string     // nil → DefaultAllowedHosts
	Prefix       string       // "" → ReleasePrefix
	MaxSize      int64        // 0 → DefaultMaxSize

	stall time.Duration // 0 → DefaultStall; tests shorten it
}

// DefaultStall aborts a download when no bytes arrive for this long.
const DefaultStall = 60 * time.Second

var (
	errRedirect = errors.New("redirect refused")
	errStalled  = errors.New("download stalled")
	errDest     = errors.New("Não foi possível salvar o download nesta pasta.")
)

// Fetch downloads asset (one of the Asset* installable names) from rel into
// dir and returns its path once its SHA256 matches the release's SHA256SUMS.
// Bytes go to a unique temp file that is renamed into place only after the
// check, so the returned file always holds verified bytes. Cancel ctx to
// stop; a download with no bytes for DefaultStall is aborted as well.
func (d Downloader) Fetch(ctx context.Context, rel domain.Release, asset, dir string, progress func(got, total int64)) (string, error) {
	if !knownAsset(asset) {
		return "", ErrNoAsset
	}
	var target, sums *domain.Asset
	for i := range rel.Assets {
		switch rel.Assets[i].Name {
		case asset:
			target = &rel.Assets[i]
		case AssetSums:
			sums = &rel.Assets[i]
		}
	}
	if target == nil {
		return "", ErrNoAsset
	}
	if sums == nil {
		return "", ErrNoChecksum
	}
	if !d.allowedStart(target.URL) || !d.allowedStart(sums.URL) {
		return "", ErrNetwork
	}

	client := d.client()
	if d.HTTP == nil {
		defer client.CloseIdleConnections()
	}
	list, err := d.get(ctx, client, sums.URL, MaxSumsSize, nil)
	if err != nil {
		return "", err
	}
	want, ok := sumFor(list, asset)
	if !ok {
		return "", ErrNoChecksum
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	final := filepath.Join(dir, asset)
	f, err := os.CreateTemp(dir, asset+".*.part")
	if err != nil {
		return "", errDest
	}
	part := f.Name()
	h := sha256.New()
	limit := d.maxSize()
	if target.Size > 0 && target.Size < limit {
		limit = target.Size
	}
	var got int64
	_, err = d.get(ctx, client, target.URL, limit, func(body io.Reader, total int64) (int64, error) {
		if target.Size > 0 {
			total = target.Size
		}
		n, cerr := copyWithProgress(io.MultiWriter(f, h), body, total, progress)
		got = n
		return n, cerr
	})
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil && target.Size > 0 && got != target.Size {
		err = ErrChecksum
	}
	if err == nil && hex.EncodeToString(h.Sum(nil)) != want {
		err = ErrChecksum
	}
	if err == nil {
		err = replace(part, final)
	}
	if err != nil {
		os.Remove(part)
		return "", err
	}
	return final, nil
}

// replace moves the verified temp file to final, refusing to follow or
// replace anything but a regular file there.
func replace(tmp, final string) error {
	if fi, err := os.Lstat(final); err == nil {
		if !fi.Mode().IsRegular() {
			return errDest
		}
		if err := os.Remove(final); err != nil {
			return errDest
		}
	}
	if err := os.Rename(tmp, final); err != nil {
		return errDest
	}
	return nil
}

// get fetches u, reading at most limit bytes. With sink nil the body is
// returned; otherwise sink consumes it and nothing is returned.
func (d Downloader) get(parent context.Context, c *http.Client, u string, limit int64, sink func(io.Reader, int64) (int64, error)) ([]byte, error) {
	ctx, cancel := context.WithCancelCause(parent)
	defer cancel(nil)
	stall := d.stall
	if stall <= 0 {
		stall = DefaultStall
	}
	timer := time.AfterFunc(stall, func() { cancel(errStalled) })
	defer timer.Stop()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, ErrNetwork
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, d.netErr(ctx, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w (HTTP %d)", ErrNetwork, resp.StatusCode)
	}
	if resp.ContentLength > limit {
		return nil, ErrTooLarge
	}
	body := io.LimitReader(stallReader{resp.Body, timer, stall}, limit+1)
	if sink == nil {
		b, err := io.ReadAll(body)
		if err != nil {
			return nil, d.netErr(ctx, err)
		}
		if int64(len(b)) > limit {
			return nil, ErrTooLarge
		}
		return b, nil
	}
	n, err := sink(body, resp.ContentLength)
	if err != nil {
		return nil, d.netErr(ctx, err)
	}
	if n > limit {
		return nil, ErrTooLarge
	}
	return nil, nil
}

// netErr maps transport errors to ErrNetwork without leaking signed URLs.
// A cancelled caller context wins; a stall is a network error.
func (d Downloader) netErr(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		if cause := context.Cause(ctx); errors.Is(cause, errStalled) {
			return fmt.Errorf("%w (%v)", ErrNetwork, errStalled)
		}
		return ctx.Err()
	}
	var ue *url.Error
	if errors.As(err, &ue) {
		err = ue.Err
	}
	if errors.Is(err, errRedirect) {
		return fmt.Errorf("%w (%v)", ErrNetwork, errRedirect)
	}
	var we writeErr
	if errors.As(err, &we) {
		return we.error // disk problems are not network problems
	}
	return ErrNetwork
}

// stallReader pushes the stall deadline back on every read that returns data.
type stallReader struct {
	r     io.Reader
	t     *time.Timer
	stall time.Duration
}

func (s stallReader) Read(p []byte) (int, error) {
	n, err := s.r.Read(p)
	if n > 0 {
		s.t.Reset(s.stall)
	}
	return n, err
}

// writeErr marks a failure writing to disk.
type writeErr struct{ error }

func copyWithProgress(dst io.Writer, src io.Reader, total int64, progress func(got, total int64)) (int64, error) {
	buf := make([]byte, 64<<10)
	var got int64
	for {
		n, rerr := src.Read(buf)
		if n > 0 {
			if _, err := dst.Write(buf[:n]); err != nil {
				return got, writeErr{err}
			}
			got += int64(n)
			if progress != nil {
				progress(got, total)
			}
		}
		if rerr == io.EOF {
			return got, nil
		}
		if rerr != nil {
			return got, rerr
		}
	}
}

func (d Downloader) client() *http.Client {
	var c http.Client
	if d.HTTP != nil {
		c = *d.HTTP
	} else {
		c = http.Client{Transport: &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			TLSHandshakeTimeout:   15 * time.Second,
			ResponseHeaderTimeout: 30 * time.Second,
		}}
	}
	c.Jar = nil // no cookies across hosts
	c.CheckRedirect = d.checkRedirect
	return &c
}

func (d Downloader) checkRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= 10 || req.URL.Scheme != "https" || !d.hostAllowed(req.URL.Hostname()) {
		return errRedirect
	}
	return nil
}

func (d Downloader) allowedStart(u string) bool {
	prefix := d.Prefix
	if prefix == "" {
		prefix = ReleasePrefix
	}
	p, err := url.Parse(u)
	return err == nil && p.Scheme == "https" && strings.HasPrefix(u, prefix) && d.hostAllowed(p.Hostname())
}

func (d Downloader) hostAllowed(host string) bool {
	hosts := d.AllowedHosts
	if hosts == nil {
		hosts = DefaultAllowedHosts
	}
	for _, h := range hosts {
		if strings.EqualFold(h, host) {
			return true
		}
	}
	return false
}

func (d Downloader) maxSize() int64 {
	if d.MaxSize > 0 {
		return d.MaxSize
	}
	return DefaultMaxSize
}

// sumFor finds the sha256 for name in sha256sum output. Conflicting lines
// for the same name count as no checksum.
func sumFor(list []byte, name string) (string, bool) {
	found := ""
	sc := bufio.NewScanner(bytes.NewReader(list))
	for sc.Scan() {
		sum, rest, ok := strings.Cut(strings.TrimSpace(sc.Text()), " ")
		if !ok || len(sum) != 64 {
			continue
		}
		if _, err := hex.DecodeString(sum); err != nil {
			continue
		}
		if strings.TrimPrefix(strings.TrimSpace(rest), "*") != name {
			continue
		}
		sum = strings.ToLower(sum)
		if found != "" && found != sum {
			return "", false
		}
		found = sum
	}
	return found, found != ""
}
