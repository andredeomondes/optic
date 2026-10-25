package control

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"optic/internal/domain"
)

// ── fakes ───────────────────────────────────────────────────────────────

type fakeSession struct {
	mu     sync.Mutex
	state  domain.State
	keys   []string
	done   chan struct{}
	closed bool
	plat   domain.Platform
}

func newSession(st domain.State) *fakeSession {
	return &fakeSession{state: st, done: make(chan struct{})}
}

func (f *fakeSession) Info() domain.DeviceInfo { return domain.DeviceInfo{Model: "Fake", Platform: f.plat} }
func (f *fakeSession) State() domain.State     { return f.state }
func (f *fakeSession) Done() <-chan struct{}   { return f.done }
func (f *fakeSession) SendText(string) error   { return nil }
func (f *fakeSession) LaunchApp(string) error  { return nil }
func (f *fakeSession) SendKey(name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.keys = append(f.keys, name)
	return nil
}
func (f *fakeSession) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed = true
	return nil
}

type fakeConnector struct {
	sess *fakeSession
	err  error
}

func (c fakeConnector) Connect(context.Context, string, domain.SessionEvents) (domain.Session, error) {
	if c.err != nil {
		return nil, c.err
	}
	return c.sess, nil
}

type fakeHosts struct{ saved string }

func (h *fakeHosts) LastHost() string             { return h.saved }
func (h *fakeHosts) SaveLastHost(ip string) error { h.saved = ip; return nil }

type fakeEvents struct{ disconnected chan struct{} }

func (e fakeEvents) State(domain.State)   {}
func (e fakeEvents) Trace(string, string) {}
func (e fakeEvents) Disconnected()        { close(e.disconnected) }

type fakePairing struct {
	results []error
	closed  bool
}

func (p *fakePairing) ServerName() string { return "TV" }
func (p *fakePairing) Close() error       { p.closed = true; return nil }
func (p *fakePairing) Finish(string) error {
	err := p.results[0]
	p.results = p.results[1:]
	return err
}

type fakePairer struct{ p *fakePairing }

func (f fakePairer) StartPairing(context.Context, string) (domain.PairingSession, error) {
	return f.p, nil
}

func newService(conn domain.Connector, pairer domain.Pairer) (*Service, *fakeHosts, fakeEvents) {
	hosts := &fakeHosts{}
	ev := fakeEvents{disconnected: make(chan struct{})}
	return New(Deps{
		Connector: conn,
		Pairer:    pairer,
		Hosts:     hosts,
		Events:    ev,
		Sleep:     func(time.Duration) {},
	}), hosts, ev
}

// ── tests ───────────────────────────────────────────────────────────────

func TestCommandsRequireConnection(t *testing.T) {
	s, _, _ := newService(fakeConnector{}, nil)
	for name, call := range map[string]func() error{
		"SendKey":   func() error { return s.SendKey("HOME") },
		"SendText":  func() error { return s.SendText("oi") },
		"LaunchApp": func() error { return s.LaunchApp("youtube") },
		"SetVolume": func() error { return s.SetVolume(10) },
	} {
		if err := call(); !errors.Is(err, domain.ErrNotConnected) {
			t.Errorf("%s: got %v, want ErrNotConnected", name, err)
		}
	}
}

func TestConnectRemembersHost(t *testing.T) {
	s, hosts, _ := newService(fakeConnector{sess: newSession(domain.State{})}, nil)
	info, err := s.Connect(context.Background(), "192.168.0.18")
	if err != nil {
		t.Fatal(err)
	}
	if info.Model != "Fake" || hosts.saved != "192.168.0.18" {
		t.Fatalf("info=%+v saved=%q", info, hosts.saved)
	}
}

func TestConnectPropagatesNotPaired(t *testing.T) {
	s, hosts, _ := newService(fakeConnector{err: domain.ErrNotPaired}, nil)
	if _, err := s.Connect(context.Background(), "x"); !errors.Is(err, domain.ErrNotPaired) {
		t.Fatalf("got %v", err)
	}
	if hosts.saved != "" {
		t.Fatal("host saved after a failed connect")
	}
}

