// Package tizen adapts pkg/tizen to the domain ports Connector, Pairer and
// Session for Samsung Tizen TVs.
package tizen

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	"optic/internal/adapter/keystore"
	"optic/internal/domain"
	"optic/pkg/tizen"
)

var (
	// ErrPromptRejected means the user chose not to allow OPTIC on the TV.
	ErrPromptRejected = errors.New("A TV recusou a conexão. Tente de novo e escolha Permitir.")
	// ErrPromptTimeout means nobody answered the prompt on the TV in time.
	ErrPromptTimeout = errors.New("A TV não respondeu ao pedido: confirme 'Permitir' na tela.")
	// ErrPairingLost means the connection dropped before the prompt was answered.
	ErrPairingLost = errors.New("Conexão com a TV perdida durante o pareamento.")
	// ErrTVNotResponding means the TV did not answer; the pairing is kept.
	ErrTVNotResponding = errors.New("A TV Samsung não respondeu. Verifique se ela está ligada e tente de novo.")
	// ErrKeyboardClosed means text was sent with no text field open on the TV.
	ErrKeyboardClosed = errors.New("Abra um campo de texto na TV antes de enviar.")
)

const (
	tokensFile            = "samsung_tokens.json"
	defaultPromptTimeout  = 60 * time.Second
	defaultConnectTimeout = 10 * time.Second
	defaultPollInterval   = 5 * time.Second
	restTimeout           = 3 * time.Second
	// pairingLifetime bounds a pairing left open without Close.
	pairingLifetime = 5 * time.Minute
)

var (
	_ domain.Connector = (*Adapter)(nil)
	_ domain.Pairer    = (*Adapter)(nil)
	_ domain.Session   = (*session)(nil)
	_ domain.AppLister = (*session)(nil)
)

// Adapter opens sessions with Samsung Tizen TVs and pairs with them.
type Adapter struct {
	keys keystore.Store
	http *http.Client
	// Dial opens the remote-control socket; nil uses tizen.Dial (tests only).
	Dial func(ctx context.Context, host, token string, secure bool, opt tizen.Options) (*tizen.Conn, string, error)
	// InfoBase gives the REST base URL for host; nil uses host:8001 (tests only).
	InfoBase func(host string) string

	promptTimeout  time.Duration
	connectTimeout time.Duration
	pollInterval   time.Duration
}

