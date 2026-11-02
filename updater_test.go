package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"optic/internal/domain"
	"optic/internal/update"
)

// fakeUpdater records events and calls; fetch and apply are set per test.
type fakeUpdater struct {
	*updater
	evMu   sync.Mutex
	events []string // "state:<state>" or "progress"
	states []UpdateState
	done   chan struct{} // closed on the first ready/error state or quit
	once   sync.Once
	quits  int
	calls  []string
}

func newFakeUpdater(t *testing.T, goos string) *fakeUpdater {
	t.Helper()
	f := &fakeUpdater{done: make(chan struct{})}
	rel := domain.Release{Tag: "v1.1.0", Assets: []domain.Asset{
		{Name: update.AssetWindowsPortable}, {Name: update.AssetWindowsSetup},
		{Name: update.AssetMacOS}, {Name: update.AssetLinux}, {Name: update.AssetSums},
	}}
	finish := func() { f.once.Do(func() { close(f.done) }) }
	f.updater = &updater{
		version: "1.0.10",
		dir:     filepath.Join(t.TempDir(), "updates"),
		goos:    goos,
		goarch:  "amd64",
		pid:     42,
		exe:     func() (string, error) { return `C:\Tools\optic.exe`, nil },
		latest:  func(context.Context) (domain.Release, error) { return rel, nil },
		fetch: func(_ context.Context, _ domain.Release, asset, dir string, progress func(int64, int64)) (string, string, error) {
			progress(50, 100)
			progress(100, 100)
			return filepath.Join(dir, asset), strings.Repeat("a", 64), nil
		},
		apply: func(update.Target, string, string, string, int, string, string) (bool, error) { return false, nil },
		emit: func(ev string, data any) {
			f.evMu.Lock()
			defer f.evMu.Unlock()
			if st, ok := data.(UpdateState); ok {
				f.events = append(f.events, "state:"+st.State)
				f.states = append(f.states, st)
				if st.State == updReady || st.State == updError {
					finish()
				}
				return
			}
			f.events = append(f.events, ev)
		},
	}
	f.quit = func() {
		f.evMu.Lock()
		f.quits++
		f.evMu.Unlock()
		finish()
	}
	return f
}