func TestReconnectClosesPreviousSession(t *testing.T) {
	first := newSession(domain.State{})
	conn := &switchConnector{next: first}
	s, _, _ := newService(conn, nil)
	s.Connect(context.Background(), "a")
	conn.next = newSession(domain.State{})
	s.Connect(context.Background(), "b")
	if !first.closed {
		t.Fatal("previous session left open")
	}
}

type switchConnector struct{ next *fakeSession }

func (c *switchConnector) Connect(context.Context, string, domain.SessionEvents) (domain.Session, error) {
	return c.next, nil
}

func TestSessionEndEmitsDisconnected(t *testing.T) {
	sess := newSession(domain.State{})
	s, _, ev := newService(fakeConnector{sess: sess}, nil)
	s.Connect(context.Background(), "x")
	close(sess.done)
	select {
	case <-ev.disconnected:
	case <-time.After(time.Second):
		t.Fatal("no Disconnected event")
	}
	if err := s.SendKey("HOME"); !errors.Is(err, domain.ErrNotConnected) {
		t.Fatalf("session not cleared: %v", err)
	}
}

func TestSetVolumeRokuNotReported(t *testing.T) {
	sess := newSession(domain.State{Volume: 10, VolumeMax: 100})
	sess.plat = domain.PlatformRoku
	s, _, _ := newService(fakeConnector{sess: sess}, nil)
	s.Connect(context.Background(), "x")
	if err := s.SetVolume(20); !errors.Is(err, domain.ErrVolumeNotReported) {
		t.Fatalf("err = %v", err)
	}
	if len(sess.keys) != 0 {
		t.Fatalf("sent %v", sess.keys)
	}
}

func TestSetVolume(t *testing.T) {
	cases := []struct {
		name      string
		state     domain.State
		level     int
		wantKey   string
		wantCount int
		wantErr   error
	}{
		{"up", domain.State{Volume: 10, VolumeMax: 100}, 13, "VOLUME_UP", 3, nil},
		{"down", domain.State{Volume: 10, VolumeMax: 100}, 8, "VOLUME_DOWN", 2, nil},
		{"same", domain.State{Volume: 10, VolumeMax: 100}, 10, "", 0, nil},
		{"clamped high", domain.State{Volume: 98, VolumeMax: 100}, 150, "VOLUME_UP", 2, nil},
		{"clamped low", domain.State{Volume: 1, VolumeMax: 100}, -5, "VOLUME_DOWN", 1, nil},
		{"unknown", domain.State{}, 10, "", 0, domain.ErrVolumeUnknown},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sess := newSession(tc.state)
			s, _, _ := newService(fakeConnector{sess: sess}, nil)
			s.Connect(context.Background(), "x")
			err := s.SetVolume(tc.level)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if len(sess.keys) != tc.wantCount {
				t.Fatalf("sent %v, want %d × %s", sess.keys, tc.wantCount, tc.wantKey)
			}
			if tc.wantCount > 0 && slices.ContainsFunc(sess.keys, func(k string) bool { return k != tc.wantKey }) {
				t.Fatalf("sent %v, want only %s", sess.keys, tc.wantKey)
			}
		})
	}
}

func TestPairingTypoKeepsSessionOpen(t *testing.T) {
	p := &fakePairing{results: []error{domain.ErrCodeTypo, nil}}
	s, _, _ := newService(nil, fakePairer{p})
	if _, err := s.StartPairing(context.Background(), "x"); err != nil {
		t.Fatal(err)
	}
	if err := s.FinishPairing("AAAAAA"); !errors.Is(err, domain.ErrCodeTypo) || p.closed {
		t.Fatalf("typo: err=%v closed=%v", err, p.closed)
	}
	if err := s.FinishPairing("DF5FB1"); err != nil || !p.closed {
		t.Fatalf("success: err=%v closed=%v", err, p.closed)
	}
	if err := s.FinishPairing("DF5FB1"); !errors.Is(err, domain.ErrNoPairing) {
		t.Fatalf("after finish: %v", err)
	}
}

