package webos

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"optic/internal/domain"
	"optic/pkg/webos"

	"github.com/gorilla/websocket"
)

const host = "10.0.0.5"

var upgrader = websocket.Upgrader{}

// fakeTV answers register, requests and subscriptions on "/" and records
// pointer buttons on "/pointer".
type fakeTV struct {
	t   *testing.T
	url string

	register string         // "accept" (default), "prompt", "reject" or "silent"
	replies  map[string]any // request uri -> payload (default {"returnValue":true})
	subs     map[string]any // subscription uri -> first payload

	regs    chan map[string]any // register payloads
	reqs    chan map[string]any // requests
	buttons chan string
}

func newFakeTV(t *testing.T) *fakeTV {
	tv := &fakeTV{
		t:       t,
		replies: map[string]any{},
		subs:    map[string]any{},
		regs:    make(chan map[string]any, 8),
		reqs:    make(chan map[string]any, 64),
		buttons: make(chan string, 64),
	}
	tv.replies["ssap://system/getSystemInfo"] = map[string]any{"returnValue": true, "modelName": "OLED55C1"}
	tv.replies["ssap://com.webos.service.update/getCurrentSWInformation"] = map[string]any{
		"returnValue": true, "product_name": "webOSTV 6.0", "major_ver": "03", "minor_ver": "20.11",
	}
	tv.replies["ssap://com.webos.service.networkinput/getPointerInputSocket"] = map[string]any{
		"returnValue": true, "socketPath": "ws://elsewhere:1/pointer",
	}
	tv.replies["ssap://com.webos.applicationManager/listLaunchPoints"] = map[string]any{
		"returnValue": true,
		"launchPoints": []map[string]any{
			{"id": "netflix", "title": "Netflix"},
			{"id": "youtube.leanback.v4", "title": "YouTube"},
			{"id": "com.webos.app.hdmi1", "title": "HDMI 1"},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer ws.Close()
		if r.URL.Path == "/pointer" {
			tv.pointer(ws)
			return
		}
		tv.serve(ws)
	}))
	t.Cleanup(srv.Close)
	tv.url = "ws" + strings.TrimPrefix(srv.URL, "http")
	return tv
}

func (tv *fakeTV) pointer(ws *websocket.Conn) {
	for {
		_, b, err := ws.ReadMessage()
		if err != nil {
			return
		}
		for _, line := range strings.Split(string(b), "\n") {
			if name, ok := strings.CutPrefix(line, "name:"); ok {
				tv.buttons <- name
			}
		}
	}
}

func (tv *fakeTV) serve(ws *websocket.Conn) {
	for {
		var m map[string]any
		if err := ws.ReadJSON(&m); err != nil {
			return
		}
		id := m["id"]
		switch m["type"] {
		case "register":
			p, _ := m["payload"].(map[string]any)
			tv.regs <- p
			switch tv.register {
			case "silent":
			case "reject":
				ws.WriteJSON(map[string]any{"type": "error", "id": id, "error": "403 rejected"})
			case "prompt":
				ws.WriteJSON(map[string]any{"type": "response", "id": id, "payload": map[string]any{"pairingType": "PROMPT"}})
				ws.WriteJSON(map[string]any{"type": "registered", "id": id, "payload": map[string]any{"client-key": "new"}})
			default:
				key, _ := p["client-key"].(string)
				if key == "" {
					key = "new"
				}
				ws.WriteJSON(map[string]any{"type": "registered", "id": id, "payload": map[string]any{"client-key": key}})
			}
		case "request":
			tv.reqs <- m
			reply, ok := tv.replies[m["uri"].(string)]
			if !ok {
				reply = map[string]any{"returnValue": true}
			}
			ws.WriteJSON(map[string]any{"type": "response", "id": id, "payload": reply})
		case "subscribe":
			if p, ok := tv.subs[m["uri"].(string)]; ok {
				ws.WriteJSON(map[string]any{"type": "response", "id": id, "payload": p})
			}
		}
	}
}

// nextReq returns the next request whose uri is not an info/pointer call.
func (tv *fakeTV) nextReq(t *testing.T) map[string]any {
	t.Helper()
	for {
		select {
		case m := <-tv.reqs:
			switch m["uri"] {
			case "ssap://system/getSystemInfo",
				"ssap://com.webos.service.update/getCurrentSWInformation",
				"ssap://com.webos.service.networkinput/getPointerInputSocket":
				continue
			}
			return m
		case <-time.After(2 * time.Second):
			t.Fatal("no request")
			return nil
		}
	}
}

