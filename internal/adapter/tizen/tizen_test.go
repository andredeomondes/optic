package tizen

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"optic/internal/domain"
	"optic/pkg/tizen"

	"github.com/gorilla/websocket"
)

const host = "10.0.0.7"

var upgrader = websocket.Upgrader{}

// fakeTV serves the REST info on /api/v2/ and the remote-control socket.
type fakeTV struct {
	t    *testing.T
	base string // http://127.0.0.1:port
	url  string // ws://127.0.0.1:port

	mu        sync.Mutex // guards the fields below once the server runs
	connect   string     // "accept" (default), "unauthorized", "timeout", "silent", "drop"
	issue     string     // token sent on accept; default echoes the query token or "tok-new"
	power     string
	tokenAuth bool
	restFail  bool
	ws        *websocket.Conn // last socket
	apps      []map[string]any
	model     string        // REST device.model
	imeSynced bool          // REST ImeSyncedSupport
	appsMode  string        // "" answers; "silent" never answers the app list
	timeout   time.Duration // client request timeout; 0 keeps the default

	msgs   chan map[string]any // frames the TV received
	closed chan struct{}       // a socket was closed by the client
	tokens chan string         // token query of each socket
}

func newFakeTV(t *testing.T) *fakeTV {
	tv := &fakeTV{
		t:         t,
		power:     "on",
		tokenAuth: true,
		model:     "18_KANTM2_QTV",
		imeSynced: true,
		apps: []map[string]any{
			{"appId": "3201907018807", "name": "Netflix", "app_type": 2},
			{"appId": "111299001912", "name": "YouTube", "app_type": 2},
			{"appId": "org.tizen.browser", "name": "Internet", "app_type": 4},
		},
		msgs:   make(chan map[string]any, 64),
		closed: make(chan struct{}, 8),
		tokens: make(chan string, 8),
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/" {
			tv.rest(w)
			return
		}
		tv.tokens <- r.URL.Query().Get("token")
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer ws.Close()
		tv.serve(ws, r.URL.Query().Get("token"))
	}))
	t.Cleanup(srv.Close)
	tv.base = srv.URL
	tv.url = "ws" + strings.TrimPrefix(srv.URL, "http")
	return tv
}

func (tv *fakeTV) set(fn func(tv *fakeTV)) {
	tv.mu.Lock()
	fn(tv)
	tv.mu.Unlock()
}

func (tv *fakeTV) rest(w http.ResponseWriter) {
	tv.mu.Lock()
	defer tv.mu.Unlock()
	if tv.restFail {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	auth := "false"
	if tv.tokenAuth {
		auth = "true"
	}
	dev := map[string]any{
		"name": "[TV] Sala", "modelName": "QN55Q60", "PowerState": tv.power,
		"OS": "Tizen", "TokenAuthSupport": auth, "model": tv.model,
	}
	if tv.imeSynced {
		dev["ImeSyncedSupport"] = "true"
	}
	json.NewEncoder(w).Encode(map[string]any{"device": dev})
}

func (tv *fakeTV) serve(ws *websocket.Conn, token string) {
	tv.mu.Lock()
	tv.ws = ws
	mode, issue := tv.connect, tv.issue
	tv.mu.Unlock()
	if issue == "" {
		issue = token
		if issue == "" {
			issue = "tok-new"
		}
	}
	tv.write(func() {
		switch mode {
		case "unauthorized":
			ws.WriteJSON(map[string]any{"event": "ms.channel.unauthorized"})
		case "timeout":
			ws.WriteJSON(map[string]any{"event": "ms.channel.timeOut"})
		case "silent", "drop":
		default:
			ws.WriteJSON(map[string]any{"event": "ms.channel.connect", "data": map[string]any{"token": issue}})
		}
	})
	if mode == "drop" {
		return
	}
	for {
		var m map[string]any
		if err := ws.ReadJSON(&m); err != nil {
			tv.closed <- struct{}{}
			return
		}
		tv.msgs <- m
		if p, _ := m["params"].(map[string]any); p["event"] == "ed.installedApp.get" {
			tv.mu.Lock()
			apps, mode := tv.apps, tv.appsMode
			tv.mu.Unlock()
			if mode == "silent" {
				continue
			}
			tv.write(func() {
				ws.WriteJSON(map[string]any{"event": "ed.installedApp.get", "data": map[string]any{"data": apps}})
			})
		}
	}
}

// write serializes server writes on the last socket.
func (tv *fakeTV) write(fn func()) {
	tv.mu.Lock()
	defer tv.mu.Unlock()
	fn()
}

// event pushes an event on the last socket.
func (tv *fakeTV) event(name string) {
	tv.write(func() { tv.ws.WriteJSON(map[string]any{"event": name}) })
}

// drop closes the last socket from the TV side.
func (tv *fakeTV) drop() {
	tv.write(func() { tv.ws.Close() })
}

func (tv *fakeTV) next(t *testing.T) map[string]any {
	t.Helper()
	select {
	case m := <-tv.msgs:
		p, _ := m["params"].(map[string]any)
		return p
	case <-time.After(2 * time.Second):
		t.Fatal("no message")
		return nil
	}
}

func (tv *fakeTV) noIO(t *testing.T) {
	t.Helper()
	select {
	case m := <-tv.msgs:
		t.Errorf("unexpected message %v", m)
	case <-time.After(100 * time.Millisecond):
	}
}

type events struct {
	states chan domain.State
	mu     sync.Mutex
	lines  []string
}

func newEvents() *events { return &events{states: make(chan domain.State, 64)} }

func (e *events) OnState(s domain.State) { e.states <- s }
func (e *events) OnTrace(dir, line string) {
	e.mu.Lock()
	e.lines = append(e.lines, dir+" "+line)
	e.mu.Unlock()
}
func (e *events) traces() string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return strings.Join(e.lines, "\n")
}

