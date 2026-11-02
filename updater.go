package main

import (
	"context"
	"errors"
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

// updateTimeout bounds a whole update run (download + install start).
const updateTimeout = 15 * time.Minute

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

// UpdateState is the payload of update:state.
type UpdateState struct {
	State string `json:"state"`
	Error string `json:"error,omitempty"`
	Kind  string `json:"kind"`
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

	mu      sync.Mutex
	running bool
	last    *update.Result
}

// load reads the outcome of the previous update and removes stale
// downloads. Called once at startup.
func (u *updater) load() {
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
func (u *updater) lastResult() *update.Result {
	u.mu.Lock()
	defer u.mu.Unlock()
	r := u.last
	u.last = nil
	return r
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
	rel, err := u.latest(ctx)
	if err != nil {
		s.Reason = err.Error()
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
	u.running = true
	u.mu.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(parent, updateTimeout)
		defer cancel()
		quit := u.run(ctx)
		if quit {
			u.quit() // the session is closed there; running stays true until exit
			return
		}
		u.mu.Lock()
		u.running = false
		u.mu.Unlock()
	}()
}

// run does one update and reports each step. It returns true when the app
// must exit so the Windows script can install.
func (u *updater) run(ctx context.Context) bool {
	kind := ""
	state := func(s string, err error) {
		st := UpdateState{State: s, Kind: kind}
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
	state(updDownloading, nil)
	rel, err := u.latest(ctx)
	if err != nil {
		return fail(err)
	}
	if !update.IsNewer(u.version, rel.Tag) {
		return fail(errUpToDate)
	}
	version := strings.TrimPrefix(rel.Tag, "v")
	dir := filepath.Join(u.dir, version)

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
// known pt-BR sentinel without technical tails, a timeout as a network
// error, anything else as it came.
func userError(ctx context.Context, err error) string {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) || ctx.Err() != nil {
		return update.ErrNetwork.Error()
	}
	if errors.Is(err, update.ErrNoXdgOpen) {
		return err.Error() // keeps the folder to open by hand
	}
	for _, s := range []error{
		update.ErrNoChecksum, update.ErrNoAsset, update.ErrChecksum, update.ErrTooLarge,
		update.ErrNetwork, update.ErrUnsupported, update.ErrDest, update.ErrApplyBlocked,
		errUpToDate, domain.ErrNoReleaseSource,
	} {
		if errors.Is(err, s) {
			return s.Error()
		}
	}
	return err.Error()
}

// progressGate thins out progress events: one per whole percent or per
// 250 ms, plus the final one.
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