func (tv *fakeTV) noIO(t *testing.T) {
	t.Helper()
	select {
	case b := <-tv.buttons:
		t.Errorf("unexpected button %s", b)
	case m := <-tv.reqs:
		if u := m["uri"]; u != "ssap://system/getSystemInfo" &&
			u != "ssap://com.webos.service.update/getCurrentSWInformation" &&
			u != "ssap://com.webos.service.networkinput/getPointerInputSocket" {
			t.Errorf("unexpected request %v", u)
		}
	case <-time.After(100 * time.Millisecond):
	}
}

func (tv *fakeTV) button(t *testing.T) string {
	t.Helper()
	select {
	case b := <-tv.buttons:
		return b
	case <-time.After(2 * time.Second):
		t.Fatal("no button")
		return ""
	}
}

type events struct {
	states chan domain.State
	mu     sync.Mutex
	lines  []string
}

func newEvents() *events { return &events{states: make(chan domain.State, 32)} }

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

func newAdapter(t *testing.T, tv *fakeTV) (*Adapter, *atomic.Int32) {
	a := New(t.TempDir())
	var dials atomic.Int32
	a.Dial = func(ctx context.Context, h string, opt webos.Options) (*webos.Conn, error) {
		dials.Add(1)
		opt.URLs = []string{tv.url}
		return webos.Dial(ctx, h, opt)
	}
	return a, &dials
}

