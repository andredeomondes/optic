// Package control holds OPTIC's use cases: discover, pair, connect and send
// commands to a TV. It depends only on domain ports; adapters are injected.
package control

import (
	"context"
	"errors"
	"sync"
	"time"

	"optic/internal/domain"
)

// discoverTimeout is how long a network scan listens for answers.
const discoverTimeout = 3 * time.Second

// volumeStepDelay spaces volume key presses so the TV registers each one.
const volumeStepDelay = 40 * time.Millisecond

// Deps are the ports the service needs.
type Deps struct {
	Discoverer domain.Discoverer
	Connector  domain.Connector
	Pairer     domain.Pairer
	Hosts      domain.HostStore
	Known      domain.KnownTVs
	MACs       domain.MACResolver // optional: without it PowerOn is unavailable
	Waker      domain.WakeSender  // optional: without it PowerOn is unavailable
	Events     domain.EventSink
	// Sleep is time.Sleep by default; tests replace it.
	Sleep func(time.Duration)
}

// Service is the single entry point for TV control. Safe for concurrent use.
type Service struct {
	d Deps

	mu      sync.Mutex
	session domain.Session
	pairing domain.PairingSession
}

func New(d Deps) *Service {
	if d.Sleep == nil {
		d.Sleep = time.Sleep
	}
	return &Service{d: d}
}

// Discover scans the network for TVs and remembers each one found.
func (s *Service) Discover(ctx context.Context) ([]domain.TV, error) {
	tvs, err := s.d.Discoverer.Discover(ctx, discoverTimeout)
	if s.d.Known != nil {
		for _, tv := range tvs {
			// Never touches MAC; a failed write must not fail the scan.
			_ = s.d.Known.Put(domain.KnownTV{IP: tv.IP, Name: tv.Name, Platform: tv.Platform})
		}
		// Hand the stored MAC to the UI so it can offer "turn on" for a TV
		// that is off and so answers no discovery.
		for i := range tvs {
			if k, ok := s.d.Known.Get(tvs[i].IP); ok {
				tvs[i].MAC = k.MAC
			}
		}
	}
	return tvs, err
}

// LastHost is the TV used last time, for reconnecting on launch.
func (s *Service) LastHost() string { return s.d.Hosts.LastHost() }

// Connect replaces any current session with a new one to host.
// It returns domain.ErrNotPaired when the TV needs pairing first.
func (s *Service) Connect(ctx context.Context, host string) (domain.DeviceInfo, error) {
	s.Disconnect()

	sess, err := s.d.Connector.Connect(ctx, host, sessionEvents{s.d.Events})
	if err != nil {
		return domain.DeviceInfo{}, err
	}

	s.mu.Lock()
	s.session = sess
	s.mu.Unlock()
	_ = s.d.Hosts.SaveLastHost(host) // a convenience; never fail the connection over it

	go s.watch(sess)

	info := sess.Info()
	info.Capabilities.PowerOn = s.learnMAC(host) != ""
	return info, nil
}

// learnMAC reads host's MAC from the ARP table and stores it, falling back to
// the one stored earlier. Empty when unknown; never fails a connection.
func (s *Service) learnMAC(host string) string {
	if s.d.MACs != nil {
		if mac, ok := s.d.MACs.LookupMAC(host); ok {
			if s.d.Known != nil {
				_ = s.d.Known.Put(domain.KnownTV{IP: host, MAC: mac})
			}
			return mac
		}
	}
	if s.d.Known != nil {
		if k, ok := s.d.Known.Get(host); ok {
			return k.MAC
		}
	}
	return ""
}

// PowerOn wakes the TV at host with Wake-on-LAN, using the MAC learned the
// last time it was connected while on. It returns domain.ErrNoMAC if unknown.
func (s *Service) PowerOn(host string) error {
	if s.d.Known == nil || s.d.Waker == nil {
		return domain.ErrNoMAC
	}
	k, ok := s.d.Known.Get(host)
	if !ok || k.MAC == "" {
		return domain.ErrNoMAC
	}
	return s.d.Waker.Send(k.MAC, host)
}

// watch clears the session when it ends on its own and tells the UI.
func (s *Service) watch(sess domain.Session) {
	<-sess.Done()
	s.mu.Lock()
	current := s.session == sess
	if current {
		s.session = nil
	}
	s.mu.Unlock()
	if current {
		s.d.Events.Disconnected()
	}
}

// Disconnect closes the current session, if any. Not reported as an event.
func (s *Service) Disconnect() {
	s.mu.Lock()
	sess := s.session
	s.session = nil
	s.mu.Unlock()
	if sess != nil {
		sess.Close()
	}
}

// StartPairing makes the TV show a code and returns the TV's name.
func (s *Service) StartPairing(ctx context.Context, host string) (string, error) {
	s.CancelPairing()
	p, err := s.d.Pairer.StartPairing(ctx, host)
	if err != nil {
		return "", err
	}
	s.mu.Lock()
	s.pairing = p
	s.mu.Unlock()
	return p.ServerName(), nil
}

// FinishPairing submits the code shown on the TV. On domain.ErrCodeTypo the
// pairing stays open so the user can retry; otherwise it ends.
func (s *Service) FinishPairing(code string) error {
	s.mu.Lock()
	p := s.pairing
	s.mu.Unlock()
	if p == nil {
		return domain.ErrNoPairing
	}
	err := p.Finish(code)
	if errors.Is(err, domain.ErrCodeTypo) {
		return err
	}
	s.CancelPairing()
	return err
}

// CancelPairing abandons an in-progress pairing.
func (s *Service) CancelPairing() {
	s.mu.Lock()
	p := s.pairing
	s.pairing = nil
	s.mu.Unlock()
	if p != nil {
		p.Close()
	}
}

func (s *Service) SendKey(name string) error {
	sess, err := s.current()
	if err != nil {
		return err
	}
	return sess.SendKey(name)
}

// SendText types into the focused field on the TV.
func (s *Service) SendText(text string) error {
	sess, err := s.current()
	if err != nil {
		return err
	}
	return sess.SendText(text)
}

// LaunchApp opens an app by shortcut name (youtube, netflix…) or URI.
func (s *Service) LaunchApp(nameOrURI string) error {
	sess, err := s.current()
	if err != nil {
		return err
	}
	return sess.LaunchApp(nameOrURI)
}

// SetVolume steps the volume to level with VOLUME_UP/DOWN presses: the
// protocol reports the level but cannot set it directly.
func (s *Service) SetVolume(level int) error {
	sess, err := s.current()
	if err != nil {
		return err
	}
	st := sess.State()
	if st.VolumeMax == 0 {
		return domain.ErrVolumeUnknown
	}
	level = max(0, min(level, st.VolumeMax))
	key, steps := "VOLUME_UP", level-st.Volume
	if steps < 0 {
		key, steps = "VOLUME_DOWN", -steps
	}
	for i := range steps {
		if i > 0 {
			s.d.Sleep(volumeStepDelay)
		}
		if err := sess.SendKey(key); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) current() (domain.Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.session == nil {
		return nil, domain.ErrNotConnected
	}
	return s.session, nil
}

// sessionEvents forwards a session's callbacks to the UI sink.
type sessionEvents struct{ sink domain.EventSink }

func (e sessionEvents) OnState(st domain.State)  { e.sink.State(st) }
func (e sessionEvents) OnTrace(dir, line string) { e.sink.Trace(dir, line) }
