// Package webos adapts pkg/webos (SSAP) to the domain ports Connector, Pairer
// and Session for LG webOS TVs.
package webos

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"optic/internal/domain"
	"optic/pkg/webos"
)

var (
	// ErrPromptRejected means the user chose not to allow OPTIC on the TV.
	ErrPromptRejected = errors.New("A TV recusou a conexão. Tente de novo e escolha Permitir.")
	// ErrPromptTimeout means nobody answered the prompt on the TV in time.
	ErrPromptTimeout = errors.New("A TV não respondeu ao pedido: confirme 'Permitir' na tela.")
	// ErrPairingLost means the connection dropped before the prompt was answered.
	ErrPairingLost = errors.New("Conexão com a TV perdida durante o pareamento.")
)

const (
	keysFile               = "webos_keys.json"
	defaultPromptTimeout   = 60 * time.Second
	defaultRegisterTimeout = 10 * time.Second
	// pairingLifetime bounds a pairing left open without Close.
	pairingLifetime = 5 * time.Minute

	uriSystemInfo  = "ssap://system/getSystemInfo"
	uriSWInfo      = "ssap://com.webos.service.update/getCurrentSWInformation"
	uriVolume      = "ssap://audio/getVolume"
	uriSetVolume   = "ssap://audio/setVolume"
	uriForeground  = "ssap://com.webos.applicationManager/getForegroundAppInfo"
	uriLaunchPts   = "ssap://com.webos.applicationManager/listLaunchPoints"
	uriLaunch      = "ssap://system.launcher/launch"
	uriOpen        = "ssap://system.launcher/open"
	uriTurnOff     = "ssap://system/turnOff"
	uriInsertText  = "ssap://com.webos.service.ime/insertText"
	uriInputs      = "ssap://tv/getExternalInputList"
	uriSwitchInput = "ssap://tv/switchInput"
	uriToast       = "ssap://system.notifications/createToast"
)

var (
	_ domain.Connector = (*Adapter)(nil)
	_ domain.Pairer    = (*Adapter)(nil)
	_ domain.AppLister = (*session)(nil)
)

// Adapter opens sessions with LG webOS TVs and pairs with them.
type Adapter struct {
	keys keyStore
	// Dial opens the SSAP connection; nil uses webos.Dial (tests only).
	Dial            func(ctx context.Context, host string, opt webos.Options) (*webos.Conn, error)
	promptTimeout   time.Duration
	registerTimeout time.Duration
}

// New returns an adapter keeping the TVs' client keys in cfgDir.
func New(cfgDir string) *Adapter {
	return &Adapter{keys: keyStore{path: filepath.Join(cfgDir, keysFile)}}
}

func (a *Adapter) dial(ctx context.Context, host string, opt webos.Options) (*webos.Conn, error) {
	if a.Dial != nil {
		return a.Dial(ctx, host, opt)
	}
	return webos.Dial(ctx, host, opt)
}