// New returns an adapter keeping the TVs' tokens in cfgDir.
func New(cfgDir string) *Adapter {
	return &Adapter{
		keys: keystore.Store{Path: filepath.Join(cfgDir, tokensFile)},
		http: &http.Client{
			Timeout:       restTimeout,
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
	}
}

func (a *Adapter) dial(ctx context.Context, host, token string, secure bool, opt tizen.Options) (*tizen.Conn, string, error) {
	if a.Dial != nil {
		return a.Dial(ctx, host, token, secure, opt)
	}
	return tizen.Dial(ctx, host, token, secure, opt)
}

func (a *Adapter) info(ctx context.Context, host string) (tizen.Info, error) {
	base := host
	if a.InfoBase != nil {
		base = a.InfoBase(host)
	}
	return tizen.GetInfo(ctx, a.http, base)
}

// notResponding hides raw network text from the UI.
func notResponding(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return ErrTVNotResponding
}

// Connect reads the REST info, then opens the remote socket: with the stored
// token on TVs that use tokens, plainly on older ones. Without a token, or
// when the TV revoked it, it returns domain.ErrNotPaired.
func (a *Adapter) Connect(ctx context.Context, host string, ev domain.SessionEvents) (domain.Session, error) {
	info, err := a.info(ctx, host)
	if err != nil {
		return nil, notResponding(ctx, err)
	}
	token := ""
	if info.TokenAuth {
		if token = a.keys.Get(host); token == "" {
			return nil, domain.ErrNotPaired
		}
	}
	timeout := a.connectTimeout
	if timeout <= 0 {
		timeout = defaultConnectTimeout
	}
	dctx, cancel := context.WithTimeout(ctx, timeout)
	conn, newToken, err := a.dial(dctx, host, token, info.TokenAuth, tizen.Options{Trace: ev.OnTrace})
	cancel()
	if err != nil {
		if errors.Is(err, tizen.ErrUnauthorized) {
			if !info.TokenAuth {
				// No token involved: the user denied the prompt on the TV.
				return nil, ErrPromptRejected
			}
			_ = a.keys.Delete(host)
			return nil, domain.ErrNotPaired
		}
		// Silence or a refused dial is not a revoked token: keep it.
		return nil, notResponding(ctx, err)
	}
	if info.TokenAuth && newToken != "" && newToken != token {
		_ = a.keys.Put(host, newToken)
	}

	interval := a.pollInterval
	if interval <= 0 {
		interval = defaultPollInterval
	}
	sctx, scancel := context.WithCancel(context.Background())
	s := &session{
		conn: conn, ev: ev, ctx: sctx, cancel: scancel,
		poll: func(ctx context.Context) (tizen.Info, error) { return a.info(ctx, host) },
		info: domain.DeviceInfo{
			Model:        info.Model,
			Vendor:       "Samsung",
			Version:      info.OS,
			Platform:     domain.PlatformTizen,
			Capabilities: domain.Capabilities{Text: true, AppList: true},
		},
		state: domain.State{On: info.PowerState == "on"},
	}
	// Safe outside emitMu: the session is not shared yet.
	ev.OnState(s.state)
	conn.OnIME(func(open bool) {
		s.update(func(st *domain.State) { st.Keyboard = open })
	})
	go s.pollLoop(interval)
	return s, nil
}

// StartPairing reads the TV name and opens the socket without a token, which
// makes the TV show its "allow" prompt.
func (a *Adapter) StartPairing(ctx context.Context, host string) (domain.PairingSession, error) {
	info, err := a.info(ctx, host)
	if err != nil {
		return nil, notResponding(ctx, err)
	}
	name := info.Name
	if name == "" {
		name = "TV Samsung"
	}
	timeout := a.promptTimeout
	if timeout <= 0 {
		timeout = defaultPromptTimeout
	}
	rctx, cancel := context.WithTimeout(context.Background(), pairingLifetime)
	p := &pairing{
		keys: a.keys, host: host, name: name, timeout: timeout, cancel: cancel,
		res: make(chan dialResult, 1),
	}
	go func() {
		conn, token, err := a.dial(rctx, host, "", true, tizen.Options{})
		if conn != nil {
			// Only the token is needed: Connect opens a fresh socket.
			conn.Close()
		}
		p.res <- dialResult{token, err}
	}()
	return p, nil
}

type dialResult struct {
	token string
	err   error
}

// pairing waits for the user to answer the prompt on the TV.
type pairing struct {
	keys    keystore.Store
	host    string
	name    string
	timeout time.Duration
	cancel  context.CancelFunc
	res     chan dialResult

	mu       sync.Mutex
	answered bool
	final    error
}

func (p *pairing) Mode() string       { return domain.PairingPrompt }
func (p *pairing) ServerName() string { return p.name }

// Finish ignores code: it waits for the user to answer the prompt.
func (p *pairing) Finish(string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.answered {
		return p.final
	}
	t := time.NewTimer(p.timeout)
	defer t.Stop()
	select {
	case r := <-p.res:
		p.answered = true
		switch {
		case errors.Is(r.err, tizen.ErrUnauthorized):
			p.final = ErrPromptRejected
		case errors.Is(r.err, tizen.ErrTimeout):
			p.final = ErrPromptTimeout
		case r.err != nil, r.token == "":
			// A TV that accepts without a token cannot be reconnected to.
			p.final = ErrPairingLost
		default:
			p.final = p.keys.Put(p.host, r.token)
		}
		return p.final
	case <-t.C:
		return ErrPromptTimeout
	}
}

// Close stops a pending prompt wait; the socket closes when the dial returns.
func (p *pairing) Close() error {
	p.cancel()
	return nil
}

type session struct {
	conn   *tizen.Conn
	ev     domain.SessionEvents
	ctx    context.Context
	cancel context.CancelFunc
	info   domain.DeviceInfo
	poll   func(ctx context.Context) (tizen.Info, error)

	keyMu  sync.Mutex // guards paused; serializes play/pause
	paused bool       // last MEDIA_PLAY_PAUSE sent KEY_PAUSE

	mu    sync.Mutex // guards state and apps
	state domain.State
	apps  []tizen.App

	// emitMu serializes OnState with Close. OnState runs with it held, so it
	// must not call Close.
	emitMu sync.Mutex
	closed bool
}

func (s *session) Info() domain.DeviceInfo { return s.info }

func (s *session) State() domain.State {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}

func (s *session) Done() <-chan struct{} { return s.conn.Done() }

// Close closes the connection and stops the poll. Safe to call more than once.
func (s *session) Close() error {
	s.emitMu.Lock()
	s.closed = true
	s.emitMu.Unlock()
	s.cancel()
	return s.conn.Close()
}

// update applies fn to the state and reports it when it changed.
func (s *session) update(fn func(*domain.State)) {
	s.emitMu.Lock()
	defer s.emitMu.Unlock()
	if s.closed {
		return
	}
	s.mu.Lock()
	st := s.state
	fn(&st)
	changed := st != s.state
	s.state = st
	s.mu.Unlock()
	if changed {
		s.ev.OnState(st)
	}
}

// pollLoop follows the power state over REST. A REST failure alone does not
// end the session; only the socket dropping does.
func (s *session) pollLoop(interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-s.conn.Done():
			return
		case <-t.C:
			info, err := s.poll(s.ctx)
			if err != nil {
				continue
			}
			on := info.PowerState == "on"
			s.update(func(st *domain.State) { st.On = on })
		}
	}
}