func (f *fakeUpdater) wait(t *testing.T) {
	t.Helper()
	select {
	case <-f.done:
	case <-time.After(5 * time.Second):
		t.Fatal("update did not finish")
	}
	// Let start's goroutine clear running after the last event.
	for i := 0; i < 100; i++ {
		f.mu.Lock()
		r := f.running
		f.mu.Unlock()
		if !r || f.quits > 0 {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func (f *fakeUpdater) stateList() []string {
	f.evMu.Lock()
	defer f.evMu.Unlock()
	var out []string
	for _, e := range f.events {
		if strings.HasPrefix(e, "state:") {
			out = append(out, strings.TrimPrefix(e, "state:"))
		}
	}
	return out
}

func (f *fakeUpdater) lastState() UpdateState {
	f.evMu.Lock()
	defer f.evMu.Unlock()
	return f.states[len(f.states)-1]
}

func TestStartUpdateSingleFlight(t *testing.T) {
	f := newFakeUpdater(t, "darwin")
	release := make(chan struct{})
	var mu sync.Mutex
	fetches := 0
	f.fetch = func(ctx context.Context, rel domain.Release, asset, dir string, progress func(int64, int64)) (string, string, error) {
		mu.Lock()
		fetches++
		mu.Unlock()
		<-release
		return filepath.Join(dir, asset), strings.Repeat("b", 64), nil
	}
	f.start(context.Background())
	f.start(context.Background())
	f.start(context.Background())
	close(release)
	f.wait(t)
	mu.Lock()
	n := fetches
	mu.Unlock()
	if n != 1 {
		t.Fatalf("fetch called %d times, want 1", n)
	}
	// Once finished, a new update may start.
	f.done = make(chan struct{})
	f.once = sync.Once{}
	f.start(context.Background())
	f.wait(t)
	mu.Lock()
	n = fetches
	mu.Unlock()
	if n != 2 {
		t.Fatalf("fetch called %d times after a finished run, want 2", n)
	}
}

func TestStartUpdateReadyOnMac(t *testing.T) {
	f := newFakeUpdater(t, "darwin")
	var got struct {
		path, sum, version, dir string
		kind                    update.Kind
	}
	f.apply = func(tg update.Target, path, sum, exe string, pid int, dir, version string) (bool, error) {
		got.path, got.sum, got.version, got.dir, got.kind = path, sum, version, dir, tg.Kind
		return false, nil
	}
	f.start(context.Background())
	f.wait(t)
	want := []string{updDownloading, updVerifying, updInstalling, updReady}
	if s := f.stateList(); strings.Join(s, ",") != strings.Join(want, ",") {
		t.Fatalf("states = %v, want %v", s, want)
	}
	if st := f.lastState(); st.Kind != "dmg" {
		t.Errorf("kind = %q", st.Kind)
	}
	if got.path != filepath.Join(f.dir, "1.1.0", update.AssetMacOS) || got.sum != strings.Repeat("a", 64) ||
		got.version != "1.1.0" || got.dir != f.dir || got.kind != update.KindDMG {
		t.Errorf("apply got %+v", got)
	}
	if f.quits != 0 {
		t.Error("quit called on macOS")
	}
}

func TestStartUpdateWindowsQuits(t *testing.T) {
	f := newFakeUpdater(t, "windows")
	f.apply = func(update.Target, string, string, string, int, string, string) (bool, error) { return true, nil }
	f.start(context.Background())
	f.wait(t)
	if f.quits != 1 {
		t.Fatalf("quits = %d, want 1", f.quits)
	}
	if s := f.stateList(); s[len(s)-1] != updInstalling {
		t.Errorf("states = %v, want to end at installing", s)
	}
}

func TestStartUpdateApplyBlocked(t *testing.T) {
	f := newFakeUpdater(t, "windows")
	f.apply = func(update.Target, string, string, string, int, string, string) (bool, error) {
		return false, update.ErrApplyBlocked
	}
	f.start(context.Background())
	f.wait(t)
	st := f.lastState()
	if st.State != updError || st.Error != update.ErrApplyBlocked.Error() || st.Kind != "portable" {
		t.Fatalf("state = %+v", st)
	}
	if f.quits != 0 {
		t.Error("quit called after a blocked apply")
	}
}

func TestStartUpdateErrors(t *testing.T) {
	cases := []struct {
		name  string
		setup func(f *fakeUpdater)
		want  string
	}{
		{"up to date", func(f *fakeUpdater) { f.version = "1.1.0" }, errUpToDate.Error()},
		{"checksum with tail", func(f *fakeUpdater) {
			f.fetch = func(context.Context, domain.Release, string, string, func(int64, int64)) (string, string, error) {
				return "", "", errors.Join(update.ErrChecksum, errors.New("detail"))
			}
		}, update.ErrChecksum.Error()},
		{"timeout", func(f *fakeUpdater) {
			f.fetch = func(context.Context, domain.Release, string, string, func(int64, int64)) (string, string, error) {
				return "", "", context.DeadlineExceeded
			}
		}, update.ErrNetwork.Error()},
		{"unsupported os", func(f *fakeUpdater) { f.goos = "plan9" }, update.ErrUnsupported.Error()},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newFakeUpdater(t, "linux")
			c.setup(f)
			f.start(context.Background())
			f.wait(t)
			if st := f.lastState(); st.State != updError || st.Error != c.want {
				t.Fatalf("state = %+v, want error %q", st, c.want)
			}
		})
	}
}

func TestUpdateSupport(t *testing.T) {
	f := newFakeUpdater(t, "windows")
	if s := f.support(context.Background()); !s.Auto || s.Kind != "portable" || s.Reason != "" {
		t.Errorf("windows portable: %+v", s)
	}
	f.programFiles = []string{`C:\Tools`}
	if s := f.support(context.Background()); !s.Auto || s.Kind != "installer" {
		t.Errorf("windows installer: %+v", s)
	}
	f.latest = func(context.Context) (domain.Release, error) {
		return domain.Release{Tag: "v1.0.9", Assets: []domain.Asset{{Name: update.AssetWindowsSetup}}}, nil
	}
	if s := f.support(context.Background()); s.Auto || s.Reason != update.ErrNoChecksum.Error() {
		t.Errorf("no sums: %+v", s)
	}
	f.latest = func(context.Context) (domain.Release, error) {
		return domain.Release{Tag: "v1.1.0", Assets: []domain.Asset{{Name: update.AssetSums}}}, nil
	}
	if s := f.support(context.Background()); s.Auto || s.Reason != update.ErrNoAsset.Error() {
		t.Errorf("no asset: %+v", s)
	}
	f.goos = "freebsd"
	if s := f.support(context.Background()); s.Auto || s.Kind != "" || s.Reason != update.ErrUnsupported.Error() {
		t.Errorf("unsupported: %+v", s)
	}
}

func TestUpdaterLoadResultOnce(t *testing.T) {
	f := newFakeUpdater(t, "windows")
	os.MkdirAll(filepath.Join(f.dir, "1.0.9"), 0o755)
	b, _ := json.Marshal(update.Result{Version: "1.0.10", OK: true})
	os.WriteFile(filepath.Join(f.dir, update.ResultFile), b, 0o644)
	f.load()
	r := f.lastResult()
	if r == nil || !r.OK || r.Version != "1.0.10" {
		t.Fatalf("result = %+v", r)
	}
	if f.lastResult() != nil {
		t.Error("result shown twice")
	}
	if _, err := os.Stat(filepath.Join(f.dir, "1.0.9")); !os.IsNotExist(err) {
		t.Error("stale version folder kept")
	}
}

func TestUpdaterLoadOKButOldVersion(t *testing.T) {
	f := newFakeUpdater(t, "windows")
	os.MkdirAll(f.dir, 0o755)
	b, _ := json.Marshal(update.Result{Version: "1.1.0", OK: true})
	os.WriteFile(filepath.Join(f.dir, update.ResultFile), b, 0o644)
	f.load()
	if r := f.lastResult(); r == nil || r.OK {
		t.Fatalf("result = %+v, want ok=false", r)
	}
}

func TestProgressGate(t *testing.T) {
	var g progressGate
	t0 := time.Unix(1000, 0)
	if !g.allow(0, 1000, t0) {
		t.Error("first event dropped")
	}
	if g.allow(5, 1000, t0.Add(10*time.Millisecond)) {
		t.Error("sub-percent change let through")
	}
	if !g.allow(10, 1000, t0.Add(20*time.Millisecond)) {
		t.Error("1% change dropped")
	}
	if !g.allow(11, 1000, t0.Add(300*time.Millisecond)) {
		t.Error("250 ms tick dropped")
	}
	if !g.allow(1000, 1000, t0.Add(310*time.Millisecond)) {
		t.Error("final event dropped")
	}
	var u progressGate
	u.allow(1, 0, t0)
	if u.allow(2, 0, t0.Add(100*time.Millisecond)) || !u.allow(3, 0, t0.Add(400*time.Millisecond)) {
		t.Error("unknown total should tick by time")
	}
}

func TestProductVersion(t *testing.T) {
	if v := productVersion(wailsJSON); !update.IsNewer("0.0.1", v) {
		t.Fatalf("productVersion(wails.json) = %q", v)
	}
	if productVersion([]byte("{")) != "" {
		t.Error("bad json gave a version")
	}
}
