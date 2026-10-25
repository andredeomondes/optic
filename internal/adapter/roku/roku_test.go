package roku

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"optic/internal/domain"
	"optic/pkg/roku"
)

type fakeRoku struct {
	mu        sync.Mutex
	power     string
	active    string
	forbidden bool
	reqs      []string
	srv       *httptest.Server
}

func newFake(t *testing.T) *fakeRoku {
	t.Helper()
	f := &fakeRoku{power: "PowerOn", active: "Roku"}
	f.srv = httptest.NewServer(http.HandlerFunc(f.handle))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeRoku) handle(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reqs = append(f.reqs, r.URL.Path)
	if f.forbidden {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	switch {
	case r.URL.Path == "/query/device-info":
		fmt.Fprintf(w, `<device-info><vendor-name>TCL</vendor-name><model-name>55R635</model-name>`+
			`<software-version>12.5.0</software-version><power-mode>%s</power-mode><is-tv>true</is-tv></device-info>`, f.power)
	case r.URL.Path == "/query/active-app":
		fmt.Fprintf(w, `<active-app><app id="12">%s</app></active-app>`, f.active)
	case r.URL.Path == "/query/apps":
		fmt.Fprint(w, `<apps><app id="12" type="appl">Netflix</app><app id="837" type="appl">YouTube</app></apps>`)
	}
}

func (f *fakeRoku) set(power, active string) {
	f.mu.Lock()
	f.power, f.active = power, active
	f.mu.Unlock()
}

func (f *fakeRoku) count(prefix string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, p := range f.reqs {
		if strings.HasPrefix(p, prefix) {
			n++
		}
	}
	return n
}

func (f *fakeRoku) total() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.reqs)
}

type events struct {
	mu     sync.Mutex
	states []domain.State
	traces []string
}

func (e *events) OnState(s domain.State) {
	e.mu.Lock()
	e.states = append(e.states, s)
	e.mu.Unlock()
}

func (e *events) OnTrace(dir, line string) {
	e.mu.Lock()
	e.traces = append(e.traces, dir+" "+line)
	e.mu.Unlock()
}

func (e *events) sawState(pred func(domain.State) bool) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, s := range e.states {
		if pred(s) {
			return true
		}
	}
	return false
}

func adapter(f *fakeRoku) Adapter {
	return Adapter{
		BaseURL:  func(string) string { return f.srv.URL },
		Poll:     20 * time.Millisecond,
		MaxFails: 3,
	}
}

func connect(t *testing.T, f *fakeRoku) (domain.Session, *events) {
	t.Helper()
	ev := &events{}
	s, err := adapter(f).Connect(context.Background(), "tv", ev)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s, ev
}

func eventually(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for %s", what)
}

func TestConnectReadsInfo(t *testing.T) {
	f := newFake(t)
	s, ev := connect(t, f)
	i := s.Info()
	if i.Model != "55R635" || i.Vendor != "TCL" || i.Version != "12.5.0" || i.Platform != domain.PlatformRoku {
		t.Fatalf("info = %+v", i)
	}
	if !i.Capabilities.Text || !i.Capabilities.AppList || !i.Capabilities.PowerOn {
		t.Fatalf("capabilities = %+v", i.Capabilities)
	}
	if !ev.sawState(func(s domain.State) bool { return s.On }) {
		t.Fatal("no initial OnState")
	}
}

func TestConnectForbidden(t *testing.T) {
	f := newFake(t)
	f.forbidden = true
	_, err := adapter(f).Connect(context.Background(), "tv", &events{})
	if !errors.Is(err, roku.ErrForbidden) {
		t.Fatalf("err = %v", err)
	}
}

func TestSendKeyMaps(t *testing.T) {
	f := newFake(t)
	s, _ := connect(t, f)
	for name, path := range map[string]string{
		"DPAD_CENTER":  "/keypress/Select",
		"keycode_home": "/keypress/Home",
	} {
		if err := s.SendKey(name); err != nil {
			t.Fatal(err)
		}
		if f.count(path) != 1 {
			t.Fatalf("%s: no request to %s", name, path)
		}
	}
}