// waitState waits for a state matching ok.
func (e *events) waitState(t *testing.T, ok func(domain.State) bool) domain.State {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		select {
		case st := <-e.states:
			if ok(st) {
				return st
			}
		case <-deadline:
			t.Fatal("state never matched")
			return domain.State{}
		}
	}
}

type dialRec struct {
	secure bool
	token  string
}

func newAdapter(t *testing.T, tv *fakeTV) (*Adapter, chan dialRec) {
	a := New(t.TempDir())
	dials := make(chan dialRec, 8)
	a.Dial = func(ctx context.Context, h, token string, secure bool, opt tizen.Options) (*tizen.Conn, string, error) {
		dials <- dialRec{secure, token}
		opt.URL = tv.url
		tv.mu.Lock()
		opt.Timeout = tv.timeout
		tv.mu.Unlock()
		return tizen.Dial(ctx, h, token, secure, opt)
	}
	a.InfoBase = func(string) string { return tv.base }
	return a, dials
}

// connect stores a token and opens a session.
func connect(t *testing.T, tv *fakeTV) (*session, *events) {
	t.Helper()
	a, _ := newAdapter(t, tv)
	if err := a.keys.Put(host, "tok1"); err != nil {
		t.Fatal(err)
	}
	ev := newEvents()
	s, err := a.Connect(context.Background(), host, ev)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s.(*session), ev
}

func TestConnectWithoutTokenNotPaired(t *testing.T) {
	tv := newFakeTV(t)
	a, dials := newAdapter(t, tv)
	_, err := a.Connect(context.Background(), host, newEvents())
	if !errors.Is(err, domain.ErrNotPaired) {
		t.Fatalf("err = %v", err)
	}
	if len(dials) != 0 {
		t.Error("dialed without a token")
	}
}

func TestPairingStoresToken(t *testing.T) {
	tv := newFakeTV(t)
	a, dials := newAdapter(t, tv)
	p, err := a.StartPairing(context.Background(), host)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	if p.Mode() != domain.PairingPrompt || p.ServerName() != "[TV] Sala" {
		t.Errorf("mode=%q name=%q", p.Mode(), p.ServerName())
	}
	if err := p.Finish("ignored"); err != nil {
		t.Fatal(err)
	}
	if d := <-dials; !d.secure || d.token != "" {
		t.Errorf("pairing dial = %+v", d)
	}
	if got := a.keys.Get(host); got != "tok-new" {
		t.Fatalf("stored token = %q", got)
	}
	select {
	case <-tv.closed:
	case <-time.After(2 * time.Second):
		t.Error("pairing socket left open")
	}
	if err := p.Finish(""); err != nil {
		t.Errorf("second Finish = %v", err)
	}

	s, err := a.Connect(context.Background(), host, newEvents())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if d := <-dials; !d.secure || d.token != "tok-new" {
		t.Errorf("connect dial = %+v", d)
	}
}