// connect stores a key and opens a session.
func connect(t *testing.T, tv *fakeTV) (*session, *events) {
	t.Helper()
	a, _ := newAdapter(t, tv)
	if err := a.keys.Put(host, "k1"); err != nil {
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

func payload(m map[string]any) map[string]any {
	p, _ := m["payload"].(map[string]any)
	return p
}

func TestConnectWithoutKeyNotPaired(t *testing.T) {
	tv := newFakeTV(t)
	a, dials := newAdapter(t, tv)
	_, err := a.Connect(context.Background(), host, newEvents())
	if !errors.Is(err, domain.ErrNotPaired) {
		t.Fatalf("err = %v", err)
	}
	if dials.Load() != 0 {
		t.Error("dialed without a key")
	}
}

func TestPairingPromptStoresKey(t *testing.T) {
	tv := newFakeTV(t)
	tv.register = "prompt"
	a, _ := newAdapter(t, tv)
	p, err := a.StartPairing(context.Background(), host)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	if p.Mode() != domain.PairingPrompt || p.ServerName() != "TV LG" {
		t.Errorf("mode=%q name=%q", p.Mode(), p.ServerName())
	}
	if err := p.Finish("ignored"); err != nil {
		t.Fatal(err)
	}
	if got := a.keys.Get(host); got != "new" {
		t.Fatalf("stored key = %q", got)
	}
	if _, ok := (<-tv.regs)["client-key"]; ok {
		t.Error("pairing must register without a key")
	}

	tv.register = "accept"
	s, err := a.Connect(context.Background(), host, newEvents())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if k := (<-tv.regs)["client-key"]; k != "new" {
		t.Errorf("connect registered with %v", k)
	}
}

func TestPairingRejected(t *testing.T) {
	tv := newFakeTV(t)
	tv.register = "reject"
	a, _ := newAdapter(t, tv)
	p, err := a.StartPairing(context.Background(), host)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	if err := p.Finish(""); !errors.Is(err, ErrPromptRejected) {
		t.Fatalf("err = %v", err)
	}
	if a.keys.Get(host) != "" {
		t.Error("key stored after rejection")
	}
}

func TestPairingTimeout(t *testing.T) {
	tv := newFakeTV(t)
	tv.register = "silent"
	a, _ := newAdapter(t, tv)
	a.promptTimeout = 50 * time.Millisecond
	p, err := a.StartPairing(context.Background(), host)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	if err := p.Finish(""); !errors.Is(err, ErrPromptTimeout) {
		t.Fatalf("err = %v", err)
	}
}

func TestConnectRevokedKeyDeletesIt(t *testing.T) {
	tv := newFakeTV(t)
	tv.register = "reject"
	a, _ := newAdapter(t, tv)
	a.keys.Put(host, "old")
	_, err := a.Connect(context.Background(), host, newEvents())
	if !errors.Is(err, domain.ErrNotPaired) {
		t.Fatalf("err = %v", err)
	}
	if a.keys.Get(host) != "" {
		t.Error("revoked key kept")
	}
}

func TestInfoAndState(t *testing.T) {
	tv := newFakeTV(t)
	tv.subs["ssap://audio/getVolume"] = map[string]any{
		"returnValue": true, "subscribed": true,
		"volumeStatus": map[string]any{"volume": 12, "muteStatus": false, "maxVolume": 100},
	}
	tv.subs["ssap://com.webos.applicationManager/getForegroundAppInfo"] = map[string]any{
		"returnValue": true, "appId": "netflix",
	}
	s, ev := connect(t, tv)
	info := s.Info()
	if info.Platform != domain.PlatformWebOS || info.Vendor != "LG" || info.Model != "OLED55C1" {
		t.Errorf("info = %+v", info)
	}
	if info.Version != "webOSTV 6.0 03.20.11" {
		t.Errorf("version = %q", info.Version)
	}
	want := domain.Capabilities{Text: true, AppList: true, AbsoluteVolume: true, Inputs: true, Toast: true}
	if info.Capabilities != want {
		t.Errorf("caps = %+v", info.Capabilities)
	}
	wantState := domain.State{On: true, Volume: 12, VolumeMax: 100, App: "netflix"}
	deadline := time.After(2 * time.Second)
	for {
		select {
		case st := <-ev.states:
			if st == wantState {
				if s.State() != wantState {
					t.Errorf("State() = %+v", s.State())
				}
				return
			}
		case <-deadline:
			t.Fatalf("state never reached %+v (now %+v)", wantState, s.State())
		}
	}
}

func TestVolumeLegacyPayload(t *testing.T) {
	tv := newFakeTV(t)
	tv.subs["ssap://audio/getVolume"] = map[string]any{"returnValue": true, "volume": 7, "muted": true}
	s, ev := connect(t, tv)
	deadline := time.After(2 * time.Second)
	for {
		select {
		case st := <-ev.states:
			if st.Volume == 7 {
				if !st.Muted || st.VolumeMax != 100 {
					t.Errorf("state = %+v", st)
				}
				return
			}
		case <-deadline:
			t.Fatalf("state = %+v", s.State())
		}
	}
}

func TestSendKeyMapsToPointer(t *testing.T) {
	tv := newFakeTV(t)
	s, _ := connect(t, tv)
	for _, c := range []struct{ key, button string }{
		{"DPAD_CENTER", "ENTER"},
		{"DPAD_UP", "UP"},
		{"VOLUME_UP", "VOLUMEUP"},
		{"MEDIA_PLAY_PAUSE", "PAUSE"},
		{"MEDIA_PLAY_PAUSE", "PLAY"},
		{"SETTINGS", "MENU"},
	} {
		if err := s.SendKey(c.key); err != nil {
			t.Fatalf("%s: %v", c.key, err)
		}
		if got := tv.button(t); got != c.button {
			t.Errorf("%s -> %s, want %s", c.key, got, c.button)
		}
	}
	if err := s.SendKey("POWER"); err != nil {
		t.Fatal(err)
	}
	if u := tv.nextReq(t)["uri"]; u != "ssap://system/turnOff" {
		t.Errorf("POWER -> %v", u)
	}
}

func TestSendKeyUnsupported(t *testing.T) {
	tv := newFakeTV(t)
	s, _ := connect(t, tv)
	err := s.SendKey("MEDIA_NEXT")
	if err == nil || err.Error() != "Tecla não suportada pela LG: MEDIA_NEXT" {
		t.Fatalf("err = %v", err)
	}
	tv.noIO(t)
}

func TestLaunchResolves(t *testing.T) {
	tv := newFakeTV(t)
	s, _ := connect(t, tv)
	for _, c := range []struct{ target, id string }{
		{"market://launch?id=com.netflix.ninja", "netflix"},
		{"youtube", "youtube.leanback.v4"},
		{"YouTube", "youtube.leanback.v4"},
		{"net", "netflix"},
	} {
		if err := s.LaunchApp(c.target); err != nil {
			t.Fatalf("%s: %v", c.target, err)
		}
		m := tv.nextReq(t)
		if m["uri"] == "ssap://com.webos.applicationManager/listLaunchPoints" {
			m = tv.nextReq(t)
		}
		if m["uri"] != "ssap://system.launcher/launch" || payload(m)["id"] != c.id {
			t.Errorf("%s -> %v %v", c.target, m["uri"], payload(m))
		}
	}
	if err := s.LaunchApp("https://example.com/x"); err != nil {
		t.Fatal(err)
	}
	m := tv.nextReq(t)
	if m["uri"] != "ssap://system.launcher/open" || payload(m)["target"] != "https://example.com/x" {
		t.Errorf("link -> %v %v", m["uri"], payload(m))
	}
}

func TestLaunchNotInstalled(t *testing.T) {
	tv := newFakeTV(t)
	s, _ := connect(t, tv)
	for _, target := range []string{"market://launch?id=com.disney.disneyplus", "Globoplay", ""} {
		err := s.LaunchApp(target)
		if !errors.Is(err, ErrAppNotInstalled) {
			t.Fatalf("%q: err = %v", target, err)
		}
	}
	if ErrAppNotInstalled.Error() != "Este app não está instalado na TV LG." {
		t.Errorf("text = %q", ErrAppNotInstalled)
	}
}

func TestAppsList(t *testing.T) {
	tv := newFakeTV(t)
	s, _ := connect(t, tv)
	apps, err := s.Apps(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 3 || apps[0] != (domain.App{ID: "netflix", Name: "Netflix"}) {
		t.Errorf("apps = %+v", apps)
	}
}

func TestSetVolumeInputsToast(t *testing.T) {
	tv := newFakeTV(t)
	tv.replies["ssap://tv/getExternalInputList"] = map[string]any{
		"returnValue": true,
		"devices": []map[string]any{
			{"id": "HDMI_1", "label": "HDMI 1"},
			{"id": "HDMI_2", "label": "PlayStation"},
		},
	}
	s, _ := connect(t, tv)
	ctx := context.Background()

	if err := s.SetVolume(ctx, 15); err != nil {
		t.Fatal(err)
	}
	m := tv.nextReq(t)
	if m["uri"] != "ssap://audio/setVolume" || payload(m)["volume"] != float64(15) {
		t.Errorf("setVolume -> %v %v", m["uri"], payload(m))
	}

	ins, err := s.Inputs(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(ins) != 2 || ins[1] != (domain.Input{ID: "HDMI_2", Name: "PlayStation"}) {
		t.Errorf("inputs = %+v", ins)
	}
	tv.nextReq(t)

	if err := s.SwitchInput(ctx, "HDMI_2"); err != nil {
		t.Fatal(err)
	}
	m = tv.nextReq(t)
	if m["uri"] != "ssap://tv/switchInput" || payload(m)["inputId"] != "HDMI_2" {
		t.Errorf("switchInput -> %v %v", m["uri"], payload(m))
	}

	if err := s.Toast(ctx, "Olá"); err != nil {
		t.Fatal(err)
	}
	m = tv.nextReq(t)
	if m["uri"] != "ssap://system.notifications/createToast" || payload(m)["message"] != "Olá" {
		t.Errorf("toast -> %v %v", m["uri"], payload(m))
	}
}

func TestTextNotTraced(t *testing.T) {
	tv := newFakeTV(t)
	s, ev := connect(t, tv)
	if err := s.SendText("segredo123"); err != nil {
		t.Fatal(err)
	}
	m := tv.nextReq(t)
	if m["uri"] != "ssap://com.webos.service.ime/insertText" ||
		payload(m)["text"] != "segredo123" || payload(m)["replace"] != false {
		t.Errorf("insertText -> %v %v", m["uri"], payload(m))
	}

	tv.replies["ssap://com.webos.service.ime/insertText"] = map[string]any{"returnValue": false, "errorText": "no keyboard"}
	err := s.SendText("segredo456")
	if err == nil || strings.Contains(err.Error(), "segredo") {
		t.Errorf("err = %v", err)
	}
	if tr := ev.traces(); strings.Contains(tr, "segredo") || tr == "" {
		t.Errorf("traces = %q", tr)
	}
}

func TestCloseIdempotentAndDone(t *testing.T) {
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
	// No state after Close.
	for len(ev.states) > 0 {
		<-ev.states
	}
	s.update(func(st *domain.State) { st.Volume = 99 })
	if len(ev.states) != 0 {
		t.Error("state emitted after Close")
	}
}

func TestKeyStoreAtomicAndCorrupt(t *testing.T) {
	dir := t.TempDir()
	ks := keyStore{path: filepath.Join(dir, "webos_keys.json")}
	if ks.Get(host) != "" {
		t.Error("missing file must read empty")
	}
	if err := ks.Put(host, "a"); err != nil {
		t.Fatal(err)
	}
	if err := ks.Put("10.0.0.6", "b"); err != nil {
		t.Fatal(err)
	}
	if ks.Get(host) != "a" || ks.Get("10.0.0.6") != "b" {
		t.Error("keys not kept")
	}
	if _, err := os.Stat(ks.path + ".tmp"); !os.IsNotExist(err) {
		t.Error("temp file left behind")
	}
	var m map[string]string
	b, _ := os.ReadFile(ks.path)
	if json.Unmarshal(b, &m) != nil || m[host] != "a" {
		t.Errorf("file = %s", b)
	}
	if err := ks.Delete(host); err != nil {
		t.Fatal(err)
	}
	if ks.Get(host) != "" || ks.Get("10.0.0.6") != "b" {
		t.Error("delete wrong")
	}

	os.WriteFile(ks.path, []byte("{not json"), 0o600)
	if ks.Get("10.0.0.6") != "" {
		t.Error("corrupt file must read empty")
	}
	if err := ks.Put(host, "c"); err != nil || ks.Get(host) != "c" {
		t.Errorf("put over corrupt: %v", err)
	}

	if New(dir).keys.path != filepath.Join(dir, "webos_keys.json") {
		t.Error("keys file path")
	}
}