type fakeDiscoverer struct{ tvs []domain.TV }

func (f fakeDiscoverer) Discover(context.Context, time.Duration) ([]domain.TV, error) {
	return f.tvs, nil
}

type memKnown struct {
	m      map[string]domain.KnownTV
	putErr error
}

func (k *memKnown) Get(ip string) (domain.KnownTV, bool) { tv, ok := k.m[ip]; return tv, ok }
func (k *memKnown) List() []domain.KnownTV {
	out := make([]domain.KnownTV, 0, len(k.m))
	for _, tv := range k.m {
		out = append(out, tv)
	}
	return out
}
func (k *memKnown) Put(tv domain.KnownTV) error {
	if k.putErr != nil {
		return k.putErr
	}
	// Merge like the real store: empty fields keep what is stored.
	old := k.m[tv.IP]
	if tv.Name == "" {
		tv.Name = old.Name
	}
	if tv.Platform == "" {
		tv.Platform = old.Platform
	}
	if tv.MAC == "" {
		tv.MAC = old.MAC
	}
	k.m[tv.IP] = tv
	return nil
}

func TestDiscoverRemembersPlatform(t *testing.T) {
	known := &memKnown{m: map[string]domain.KnownTV{"192.168.0.40": {IP: "192.168.0.40", MAC: "aa:bb:cc:dd:ee:ff"}}}
	s := New(Deps{
		Discoverer: fakeDiscoverer{tvs: []domain.TV{{Name: "Sala", IP: "192.168.0.40", Platform: domain.PlatformWebOS}}},
		Known:      known,
	})
	tvs, err := s.Discover(context.Background())
	if err != nil || len(tvs) != 1 {
		t.Fatalf("tvs=%v err=%v", tvs, err)
	}
	got, ok := known.Get("192.168.0.40")
	if !ok || got.Platform != domain.PlatformWebOS || got.Name != "Sala" {
		t.Fatalf("got %+v ok=%v", got, ok)
	}
}

func TestDiscoverSurvivesKnownPutError(t *testing.T) {
	known := &memKnown{m: map[string]domain.KnownTV{}, putErr: errors.New("disk full")}
	s := New(Deps{
		Discoverer: fakeDiscoverer{tvs: []domain.TV{{IP: "192.168.0.40", Platform: domain.PlatformWebOS}}},
		Known:      known,
	})
	if tvs, err := s.Discover(context.Background()); err != nil || len(tvs) != 1 {
		t.Fatalf("tvs=%v err=%v", tvs, err)
	}
}

type fakeMACs struct {
	mac string
	ok  bool
}

func (f fakeMACs) LookupMAC(string) (string, bool) { return f.mac, f.ok }

type fakeWaker struct{ mac, ip string }

func (w *fakeWaker) Send(mac, ip string) error { w.mac, w.ip = mac, ip; return nil }

func TestConnectLearnsMACAndPowerOn(t *testing.T) {
	known := &memKnown{m: map[string]domain.KnownTV{"10.0.0.2": {IP: "10.0.0.2", Name: "Sala"}}}
	waker := &fakeWaker{}
	s := New(Deps{
		Connector: fakeConnector{sess: newSession(domain.State{})},
		Hosts:     &fakeHosts{},
		Known:     known,
		MACs:      fakeMACs{mac: "aa:bb:cc:dd:ee:ff", ok: true},
		Waker:     waker,
		Events:    fakeEvents{disconnected: make(chan struct{})},
	})
	info, err := s.Connect(context.Background(), "10.0.0.2")
	if err != nil || !info.Capabilities.PowerOn {
		t.Fatalf("info=%+v err=%v", info, err)
	}
	if got := known.m["10.0.0.2"]; got.MAC != "aa:bb:cc:dd:ee:ff" || got.Name != "Sala" {
		t.Fatalf("known=%+v", got)
	}
	if err := s.PowerOn("10.0.0.2"); err != nil {
		t.Fatal(err)
	}
	if waker.mac != "aa:bb:cc:dd:ee:ff" || waker.ip != "10.0.0.2" {
		t.Fatalf("waker=%+v", waker)
	}
}