// Connect registers with the stored client key, then follows volume and the
// foreground app. Without a key, or when the TV revoked it, it returns
// domain.ErrNotPaired.
func (a *Adapter) Connect(ctx context.Context, host string, ev domain.SessionEvents) (domain.Session, error) {
	key := a.keys.Get(host)
	if key == "" {
		return nil, domain.ErrNotPaired
	}
	conn, err := a.dial(ctx, host, webos.Options{Trace: ev.OnTrace})
	if err != nil {
		return nil, err
	}
	timeout := a.registerTimeout
	if timeout <= 0 {
		timeout = defaultRegisterTimeout
	}
	rctx, cancel := context.WithTimeout(ctx, timeout)
	newKey, err := conn.Register(rctx, key)
	cancel()
	if err != nil {
		conn.Close()
		// A revoked key is answered with an error, a new prompt, or silence
		// until our deadline: in all cases pair again.
		if errors.Is(err, webos.ErrRegisterRejected) || errors.Is(err, webos.ErrPromptShown) ||
			(errors.Is(err, context.DeadlineExceeded) && ctx.Err() == nil) {
			_ = a.keys.Delete(host)
			return nil, domain.ErrNotPaired
		}
		return nil, err
	}
	if newKey != key {
		_ = a.keys.Put(host, newKey)
	}

	ptr, err := conn.Pointer(ctx)
	if err != nil {
		conn.Close()
		return nil, err
	}
	sctx, scancel := context.WithCancel(context.Background())
	s := &session{
		conn: conn, ptr: ptr, ev: ev, ctx: sctx, cancel: scancel,
		info:  readInfo(ctx, conn),
		state: domain.State{On: true},
	}
	// Safe outside emitMu: the session is not shared yet.
	ev.OnState(s.state)
	if err := conn.Subscribe(ctx, uriVolume, s.onVolume); err != nil {
		s.Close()
		return nil, err
	}
	if err := conn.Subscribe(ctx, uriForeground, s.onForeground); err != nil {
		s.Close()
		return nil, err
	}
	return s, nil
}

// readInfo fetches model and firmware; failures leave the fields empty.
func readInfo(ctx context.Context, conn *webos.Conn) domain.DeviceInfo {
	info := domain.DeviceInfo{
		Vendor:   "LG",
		Platform: domain.PlatformWebOS,
		Capabilities: domain.Capabilities{
			Text: true, AppList: true, AbsoluteVolume: true, Inputs: true, Toast: true,
		},
	}
	if raw, err := conn.Request(ctx, uriSystemInfo, nil); err == nil {
		var p struct {
			ModelName string `json:"modelName"`
		}
		if json.Unmarshal(raw, &p) == nil {
			info.Model = p.ModelName
		}
	}
	if raw, err := conn.Request(ctx, uriSWInfo, nil); err == nil {
		var p struct {
			Product string `json:"product_name"`
			Major   string `json:"major_ver"`
			Minor   string `json:"minor_ver"`
		}
		if json.Unmarshal(raw, &p) == nil {
			ver := p.Major
			if p.Minor != "" {
				ver += "." + p.Minor
			}
			info.Version = strings.TrimSpace(p.Product + " " + ver)
		}
	}
	return info
}

// StartPairing connects and asks the TV to show its "allow" prompt.
func (a *Adapter) StartPairing(ctx context.Context, host string) (domain.PairingSession, error) {
	conn, err := a.dial(ctx, host, webos.Options{})
	if err != nil {
		return nil, err
	}
	timeout := a.promptTimeout
	if timeout <= 0 {
		timeout = defaultPromptTimeout
	}
	rctx, cancel := context.WithTimeout(context.Background(), pairingLifetime)
	p := &pairing{
		conn: conn, keys: a.keys, host: host, timeout: timeout, cancel: cancel,
		res: make(chan regResult, 1),
	}
	go func() {
		key, err := conn.Register(rctx, "")
		p.res <- regResult{key, err}
	}()
	return p, nil
}

type regResult struct {
	key string
	err error
}

// pairing waits for the user to answer the prompt on the TV.
type pairing struct {
	conn    *webos.Conn
	keys    keyStore
	host    string
	timeout time.Duration
	cancel  context.CancelFunc
	res     chan regResult

	mu       sync.Mutex
	answered bool
	final    error
}

func (p *pairing) Mode() string       { return domain.PairingPrompt }
func (p *pairing) ServerName() string { return "TV LG" }

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
		case errors.Is(r.err, webos.ErrRegisterRejected):
			p.final = ErrPromptRejected
		case errors.Is(r.err, context.DeadlineExceeded):
			p.final = ErrPromptTimeout
		case r.err != nil:
			p.final = ErrPairingLost
		default:
			p.final = p.keys.Put(p.host, r.key)
		}
		return p.final
	case <-t.C:
		return ErrPromptTimeout
	}
}