func TestSendKeyUnsupported(t *testing.T) {
	f := newFake(t)
	s, _ := connect(t, f)
	before := f.count("/keypress")
	err := s.SendKey("MEDIA_NEXT")
	if err == nil || !strings.Contains(err.Error(), "MEDIA_NEXT") {
		t.Fatalf("err = %v", err)
	}
	if f.count("/keypress") != before {
		t.Fatal("request sent for unsupported key")
	}
}

func TestLaunchResolves(t *testing.T) {
	f := newFake(t)
	s, _ := connect(t, f)
	for target, path := range map[string]string{
		"market://launch?id=com.netflix.ninja": "/launch/12",
		"youtube":                              "/launch/837",
		"837":                                  "/launch/837",
	} {
		before := f.count(path)
		if err := s.LaunchApp(target); err != nil {
			t.Fatalf("%s: %v", target, err)
		}
		if f.count(path) != before+1 {
			t.Fatalf("%s: no request to %s", target, path)
		}
	}
}

func TestLaunchNotInstalled(t *testing.T) {
	f := newFake(t)
	s, _ := connect(t, f)
	err := s.LaunchApp("market://launch?id=com.disney.disneyplus")
	if !errors.Is(err, ErrAppNotInstalled) {
		t.Fatalf("err = %v", err)
	}
	if f.count("/launch") != 0 {
		t.Fatal("launch sent for missing app")
	}
}

func TestStatePolling(t *testing.T) {
	f := newFake(t)
	s, ev := connect(t, f)
	f.set("DisplayOff", "Netflix")
	eventually(t, "off state", func() bool {
		return ev.sawState(func(s domain.State) bool { return !s.On })
	})
	eventually(t, "app", func() bool { return s.State().App == "Netflix" })
}

func TestDoneAfterFailures(t *testing.T) {
	f := newFake(t)
	s, _ := connect(t, f)
	f.srv.Close()
	select {
	case <-s.Done():
	case <-time.After(20 * time.Millisecond * (3 + 2) * 10):
		t.Fatal("Done not closed")
	}
}

func TestTextNotTraced(t *testing.T) {
	f := newFake(t)
	s, ev := connect(t, f)
	if err := s.SendText("segredo"); err != nil {
		t.Fatal(err)
	}
	ev.mu.Lock()
	defer ev.mu.Unlock()
	if len(ev.traces) == 0 {
		t.Fatal("no trace")
	}
	for _, l := range ev.traces {
		if strings.Contains(l, "segredo") {
			t.Fatalf("text leaked: %q", l)
		}
	}
}

func TestCloseStopsPolling(t *testing.T) {
	f := newFake(t)
	s, _ := connect(t, f)
	s.Close()
	s.Close()
	time.Sleep(40 * time.Millisecond)
	n := f.total()
	time.Sleep(100 * time.Millisecond)
	if f.total() != n {
		t.Fatal("requests after Close")
	}
	select {
	case <-s.Done():
	default:
		t.Fatal("Done not closed")
	}
}

func TestStartPairing(t *testing.T) {
	_, err := Adapter{}.StartPairing(context.Background(), "tv")
	if !errors.Is(err, ErrNoPairingNeeded) {
		t.Fatalf("err = %v", err)
	}
}

func TestPowerOn(t *testing.T) {
	f := newFake(t)
	if err := adapter(f).PowerOn(context.Background(), "tv"); err != nil {
		t.Fatal(err)
	}
	if f.count("/keypress/PowerOn") != 1 {
		t.Fatal("no PowerOn keypress")
	}
}

func TestAppsListed(t *testing.T) {
	f := newFake(t)
	s, _ := connect(t, f)
	l, ok := s.(interface {
		Apps(context.Context) ([]domain.App, error)
	})
	if !ok {
		t.Fatal("session has no Apps")
	}
	apps, err := l.Apps(context.Background())
	if err != nil || len(apps) != 2 || apps[0] != (domain.App{ID: "12", Name: "Netflix"}) {
		t.Fatalf("apps = %v, err = %v", apps, err)
	}
}