func TestPowerOnWithoutMAC(t *testing.T) {
	s := New(Deps{Known: &memKnown{m: map[string]domain.KnownTV{}}, Waker: &fakeWaker{}})
	if err := s.PowerOn("10.0.0.2"); !errors.Is(err, domain.ErrNoMAC) {
		t.Fatalf("got %v", err)
	}
	// Unwired adapters mean the feature is unavailable, not a panic.
	if err := New(Deps{}).PowerOn("10.0.0.2"); !errors.Is(err, domain.ErrNoMAC) {
		t.Fatalf("nil deps: got %v", err)
	}
}

func TestConnectWithoutMACStillConnects(t *testing.T) {
	s := New(Deps{
		Connector: fakeConnector{sess: newSession(domain.State{})},
		Hosts:     &fakeHosts{},
		Known:     &memKnown{m: map[string]domain.KnownTV{}},
		MACs:      fakeMACs{},
		Events:    fakeEvents{disconnected: make(chan struct{})},
	})
	info, err := s.Connect(context.Background(), "10.0.0.2")
	if err != nil || info.Capabilities.PowerOn {
		t.Fatalf("info=%+v err=%v", info, err)
	}
}

func TestConnectUsesStoredMACWhenLookupFails(t *testing.T) {
	s := New(Deps{
		Connector: fakeConnector{sess: newSession(domain.State{})},
		Hosts:     &fakeHosts{},
		Known:     &memKnown{m: map[string]domain.KnownTV{"10.0.0.2": {IP: "10.0.0.2", MAC: "aa:bb:cc:dd:ee:ff"}}},
		MACs:      fakeMACs{},
		Events:    fakeEvents{disconnected: make(chan struct{})},
	})
	info, err := s.Connect(context.Background(), "10.0.0.2")
	if err != nil || !info.Capabilities.PowerOn {
		t.Fatalf("info=%+v err=%v", info, err)
	}
}

func TestDiscoverFillsStoredMAC(t *testing.T) {
	known := &memKnown{m: map[string]domain.KnownTV{"192.168.0.40": {IP: "192.168.0.40", MAC: "aa:bb:cc:dd:ee:ff"}}}
	s := New(Deps{
		Discoverer: fakeDiscoverer{tvs: []domain.TV{{IP: "192.168.0.40"}, {IP: "192.168.0.41"}}},
		Known:      known,
	})
	tvs, err := s.Discover(context.Background())
	if err != nil || len(tvs) != 2 {
		t.Fatalf("tvs=%v err=%v", tvs, err)
	}
	if tvs[0].MAC != "aa:bb:cc:dd:ee:ff" || tvs[1].MAC != "" {
		t.Fatalf("macs: %q %q", tvs[0].MAC, tvs[1].MAC)
	}
}

