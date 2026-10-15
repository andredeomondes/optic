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
}

func newSession(st domain.State) *fakeSession {
	return &fakeSession{state: st, done: make(chan struct{})}
}

func (f *fakeSession) Info() domain.DeviceInfo { return domain.DeviceInfo{Model: "Fake"} }
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
