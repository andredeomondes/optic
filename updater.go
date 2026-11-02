package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"optic/internal/domain"
	"optic/internal/update"
)

// Events sent to the frontend while an update runs (see useUpdates.ts).
const (
	evUpdateProgress = "update:progress"
	evUpdateState    = "update:state"
)

// States of an update as the UI sees them.
const (
	updDownloading = "downloading"
	updVerifying   = "verifying"
	updInstalling  = "installing"
	updReady       = "ready"
	updError       = "error"
)

const (
	// updateTimeout bounds a whole update run (download + install start).
	updateTimeout = 15 * time.Minute
	// releaseMaxAge is how long the release from the last check is reused,
	// so an update normally costs one release lookup.
	releaseMaxAge = 10 * time.Minute
	// stopWait is how long shutdown waits for a cancelled download to clean up.
	stopWait = 2 * time.Second
	// loadWait is how long LastUpdateResult waits for startup to read the result.
	loadWait = 2 * time.Second
)

// errUpToDate is returned when the latest release is not newer than this one.
var errUpToDate = errors.New("Você já está na versão mais recente.")

// UpdateSupport says whether "Atualizar agora" can install the latest
// release here. Reason (pt-BR, translated by the UI) explains a false Auto.
type UpdateSupport struct {
	Auto   bool   `json:"auto"`
	Kind   string `json:"kind"`
	Reason string `json:"reason,omitempty"`
}

// UpdateProgress is the payload of update:progress.
type UpdateProgress struct {
	Got   int64 `json:"got"`
	Total int64 `json:"total"`
}

// UpdateState is the payload of update:state. Version is the release being
// installed, once known.
type UpdateState struct {
	State   string `json:"state"`
	Error   string `json:"error,omitempty"`
	Kind    string `json:"kind"`
	Version string `json:"version,omitempty"`
}

// updater runs the in-app update. Everything that touches the network, the
// disk or the window is a field so tests can swap it.
type updater struct {
	version      string // running version, from wails.json
	dir          string // <cfgDir>/updates
	goos, goarch string
	programFiles []string
	webkit41     bool
	pid          int
	exe          func() (string, error)

	latest func(ctx context.Context) (domain.Release, error)
	fetch  func(ctx context.Context, rel domain.Release, asset, dir string, progress func(got, total int64)) (string, string, error)
	apply  func(t update.Target, path, sha256, exe string, pid int, dir, version string) (bool, error)
	emit   func(event string, data any)
	// quit closes the TV session and exits; called only after a Windows
	// apply step has started.
	quit func()
	now  func() time.Time // nil → time.Now

	loadOnce sync.Once
	loaded   chan struct{} // closed once load has run; use loadedC

	mu      sync.Mutex
	running bool
	cancel  context.CancelFunc // of the running update
	done    chan struct{}      // closed when the running update returns
	last    *update.Result
	rel     domain.Release // from the last lookup
	relAt   time.Time
}

func (u *updater) clock() time.Time {
	if u.now != nil {
		return u.now()
	}
	return time.Now()
}

func (u *updater) loadedC() chan struct{} {
	u.loadOnce.Do(func() { u.loaded = make(chan struct{}) })
	return u.loaded
}

// load reads the outcome of the previous update and removes stale
// downloads. Called once at startup, before any update can run.
func (u *updater) load() {
	defer close(u.loadedC())
	r, err := update.ReadResult(u.dir)
	if err != nil {
		log.Printf("update result: %v", err)
	}
	// "ok" from the script but still the old version: the install did not take.
	if r != nil && r.OK && strings.TrimPrefix(r.Version, "v") != strings.TrimPrefix(u.version, "v") {
		r.OK = false
	}
	update.Cleanup(u.dir, u.version)
	u.mu.Lock()
	u.last = r
	u.mu.Unlock()
}