func (p *pairing) Close() error {
	p.cancel()
	return p.conn.Close()
}

// Pointer buttons for OPTIC's (Android-style) key names. MEDIA_PLAY_PAUSE and
// POWER are handled apart.
var lgButtons = map[string]string{
	"HOME": "HOME", "BACK": "BACK",
	"DPAD_UP": "UP", "DPAD_DOWN": "DOWN", "DPAD_LEFT": "LEFT", "DPAD_RIGHT": "RIGHT",
	"DPAD_CENTER": "ENTER", "ENTER": "ENTER",
	"VOLUME_UP": "VOLUMEUP", "VOLUME_DOWN": "VOLUMEDOWN", "MUTE": "MUTE",
	"CHANNEL_UP": "CHANNELUP", "CHANNEL_DOWN": "CHANNELDOWN",
	"MEDIA_STOP": "STOP", "MEDIA_REWIND": "REWIND", "MEDIA_FAST_FORWARD": "FASTFORWARD",
	"MENU": "MENU", "SETTINGS": "MENU", "GUIDE": "GUIDE",
}

type session struct {
	conn   *webos.Conn
	ev     domain.SessionEvents
	ctx    context.Context
	cancel context.CancelFunc
	info   domain.DeviceInfo

	ptrMu  sync.Mutex // guards ptr and paused; serializes buttons
	ptr    *webos.Pointer
	paused bool // last MEDIA_PLAY_PAUSE sent PAUSE

	mu    sync.Mutex // guards state and apps
	state domain.State
	apps  []domain.App

	// emitMu serializes OnState with Close. OnState runs with it held, so it
	// must not call Close.
	emitMu sync.Mutex
	closed bool
}

var (
	_ domain.VolumeSetter = (*session)(nil)
	_ domain.InputLister  = (*session)(nil)
	_ domain.Notifier     = (*session)(nil)
	_ domain.AppLister    = (*session)(nil)
)

func (s *session) Info() domain.DeviceInfo { return s.info }

func (s *session) State() domain.State {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}

func (s *session) Done() <-chan struct{} { return s.conn.Done() }

// Close closes the connection. It is safe to call more than once.
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

// onVolume handles both payload shapes: webOS 5+ nests the values in
// volumeStatus, older versions send volume and muted at the top level.
func (s *session) onVolume(raw json.RawMessage) {
	var p struct {
		Volume    *int  `json:"volume"`
		Muted     *bool `json:"muted"`
		MaxVolume int   `json:"maxVolume"`
		Status    *struct {
			Volume     *int  `json:"volume"`
			MuteStatus *bool `json:"muteStatus"`
			MaxVolume  int   `json:"maxVolume"`
		} `json:"volumeStatus"`
	}
	if json.Unmarshal(raw, &p) != nil {
		return
	}
	vol, muted, maxVol := p.Volume, p.Muted, p.MaxVolume
	if p.Status != nil {
		vol, muted, maxVol = p.Status.Volume, p.Status.MuteStatus, p.Status.MaxVolume
	}
	if vol == nil {
		return
	}
	if maxVol <= 0 {
		maxVol = 100
	}
	s.update(func(st *domain.State) {
		st.Volume, st.VolumeMax = *vol, maxVol
		if muted != nil {
			st.Muted = *muted
		}
	})
}

func (s *session) onForeground(raw json.RawMessage) {
	var p struct {
		AppID *string `json:"appId"`
	}
	if json.Unmarshal(raw, &p) != nil || p.AppID == nil {
		return
	}
	s.update(func(st *domain.State) { st.App = *p.AppID })
}