func TestDiscoverAppendsKnownTVsWithMACNotFound(t *testing.T) {
	known := &memKnown{m: map[string]domain.KnownTV{
		"192.168.0.40": {IP: "192.168.0.40", Name: "Sala", MAC: "aa:bb:cc:dd:ee:ff", Platform: domain.PlatformWebOS},
		"192.168.0.41": {IP: "192.168.0.41", Name: "Quarto", MAC: "11:22:33:44:55:66"},
		"192.168.0.42": {IP: "192.168.0.42", Name: "SemMac"},
		"192.168.0.43": {IP: "192.168.0.43", Name: "Achada", MAC: "99:99:99:99:99:99"},
	}}
	s := New(Deps{
		Discoverer: fakeDiscoverer{tvs: []domain.TV{{Name: "Achada", IP: "192.168.0.43", Platform: domain.PlatformAndroidTV}}},
		Known:      known,
	})
	tvs, err := s.Discover(context.Background())
	if err != nil || len(tvs) != 3 {
		t.Fatalf("tvs=%v err=%v", tvs, err)
	}
	byIP := map[string]domain.TV{}
	for _, tv := range tvs {
		byIP[tv.IP] = tv
	}
	sala := byIP["192.168.0.40"]
	if sala.MAC != "aa:bb:cc:dd:ee:ff" || sala.Name != "Sala" || sala.Platform != domain.PlatformWebOS || sala.Remotev2 {
		t.Errorf("sala = %+v", sala)
	}
	quarto := byIP["192.168.0.41"]
	if quarto.MAC != "11:22:33:44:55:66" || quarto.Platform != domain.PlatformAndroidTV || !quarto.Remotev2 {
		t.Errorf("quarto = %+v", quarto)
	}
	if _, ok := byIP["192.168.0.42"]; ok {
		t.Error("known TV without MAC must not be listed")
	}
	if byIP["192.168.0.43"].MAC != "99:99:99:99:99:99" {
		t.Errorf("found TV lost its MAC: %+v", byIP["192.168.0.43"])
	}
}

func TestDiscoverSkipsKnownTVWithSameMACAtNewIP(t *testing.T) {
	known := &memKnown{m: map[string]domain.KnownTV{
		"192.168.0.18": {IP: "192.168.0.18", Name: "Sala", MAC: "aa:bb:cc:dd:ee:ff", Platform: domain.PlatformAndroidTV},
		"192.168.0.25": {IP: "192.168.0.25", Name: "Sala", MAC: "AA:BB:CC:DD:EE:FF", Platform: domain.PlatformAndroidTV},
	}}
	s := New(Deps{
		Discoverer: fakeDiscoverer{tvs: []domain.TV{{IP: "192.168.0.25", Platform: domain.PlatformAndroidTV}}},
		Known:      known,
	})
	tvs, err := s.Discover(context.Background())
	if err != nil || len(tvs) != 1 {
		t.Fatalf("tvs=%+v err=%v", tvs, err)
	}
	if tvs[0].IP != "192.168.0.25" || tvs[0].MAC != "AA:BB:CC:DD:EE:FF" {
		t.Errorf("tv = %+v", tvs[0])
	}
}

type fakeReleases struct {
	rel      domain.Release
	err      error
	deadline bool
}

func (f *fakeReleases) Latest(ctx context.Context) (domain.Release, error) {
	_, f.deadline = ctx.Deadline()
	return f.rel, f.err
}

func TestLatestReleaseWithoutSource(t *testing.T) {
	_, err := New(Deps{}).LatestRelease(context.Background())
	if !errors.Is(err, domain.ErrNoReleaseSource) {
		t.Fatalf("err = %v, want ErrNoReleaseSource", err)
	}
}

func TestLatestReleaseDelegatesWithTimeout(t *testing.T) {
	src := &fakeReleases{rel: domain.Release{Tag: "v1.0.3"}}
	rel, err := New(Deps{Releases: src}).LatestRelease(context.Background())
	if err != nil || rel.Tag != "v1.0.3" {
		t.Fatalf("rel=%+v err=%v", rel, err)
	}
	if !src.deadline {
		t.Error("lookup context has no deadline")
	}
}

func TestLatestReleasePropagatesError(t *testing.T) {
	boom := errors.New("boom")
	_, err := New(Deps{Releases: &fakeReleases{err: boom}}).LatestRelease(context.Background())
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v", err)
	}
}

// ── network power and installed apps ────────────────────────────────────

type fakePower struct {
	err   error
	calls int
}

func (p *fakePower) PowerOn(context.Context, string) error { p.calls++; return p.err }