// lastResult returns the previous update's outcome once; later calls get nil.
// It waits briefly for load, so an early call from the UI is not lost.
func (u *updater) lastResult() *update.Result {
	select {
	case <-u.loadedC():
	case <-time.After(loadWait):
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	r := u.last
	u.last = nil
	return r
}

// latestRelease looks the latest release up and remembers it.
func (u *updater) latestRelease(ctx context.Context) (domain.Release, error) {
	rel, err := u.latest(ctx)
	if err == nil {
		u.mu.Lock()
		u.rel, u.relAt = rel, u.clock()
		u.mu.Unlock()
	}
	return rel, err
}

// release reuses the last looked-up release while it is recent.
func (u *updater) release(ctx context.Context) (domain.Release, error) {
	u.mu.Lock()
	rel, at := u.rel, u.relAt
	u.mu.Unlock()
	if !at.IsZero() && u.clock().Sub(at) < releaseMaxAge {
		return rel, nil
	}
	rel, err := u.latestRelease(ctx)
	if err != nil && !errors.Is(err, domain.ErrNoReleaseSource) {
		err = fmt.Errorf("%w (%v)", update.ErrNetwork, err)
	}
	return rel, err
}

func (u *updater) target() (update.Target, error) {
	exe, err := u.exe()
	if err != nil {
		exe = ""
	}
	return update.Choose(u.goos, u.goarch, exe, u.programFiles, u.webkit41)
}

// support checks the latest release for this install's asset and SHA256SUMS.
func (u *updater) support(ctx context.Context) UpdateSupport {
	t, err := u.target()
	if err != nil {
		return UpdateSupport{Reason: err.Error()}
	}
	s := UpdateSupport{Kind: string(t.Kind)}
	rel, err := u.release(ctx)
	if err != nil {
		s.Reason = userError(ctx, err)
		return s
	}
	var asset, sums bool
	for _, a := range rel.Assets {
		asset = asset || a.Name == t.Asset
		sums = sums || a.Name == update.AssetSums
	}
	switch {
	case !asset:
		s.Reason = update.ErrNoAsset.Error()
	case !sums:
		s.Reason = update.ErrNoChecksum.Error()
	default:
		s.Auto = true
	}
	return s
}

// start begins an update in the background. A second call while one runs
// does nothing; its progress keeps coming through the events.
func (u *updater) start(parent context.Context) {
	u.mu.Lock()
	if u.running {
		u.mu.Unlock()
		return
	}
	ctx, cancel := context.WithTimeout(parent, updateTimeout)
	done := make(chan struct{})
	u.running, u.cancel, u.done = true, cancel, done
	u.mu.Unlock()
	go func() {
		quit := u.run(ctx)
		cancel()
		u.mu.Lock()
		u.running = quit // stays true until the app exits
		u.cancel = nil
		u.mu.Unlock()
		close(done)
		if quit {
			u.quit() // closes the session and exits; may call stop
		}
	}()
}

// stop cancels a running update and waits briefly so the download can
// delete its temp file. Called on shutdown.
func (u *updater) stop() {
	u.mu.Lock()
	cancel, done := u.cancel, u.done
	u.mu.Unlock()
	if cancel == nil {
		return
	}
	cancel()
	select {
	case <-done:
	case <-time.After(stopWait):
	}
}

// run does one update and reports each step. It returns true when the app
// must exit so the Windows script can install.
func (u *updater) run(ctx context.Context) bool {
	kind, version := "", ""
	state := func(s string, err error) {
		st := UpdateState{State: s, Kind: kind, Version: version}
		if err != nil {
			st.Error = userError(ctx, err)
		}
		u.emit(evUpdateState, st)
	}
	fail := func(err error) bool { state(updError, err); return false }

	t, err := u.target()
	if err != nil {
		return fail(err)
	}
	kind = string(t.Kind)
	rel, err := u.release(ctx)
	if err != nil {
		return fail(err)
	}
	if !update.IsNewer(u.version, rel.Tag) {
		return fail(errUpToDate)
	}
	version = strings.TrimPrefix(rel.Tag, "v")
	dir := filepath.Join(u.dir, version)
	state(updDownloading, nil)

	gate := progressGate{}
	verifying := false
	path, sum, err := u.fetch(ctx, rel, t.Asset, dir, func(got, total int64) {
		if gate.allow(got, total, time.Now()) {
			u.emit(evUpdateProgress, UpdateProgress{Got: got, Total: total})
		}
		if total > 0 && got >= total && !verifying {
			verifying = true
			state(updVerifying, nil)
		}
	})
	if err != nil {
		return fail(err)
	}
	if !verifying {
		state(updVerifying, nil)
	}

	state(updInstalling, nil)
	exe, err := u.exe()
	if err != nil {
		return fail(err)
	}
	quit, err := u.apply(t, path, sum, exe, u.pid, u.dir, version)
	if err != nil {
		return fail(err)
	}
	if quit {
		return true
	}
	state(updReady, nil)
	return false
}

// userError turns an update failure into the text shown to the user: the
// known pt-BR sentinel without technical tails, a timeout or cancel as a
// network error. Anything else is logged and gives "", which the UI shows
// as a generic failure, so raw system text never reaches it.
func userError(ctx context.Context, err error) string {
	if errors.Is(err, update.ErrNoXdgOpen) {
		return err.Error() // keeps the folder to open by hand
	}
	for _, s := range []error{
		update.ErrNoChecksum, update.ErrNoAsset, update.ErrChecksum, update.ErrTooLarge,
		update.ErrNetwork, update.ErrUnsupported, update.ErrDest, update.ErrApplyBlocked,
		update.ErrExtract, errUpToDate, domain.ErrNoReleaseSource,
	} {
		if errors.Is(err, s) {
			return s.Error()
		}
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) || ctx.Err() != nil {
		return update.ErrNetwork.Error()
	}
	log.Printf("update: %v", err)
	return ""
}

// progressGate thins out progress events: one per whole percent or per
// 250 ms, plus the first and the final one.
type progressGate struct {
	at      time.Time
	percent int64
	sent    bool
}

func (g *progressGate) allow(got, total int64, now time.Time) bool {
	pct := int64(-1)
	if total > 0 {
		pct = got * 100 / total
	}
	ok := !g.sent ||
		(total > 0 && (got >= total || pct >= g.percent+1)) ||
		now.Sub(g.at) >= 250*time.Millisecond
	if ok {
		g.sent, g.at, g.percent = true, now, pct
	}
	return ok
}
