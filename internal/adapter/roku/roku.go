// Package roku adapts pkg/roku (ECP) to the domain ports Connector, Pairer and
// Session.
package roku

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"optic/internal/domain"
	"optic/pkg/roku"
)

// ErrNoPairingNeeded is returned by StartPairing: Roku has no pairing step.
var ErrNoPairingNeeded = errors.New("TVs Roku não precisam de pareamento")

const (
	defaultPoll     = 3 * time.Second
	defaultMaxFails = 3
)

// Adapter opens sessions with Rokus. The zero value is usable.
type Adapter struct {
	HTTP *http.Client
	// BaseURL, when set, gives the ECP base URL for a host (tests only).
	BaseURL  func(host string) string
	Poll     time.Duration
	MaxFails int
}

// quick returns c with a 1 s timeout, for optional endpoints.
func (a Adapter) quick(c roku.Client) roku.Client {
	h := http.Client{}
	if c.HTTP != nil {
		h = *c.HTTP
	}
	h.Timeout = time.Second
	c.HTTP = &h
	return c
}

func (a Adapter) client(host string) roku.Client {
	c := roku.Client{HTTP: a.HTTP, Host: host}
	if a.BaseURL != nil {
		c.BaseURL = a.BaseURL(host)
	}
	return c
}

// Connect reads the device info, then polls its state until closed or lost.
func (a Adapter) Connect(ctx context.Context, host string, ev domain.SessionEvents) (domain.Session, error) {
	c := a.client(host)
	di, err := c.DeviceInfo(ctx)
	if err != nil {
		return nil, err
	}
	app, err := c.ActiveApp(ctx)
	if err != nil {
		return nil, err
	}
	poll, maxFails := a.Poll, a.MaxFails
	if poll <= 0 {
		poll = defaultPoll
	}
	if maxFails <= 0 {
		maxFails = defaultMaxFails
	}
	sctx, cancel := context.WithCancel(context.Background())
	s := &session{
		c: c, ev: ev, ctx: sctx, cancel: cancel,
		info: domain.DeviceInfo{
			Model: di.Model, Vendor: di.Vendor, Version: di.Software,
			Platform:     domain.PlatformRoku,
			Capabilities: domain.Capabilities{Text: true, AppList: true, PowerOn: true, Inputs: di.IsTV},
		},
		isTV:  di.IsTV,
		state: toState(di, app, playing(a.quick(c).MediaPlayer(ctx))),
		done:  make(chan struct{}),
	}
	ev.OnState(s.state)
	s.quick = a.quick(c)
	go s.poll(poll, maxFails)
	return s, nil
}

// StartPairing always fails: Roku needs no pairing.
func (a Adapter) StartPairing(ctx context.Context, host string) (domain.PairingSession, error) {
	return nil, ErrNoPairingNeeded
}

// PowerOn wakes a Roku TV; in fast-start standby ECP stays reachable.
func (a Adapter) PowerOn(ctx context.Context, host string) error {
	return a.client(host).Keypress(ctx, "PowerOn")
}

func toState(di roku.DeviceInfo, app roku.App, playing string) domain.State {
	return domain.State{On: di.On(), App: app.Name, Playing: playing}
}

// playing maps the media-player state. An error (older Rokus lack the
// endpoint) means unknown, never a failure.
func playing(p roku.Player, err error) string {
	if err != nil {
		return ""
	}
	switch p.State {
	case "play":
		return "playing"
	case "pause":
		return "paused"
	}
	return ""
}

// rokuInputs are the TV inputs a Roku TV accepts as keypresses.
var rokuInputs = []domain.Input{
	{ID: "InputTuner", Name: "TV"},
	{ID: "InputHDMI1", Name: "HDMI 1"},
	{ID: "InputHDMI2", Name: "HDMI 2"},
	{ID: "InputHDMI3", Name: "HDMI 3"},
	{ID: "InputHDMI4", Name: "HDMI 4"},
	{ID: "InputAV1", Name: "AV"},
}

type session struct {
	c      roku.Client
	quick  roku.Client // 1 s timeout, optional calls
	ev     domain.SessionEvents
	ctx    context.Context
	cancel context.CancelFunc
	info   domain.DeviceInfo
	isTV   bool

	mu    sync.Mutex // guards state and apps
	state domain.State
	apps  []roku.App

	// emitMu serializes OnState with Close. OnState runs with it held, so it
	// must not call Close.
	emitMu sync.Mutex
	closed bool

	done     chan struct{}
	doneOnce sync.Once
}