func (s *session) SendKey(name string) error {
	n := keyName(name)
	if n == "MEDIA_PLAY_PAUSE" {
		s.keyMu.Lock()
		defer s.keyMu.Unlock()
		key := "KEY_PAUSE"
		if s.paused {
			key = "KEY_PLAY"
		}
		if err := s.conn.Key(key); err != nil {
			return err
		}
		s.paused = !s.paused
		return nil
	}
	key, ok := samsungKeys[n]
	if !ok {
		return fmt.Errorf("Tecla não suportada pela Samsung: %s", name)
	}
	return s.conn.Key(key)
}

// SendText types text in the open TV keyboard. pkg/tizen never traces it.
func (s *session) SendText(text string) error {
	if !s.State().Keyboard {
		return ErrKeyboardClosed
	}
	return s.conn.Text(text)
}

// LaunchApp launches an installed app named by a market link, an app ID or
// its title.
func (s *session) LaunchApp(target string) error {
	installed, err := s.installed(false)
	if err != nil {
		return err
	}
	app, err := pick(target, installed)
	if errors.Is(err, ErrAppNotInstalled) {
		if installed, err = s.installed(true); err != nil {
			return err
		}
		app, err = pick(target, installed)
	}
	if err != nil {
		return err
	}
	return s.conn.Launch(app.ID, app.Type)
}

// pick resolves target among the installed apps, keeping the app type.
func pick(target string, installed []tizen.App) (tizen.App, error) {
	id, err := resolveApp(target, toDomain(installed))
	if err != nil {
		return tizen.App{}, err
	}
	for _, a := range installed {
		if a.ID == id {
			return a, nil
		}
	}
	return tizen.App{}, ErrAppNotInstalled
}

func toDomain(apps []tizen.App) []domain.App {
	out := make([]domain.App, 0, len(apps))
	for _, a := range apps {
		if a.ID != "" {
			out = append(out, domain.App{ID: a.ID, Name: a.Name})
		}
	}
	return out
}

// Apps lists the installed apps and refreshes the cache.
func (s *session) Apps(ctx context.Context) ([]domain.App, error) {
	apps, err := s.conn.Apps(ctx)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.apps = apps
	s.mu.Unlock()
	return toDomain(apps), nil
}

// installed returns the cached app list, fetching it when empty or refresh.
func (s *session) installed(refresh bool) ([]tizen.App, error) {
	s.mu.Lock()
	cached := s.apps
	s.mu.Unlock()
	if cached != nil && !refresh {
		return cached, nil
	}
	if _, err := s.Apps(s.ctx); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.apps, nil
}