func (s *session) SendKey(name string) error {
	n := strings.TrimPrefix(strings.ToUpper(strings.TrimSpace(name)), "KEYCODE_")
	if n == "POWER" {
		_, err := s.conn.Request(s.ctx, uriTurnOff, nil)
		return err
	}
	s.ptrMu.Lock()
	defer s.ptrMu.Unlock()
	if n == "MEDIA_PLAY_PAUSE" {
		button := "PAUSE"
		if s.paused {
			button = "PLAY"
		}
		if err := s.button(button); err != nil {
			return err
		}
		s.paused = !s.paused
		return nil
	}
	button, ok := lgButtons[n]
	if !ok {
		return fmt.Errorf("Tecla não suportada pela LG: %s", name)
	}
	return s.button(button)
}

// button presses a pointer button, reopening the pointer socket once if the
// TV dropped it. Callers hold ptrMu.
func (s *session) button(name string) error {
	s.ev.OnTrace("→", "button "+name)
	if s.ptr.Button(name) == nil {
		return nil
	}
	ptr, err := s.conn.Pointer(s.ctx)
	if err != nil {
		return err
	}
	s.ptr = ptr
	return ptr.Button(name)
}

// SendText types text in the focused field. pkg/webos never traces it.
func (s *session) SendText(text string) error {
	_, err := s.conn.Request(s.ctx, uriInsertText, map[string]any{"text": text, "replace": false})
	return err
}

// LaunchApp opens a web link in the browser, or launches an installed app
// named by a market link, an app ID or its title.
func (s *session) LaunchApp(target string) error {
	if isLink(target) {
		_, err := s.conn.Request(s.ctx, uriOpen, map[string]any{"target": strings.TrimSpace(target)})
		return err
	}
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
	_, err = s.conn.Request(s.ctx, uriLaunch, map[string]any{"id": id})
	return err
}

// Apps lists the installed apps and refreshes the cache.
func (s *session) Apps(ctx context.Context) ([]domain.App, error) {
	raw, err := s.conn.Request(ctx, uriLaunchPts, nil)
	if err != nil {
		return nil, err
	}
	var p struct {
		LaunchPoints []struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		} `json:"launchPoints"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("webos: lista de apps inválida: %w", err)
	}
	out := make([]domain.App, 0, len(p.LaunchPoints))
	for _, lp := range p.LaunchPoints {
		if lp.ID != "" {
			out = append(out, domain.App{ID: lp.ID, Name: lp.Title})
		}
	}
	s.mu.Lock()
	s.apps = out
	s.mu.Unlock()
	return out, nil
}

// installed returns the cached app list, fetching it when empty or refresh.
func (s *session) installed(refresh bool) ([]domain.App, error) {
	s.mu.Lock()
	cached := s.apps
	s.mu.Unlock()
	if cached != nil && !refresh {
		return cached, nil
	}
	return s.Apps(s.ctx)
}

// SetVolume sets the exact volume level.
func (s *session) SetVolume(ctx context.Context, level int) error {
	_, err := s.conn.Request(ctx, uriSetVolume, map[string]any{"volume": level})
	return err
}

// Inputs lists the TV's external inputs.
func (s *session) Inputs(ctx context.Context) ([]domain.Input, error) {
	raw, err := s.conn.Request(ctx, uriInputs, nil)
	if err != nil {
		return nil, err
	}
	var p struct {
		Devices []struct {
			ID    string `json:"id"`
			Label string `json:"label"`
		} `json:"devices"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("webos: lista de entradas inválida: %w", err)
	}
	out := make([]domain.Input, 0, len(p.Devices))
	for _, d := range p.Devices {
		out = append(out, domain.Input{ID: d.ID, Name: d.Label})
	}
	return out, nil
}

// SwitchInput switches to the input with the given ID.
func (s *session) SwitchInput(ctx context.Context, id string) error {
	_, err := s.conn.Request(ctx, uriSwitchInput, map[string]any{"inputId": id})
	return err
}

// Toast shows a short message on the TV screen.
func (s *session) Toast(ctx context.Context, msg string) error {
	_, err := s.conn.Request(ctx, uriToast, map[string]any{"message": msg})
	return err
}