var _ domain.InputLister = (*session)(nil)

func (s *session) Info() domain.DeviceInfo { return s.info }

func (s *session) State() domain.State {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}

func (s *session) Done() <-chan struct{} { return s.done }

func (s *session) finish() { s.doneOnce.Do(func() { close(s.done) }) }

// Close stops polling. It is safe to call more than once.
func (s *session) Close() error {
	s.emitMu.Lock()
	s.closed = true
	s.emitMu.Unlock()
	s.cancel()
	s.finish()
	return nil
}

func (s *session) poll(every time.Duration, maxFails int) {
	t := time.NewTicker(every)
	defer t.Stop()
	fails := 0
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-t.C:
		}
		di, err := s.c.DeviceInfo(s.ctx)
		var app roku.App
		if err == nil {
			app, err = s.c.ActiveApp(s.ctx)
		}
		if err != nil {
			if s.ctx.Err() != nil {
				return
			}
			fails++
			if fails >= maxFails {
				s.cancel()
				s.finish()
				return
			}
			continue
		}
		fails = 0
		s.update(toState(di, app, playing(s.quick.MediaPlayer(s.ctx))))
	}
}

func (s *session) update(st domain.State) {
	s.emitMu.Lock()
	defer s.emitMu.Unlock()
	if s.closed {
		return
	}
	s.mu.Lock()
	changed := st != s.state
	s.state = st
	s.mu.Unlock()
	if changed {
		s.ev.OnState(st)
	}
}

// Inputs lists the TV inputs; only Roku TVs have them.
func (s *session) Inputs(ctx context.Context) ([]domain.Input, error) {
	if !s.isTV {
		return nil, domain.ErrUnsupported{Platform: domain.PlatformRoku}
	}
	return append([]domain.Input(nil), rokuInputs...), nil
}

// SwitchInput presses the input key, only for ids from Inputs.
func (s *session) SwitchInput(ctx context.Context, id string) error {
	if !s.isTV {
		return domain.ErrUnsupported{Platform: domain.PlatformRoku}
	}
	for _, in := range rokuInputs {
		if in.ID == id {
			s.ev.OnTrace("→", "keypress "+id)
			return s.c.Keypress(ctx, id)
		}
	}
	return errors.New("Entrada desconhecida")
}

func (s *session) SendKey(name string) error {
	key, ok := rokuKey(name)
	if !ok {
		return fmt.Errorf("Tecla não suportada pela Roku: %s", name)
	}
	s.ev.OnTrace("→", "keypress "+key)
	return s.c.Keypress(s.ctx, key)
}

// SendText traces one line per call, never the typed characters.
func (s *session) SendText(text string) error {
	s.ev.OnTrace("→", "keypress Lit_… [texto omitido]")
	return s.c.Text(s.ctx, text)
}

func (s *session) LaunchApp(target string) error {
	installed, err := s.installed(false)
	if err != nil {
		return err
	}
	id, err := resolveApp(target, installed)
	if errors.Is(err, ErrAppNotInstalled) {
		if installed, err = s.installed(true); err != nil {
			return err
		}
		id, err = resolveApp(target, installed)
	}
	if err != nil {
		return err
	}
	s.ev.OnTrace("→", "launch "+id)
	return s.c.Launch(s.ctx, id)
}

// Apps lists the installed channels and refreshes the cache.
func (s *session) Apps(ctx context.Context) ([]domain.App, error) {
	list, err := s.c.Apps(ctx)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.apps = list
	s.mu.Unlock()
	out := make([]domain.App, 0, len(list))
	for _, a := range list {
		if a.ID != "" && a.Type != "tvin" {
			out = append(out, domain.App{ID: a.ID, Name: a.Name})
		}
	}
	return out, nil
}

// installed returns the cached app list, fetching it when empty or refresh.
func (s *session) installed(refresh bool) ([]roku.App, error) {
	s.mu.Lock()
	cached := s.apps
	s.mu.Unlock()
	if cached != nil && !refresh {
		return cached, nil
	}
	list, err := s.c.Apps(s.ctx)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.apps = list
	s.mu.Unlock()
	return list, nil
}