func TestPairingServerNameFallback(t *testing.T) {
	tv := newFakeTV(t)
	a, _ := newAdapter(t, tv)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"device":{"TokenAuthSupport":"true"}}`))
	}))
	defer srv.Close()
	a.InfoBase = func(string) string { return srv.URL }
	p, err := a.StartPairing(context.Background(), host)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	if p.ServerName() != "TV Samsung" {
		t.Errorf("name = %q", p.ServerName())
	}
}

func TestPairingFailures(t *testing.T) {
	for _, c := range []struct {
		mode string
		want error
		text string
	}{
		{"unauthorized", ErrPromptRejected, "A TV recusou a conexão. Tente de novo e escolha Permitir."},
		{"timeout", ErrPromptTimeout, "A TV não respondeu ao pedido: confirme 'Permitir' na tela."},
		{"silent", ErrPromptTimeout, "A TV não respondeu ao pedido: confirme 'Permitir' na tela."},
		{"drop", ErrPairingLost, "Conexão com a TV perdida durante o pareamento."},
	} {
		t.Run(c.mode, func(t *testing.T) {
			tv := newFakeTV(t)
			tv.connect = c.mode
			a, _ := newAdapter(t, tv)
			a.promptTimeout = 200 * time.Millisecond
			p, err := a.StartPairing(context.Background(), host)
			if err != nil {
				t.Fatal(err)
			}
			defer p.Close()
			err = p.Finish("")
			if !errors.Is(err, c.want) || err.Error() != c.text {
				t.Fatalf("err = %v", err)
			}
			if a.keys.Get(host) != "" {
				t.Error("token stored after failure")
			}
		})
	}
}

func TestPairingTVOff(t *testing.T) {
	tv := newFakeTV(t)
	tv.restFail = true
	a, _ := newAdapter(t, tv)
	if _, err := a.StartPairing(context.Background(), host); !errors.Is(err, ErrTVNotResponding) {
		t.Fatalf("err = %v", err)
	}
	if ErrTVNotResponding.Error() != "A TV Samsung não respondeu. Verifique se ela está ligada e tente de novo. Se aparecer 'Permitir' na TV, aceite e tente de novo." {
		t.Errorf("text = %q", ErrTVNotResponding)
	}
}

func TestConnectRevokedTokenDeleted(t *testing.T) {
	tv := newFakeTV(t)
	tv.connect = "unauthorized"
	a, _ := newAdapter(t, tv)
	a.keys.Put(host, "old")
	if _, err := a.Connect(context.Background(), host, newEvents()); !errors.Is(err, domain.ErrNotPaired) {
		t.Fatalf("err = %v", err)
	}
	if a.keys.Get(host) != "" {
		t.Error("revoked token kept")
	}
}

func TestConnectTimeoutKeepsToken(t *testing.T) {
	tv := newFakeTV(t)
	tv.connect = "silent"
	a, _ := newAdapter(t, tv)
	a.connectTimeout = 100 * time.Millisecond
	a.keys.Put(host, "tok1")
	if _, err := a.Connect(context.Background(), host, newEvents()); !errors.Is(err, ErrTVNotResponding) {
		t.Fatalf("err = %v", err)
	}
	if a.keys.Get(host) != "tok1" {
		t.Error("token deleted on timeout")
	}
}

func TestConnectDialFailureKeepsToken(t *testing.T) {
	tv := newFakeTV(t)
	a, _ := newAdapter(t, tv)
	a.Dial = func(context.Context, string, string, bool, tizen.Options) (*tizen.Conn, string, error) {
		return nil, "", errors.New("tizen: não foi possível conectar: connection refused")
	}
	a.keys.Put(host, "tok1")
	if _, err := a.Connect(context.Background(), host, newEvents()); !errors.Is(err, ErrTVNotResponding) {
		t.Fatalf("err = %v", err)
	}
	if a.keys.Get(host) != "tok1" {
		t.Error("token deleted on dial failure")
	}
}

func TestConnectRESTFailure(t *testing.T) {
	tv := newFakeTV(t)
	tv.restFail = true
	a, dials := newAdapter(t, tv)
	a.keys.Put(host, "tok1")
	if _, err := a.Connect(context.Background(), host, newEvents()); !errors.Is(err, ErrTVNotResponding) {
		t.Fatalf("err = %v", err)
	}
	if len(dials) != 0 {
		t.Error("dialed after REST failure")
	}
}

func TestConnectStoresNewToken(t *testing.T) {
	tv := newFakeTV(t)
	tv.issue = "rotated"
	a, _ := newAdapter(t, tv)
	a.keys.Put(host, "tok1")
	s, err := a.Connect(context.Background(), host, newEvents())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if got := a.keys.Get(host); got != "rotated" {
		t.Errorf("stored token = %q", got)
	}
}

func TestConnectInsecureWithoutPairing(t *testing.T) {
	tv := newFakeTV(t)
	tv.tokenAuth = false
	a, dials := newAdapter(t, tv)
	s, err := a.Connect(context.Background(), host, newEvents())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if d := <-dials; d.secure || d.token != "" {
		t.Errorf("dial = %+v", d)
	}
	if a.keys.Get(host) != "" {
		t.Error("token stored for an insecure TV")
	}
}

func TestInfoAndState(t *testing.T) {
	tv := newFakeTV(t)
	s, ev := connect(t, tv)
	info := s.Info()
	want := domain.DeviceInfo{
		Model: "QN55Q60", Vendor: "Samsung", Version: "2018", Platform: domain.PlatformTizen,
		Capabilities: domain.Capabilities{Text: true, AppList: true},
	}
	if info != want {
		t.Errorf("info = %+v", info)
	}
	if st := <-ev.states; st != (domain.State{On: true}) {
		t.Errorf("first state = %+v", st)
	}
	if s.State() != (domain.State{On: true}) {
		t.Errorf("State() = %+v", s.State())
	}
}

func TestSendKeyMap(t *testing.T) {
	tv := newFakeTV(t)
	s, _ := connect(t, tv)
	for _, c := range []struct{ key, want string }{
		{"HOME", "KEY_HOME"}, {"back", "KEY_RETURN"}, {"KEYCODE_DPAD_UP", "KEY_UP"},
		{"DPAD_DOWN", "KEY_DOWN"}, {"DPAD_LEFT", "KEY_LEFT"}, {"DPAD_RIGHT", "KEY_RIGHT"},
		{"DPAD_CENTER", "KEY_ENTER"}, {"ENTER", "KEY_ENTER"},
		{"VOLUME_UP", "KEY_VOLUP"}, {"VOLUME_DOWN", "KEY_VOLDOWN"}, {"MUTE", "KEY_MUTE"},
		{"POWER", "KEY_POWER"}, {"CHANNEL_UP", "KEY_CHUP"}, {"CHANNEL_DOWN", "KEY_CHDOWN"},
		{"MEDIA_STOP", "KEY_STOP"}, {"MEDIA_REWIND", "KEY_REWIND"}, {"MEDIA_FAST_FORWARD", "KEY_FF"},
		{"MENU", "KEY_MENU"}, {"SETTINGS", "KEY_MENU"}, {"GUIDE", "KEY_GUIDE"}, {"TV_INPUT", "KEY_SOURCE"},
		{"MEDIA_PLAY_PAUSE", "KEY_PAUSE"}, {"MEDIA_PLAY_PAUSE", "KEY_PLAY"}, {"MEDIA_PLAY_PAUSE", "KEY_PAUSE"},
	} {
		if err := s.SendKey(c.key); err != nil {
			t.Fatalf("%s: %v", c.key, err)
		}
		p := tv.next(t)
		if p["DataOfCmd"] != c.want || p["Cmd"] != "Click" {
			t.Errorf("%s -> %v", c.key, p)
		}
	}
}

func TestSendKeyUnsupported(t *testing.T) {
	tv := newFakeTV(t)
	s, _ := connect(t, tv)
	for _, k := range []string{"DEL", "MEDIA_NEXT", "MEDIA_PREVIOUS"} {
		err := s.SendKey(k)
		if err == nil || err.Error() != "Tecla não suportada pela Samsung: "+k {
			t.Fatalf("err = %v", err)
		}
	}
	tv.noIO(t)
}

func TestPlayPauseFlipsOnlyOnSuccess(t *testing.T) {
	tv := newFakeTV(t)
	s, _ := connect(t, tv)
	s.conn.Close()
	if err := s.SendKey("MEDIA_PLAY_PAUSE"); err == nil {
		t.Fatal("want error")
	}
	if s.paused {
		t.Error("paused flipped after a failed press")
	}
}

func TestSendTextNeedsKeyboard(t *testing.T) {
	tv := newFakeTV(t)
	s, ev := connect(t, tv)
	err := s.SendText("segredo123")
	if !errors.Is(err, ErrKeyboardClosed) || err.Error() != "Abra um campo de texto na TV antes de enviar." {
		t.Fatalf("err = %v", err)
	}
	tv.noIO(t)

	tv.event("ms.remote.imeStart")
	ev.waitState(t, func(st domain.State) bool { return st.Keyboard })
	if err := s.SendText("segredo123"); err != nil {
		t.Fatal(err)
	}
	p := tv.next(t)
	if p["TypeOfRemote"] != "SendInputString" || p["Cmd"] != base64.StdEncoding.EncodeToString([]byte("segredo123")) {
		t.Errorf("text frame = %v", p)
	}
	if tr := ev.traces(); strings.Contains(tr, "segredo") || tr == "" {
		t.Errorf("traces = %q", tr)
	}

	tv.event("ms.remote.imeEnd")
	ev.waitState(t, func(st domain.State) bool { return !st.Keyboard })
	if err := s.SendText("x"); !errors.Is(err, ErrKeyboardClosed) {
		t.Errorf("err after imeEnd = %v", err)
	}
}

func TestAppsList(t *testing.T) {
	tv := newFakeTV(t)
	s, _ := connect(t, tv)
	apps, err := s.Apps(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 3 || apps[0] != (domain.App{ID: "3201907018807", Name: "Netflix"}) {
		t.Errorf("apps = %+v", apps)
	}
}

func TestLaunchResolves(t *testing.T) {
	tv := newFakeTV(t)
	s, _ := connect(t, tv)
	for _, c := range []struct{ target, id, action string }{
		{"market://launch?id=com.netflix.ninja", "3201907018807", "DEEP_LINK"},
		{"market://launch?id=com.google.android.youtube.tv", "111299001912", "DEEP_LINK"},
		{"youtube", "111299001912", "DEEP_LINK"},
		{"inter", "org.tizen.browser", "NATIVE_LAUNCH"},
		{"org.tizen.browser", "org.tizen.browser", "NATIVE_LAUNCH"},
		{"market://launch?id=org.tizen.browser", "org.tizen.browser", "NATIVE_LAUNCH"},
	} {
		if err := s.LaunchApp(c.target); err != nil {
			t.Fatalf("%s: %v", c.target, err)
		}
		p := tv.next(t)
		if p["event"] == "ed.installedApp.get" {
			p = tv.next(t)
		}
		d, _ := p["data"].(map[string]any)
		if p["event"] != "ed.apps.launch" || d["appId"] != c.id || d["action_type"] != c.action {
			t.Errorf("%s -> %v", c.target, p)
		}
	}
}

func TestLaunchKnownIDWhenTitleDiffers(t *testing.T) {
	tv := newFakeTV(t)
	tv.apps = []map[string]any{{"appId": "3201910019365", "name": "Amazon Prime Video", "app_type": 4}}
	s, _ := connect(t, tv)
	if err := s.LaunchApp("market://launch?id=com.amazon.amazonvideo.livingroom"); err != nil {
		t.Fatal(err)
	}
	tv.next(t)
	d, _ := tv.next(t)["data"].(map[string]any)
	if d["appId"] != "3201910019365" || d["action_type"] != "NATIVE_LAUNCH" {
		t.Errorf("launch = %v", d)
	}
}

func TestLaunchNotInstalled(t *testing.T) {
	tv := newFakeTV(t)
	s, _ := connect(t, tv)
	for _, target := range []string{"market://launch?id=com.disney.disneyplus", "Globoplay", ""} {
		if err := s.LaunchApp(target); !errors.Is(err, ErrAppNotInstalled) {
			t.Fatalf("%q: err = %v", target, err)
		}
	}
	if ErrAppNotInstalled.Error() != "Este app não está instalado na TV Samsung." {
		t.Errorf("text = %q", ErrAppNotInstalled)
	}
}

func TestPollStandbyKeepsSession(t *testing.T) {
	tv := newFakeTV(t)
	a, _ := newAdapter(t, tv)
	a.pollInterval = 20 * time.Millisecond
	a.keys.Put(host, "tok1")
	ev := newEvents()
	s, err := a.Connect(context.Background(), host, ev)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	tv.set(func(tv *fakeTV) { tv.restFail = true })
	time.Sleep(80 * time.Millisecond)
	tv.set(func(tv *fakeTV) { tv.restFail = false; tv.power = "standby" })
	ev.waitState(t, func(st domain.State) bool { return !st.On })
	select {
	case <-s.Done():
		t.Fatal("Done closed on standby")
	default:
	}
	tv.set(func(tv *fakeTV) { tv.power = "on" })
	ev.waitState(t, func(st domain.State) bool { return st.On })
}

func TestSocketDropClosesDone(t *testing.T) {
	tv := newFakeTV(t)
	s, _ := connect(t, tv)
	tv.drop()
	select {
	case <-s.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("Done not closed")
	}
}

func TestCloseIdempotentNoStateAfter(t *testing.T) {
	tv := newFakeTV(t)
	s, ev := connect(t, tv)
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s.Close()
	select {
	case <-s.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("Done not closed")
	}
	for len(ev.states) > 0 {
		<-ev.states
	}
	s.update(func(st *domain.State) { st.On = false })
	if len(ev.states) != 0 {
		t.Error("state emitted after Close")
	}
}

func TestTokensFile(t *testing.T) {
	dir := t.TempDir()
	if New(dir).keys.Path != filepath.Join(dir, "samsung_tokens.json") {
		t.Error("tokens file path")
	}
}

func TestVersionEmptyWithoutModelYear(t *testing.T) {
	tv := newFakeTV(t)
	tv.model = ""
	s, _ := connect(t, tv)
	if v := s.Info().Version; v != "" {
		t.Errorf("version = %q", v)
	}
}

func TestSendTextSentWhenIMENotSynced(t *testing.T) {
	tv := newFakeTV(t)
	tv.imeSynced = false
	s, _ := connect(t, tv)
	if err := s.SendText("ola"); err != nil {
		t.Fatal(err)
	}
	p := tv.next(t)
	if p["TypeOfRemote"] != "SendInputString" {
		t.Errorf("frame = %v", p)
	}
	if s.State().Keyboard {
		t.Error("keyboard state invented")
	}
}

func TestAppsUnavailableIsPortuguese(t *testing.T) {
	tv := newFakeTV(t)
	tv.appsMode = "silent"
	s, _ := connect(t, tv)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err := s.Apps(ctx)
	if !errors.Is(err, ErrAppsUnavailable) || strings.Contains(err.Error(), "deadline") {
		t.Fatalf("err = %v", err)
	}
	if err.Error() != "A TV Samsung não enviou a lista de apps. Tente de novo." {
		t.Errorf("text = %q", err)
	}
}

func TestLaunchFallsBackToKnownIDWhenListFails(t *testing.T) {
	tv := newFakeTV(t)
	tv.appsMode = "silent"
	tv.timeout = 100 * time.Millisecond
	s, _ := connect(t, tv)
	if err := s.LaunchApp("market://launch?id=com.netflix.ninja"); err != nil {
		t.Fatal(err)
	}
	tv.next(t) // installedApp.get
	d, _ := tv.next(t)["data"].(map[string]any)
	if d["appId"] != "3201907018807" || d["action_type"] != "DEEP_LINK" {
		t.Errorf("launch = %v", d)
	}
	for _, target := range []string{"market://launch?id=org.xbmc.kodi", "youtube"} {
		if err := s.LaunchApp(target); !errors.Is(err, ErrAppsUnavailable) {
			t.Errorf("%s: err = %v", target, err)
		}
	}
}

func TestConnectUnauthorizedWithoutTokenAuthIsPromptRejected(t *testing.T) {
	tv := newFakeTV(t)
	tv.tokenAuth = false
	tv.connect = "unauthorized"
	a, _ := newAdapter(t, tv)
	if _, err := a.Connect(context.Background(), host, newEvents()); !errors.Is(err, ErrPromptRejected) {
		t.Fatalf("err = %v", err)
	}
}

func TestPairingEmptyTokenIsPairingLost(t *testing.T) {
	tv := newFakeTV(t)
	a, _ := newAdapter(t, tv)
	a.Dial = func(context.Context, string, string, bool, tizen.Options) (*tizen.Conn, string, error) {
		return nil, "", nil
	}
	p, err := a.StartPairing(context.Background(), host)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	if err := p.Finish(""); !errors.Is(err, ErrPairingLost) {
		t.Fatalf("err = %v", err)
	}
	if a.keys.Get(host) != "" {
		t.Error("token stored")
	}
}