func TestPowerOnPrefersNetworkPower(t *testing.T) {
	power, waker := &fakePower{}, &fakeWaker{}
	known := &memKnown{m: map[string]domain.KnownTV{"10.0.0.2": {IP: "10.0.0.2", MAC: "aa:bb:cc:dd:ee:ff"}}}
	s := New(Deps{Known: known, Waker: waker, Power: power})
	if err := s.PowerOn("10.0.0.2"); err != nil {
		t.Fatal(err)
	}
	if power.calls != 1 || waker.mac != "" {
		t.Fatalf("power calls=%d waker mac=%q", power.calls, waker.mac)
	}
}

func TestPowerOnFallsBackToWoL(t *testing.T) {
	for name, err := range map[string]error{
		"unsupported": domain.ErrUnsupported{Platform: domain.PlatformAndroidTV},
		"failure":     errors.New("timeout"),
	} {
		power, waker := &fakePower{err: err}, &fakeWaker{}
		known := &memKnown{m: map[string]domain.KnownTV{"10.0.0.2": {IP: "10.0.0.2", MAC: "aa:bb:cc:dd:ee:ff"}}}
		s := New(Deps{Known: known, Waker: waker, Power: power})
		if err := s.PowerOn("10.0.0.2"); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if waker.mac != "aa:bb:cc:dd:ee:ff" {
			t.Fatalf("%s: waker mac=%q", name, waker.mac)
		}
	}
}

func TestPowerOnNoMACAfterNetworkFail(t *testing.T) {
	power := &fakePower{err: errors.New("timeout")}
	s := New(Deps{Known: &memKnown{m: map[string]domain.KnownTV{}}, Waker: &fakeWaker{}, Power: power})
	if err := s.PowerOn("10.0.0.2"); !errors.Is(err, domain.ErrNoMAC) {
		t.Fatalf("got %v, want ErrNoMAC", err)
	}
}

type appSession struct {
	*fakeSession
	apps []domain.App
}

func (a appSession) Apps(context.Context) ([]domain.App, error) { return a.apps, nil }

type connectorFor struct{ sess domain.Session }

func (c connectorFor) Connect(context.Context, string, domain.SessionEvents) (domain.Session, error) {
	return c.sess, nil
}

func TestInstalledApps(t *testing.T) {
	want := []domain.App{{ID: "12", Name: "Netflix"}}
	s, _, _ := newService(connectorFor{appSession{newSession(domain.State{}), want}}, nil)
	ctx := context.Background()
	if _, err := s.InstalledApps(ctx); !errors.Is(err, domain.ErrNotConnected) {
		t.Fatalf("not connected: got %v", err)
	}
	if _, err := s.Connect(ctx, "10.0.0.2"); err != nil {
		t.Fatal(err)
	}
	got, err := s.InstalledApps(ctx)
	if err != nil || !slices.Equal(got, want) {
		t.Fatalf("got %v, %v", got, err)
	}

	s, _, _ = newService(fakeConnector{sess: newSession(domain.State{})}, nil)
	if _, err := s.Connect(ctx, "10.0.0.2"); err != nil {
		t.Fatal(err)
	}
	var u domain.ErrUnsupported
	if _, err := s.InstalledApps(ctx); !errors.As(err, &u) {
		t.Fatalf("got %v, want ErrUnsupported", err)
	}
}

type powerSession struct{ *fakeSession }

func (powerSession) Info() domain.DeviceInfo {
	return domain.DeviceInfo{Capabilities: domain.Capabilities{PowerOn: true}}
}

func TestConnectKeepsAdapterPowerOn(t *testing.T) {
	s, _, _ := newService(connectorFor{powerSession{newSession(domain.State{})}}, nil)
	info, err := s.Connect(context.Background(), "10.0.0.2")
	if err != nil || !info.Capabilities.PowerOn {
		t.Fatalf("info=%+v err=%v", info, err)
	}
}
