package tizen

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{}

type fake struct {
	url   string
	query chan string
}

// fakeTV starts a websocket server running h for each connection.
func fakeTV(t *testing.T, h func(ws *websocket.Conn)) fake {
	t.Helper()
	f := fake{query: make(chan string, 4)}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case f.query <- r.URL.Path + "?" + r.URL.RawQuery:
		default:
		}
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer ws.Close()
		h(ws)
	}))
	t.Cleanup(srv.Close)
	f.url = "ws" + strings.TrimPrefix(srv.URL, "http")
	return f
}

func connectEvent(ws *websocket.Conn, token string) {
	data := map[string]any{"clients": []any{}}
	if token != "" {
		data["token"] = token
	}
	ws.WriteJSON(map[string]any{"event": "ms.channel.connect", "data": data})
}

func dial(t *testing.T, f fake, token string, tr func(string, string)) *Conn {
	t.Helper()
	c, _, err := Dial(context.Background(), "ignored", token, true, Options{URL: f.url, Trace: tr})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	return c
}

func readMsg(t *testing.T, ws *websocket.Conn) map[string]any {
	t.Helper()
	var m map[string]any
	if err := ws.ReadJSON(&m); err != nil {
		t.Errorf("fake tv read: %v", err)
		return nil
	}
	return m
}

type traceLog struct {
	mu    sync.Mutex
	lines []string
}

func (l *traceLog) fn(dir, line string) {
	l.mu.Lock()
	l.lines = append(l.lines, dir+" "+line)
	l.mu.Unlock()
}

func (l *traceLog) all() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return strings.Join(l.lines, "\n")
}

func TestDialSecureSendsNameAndToken(t *testing.T) {
	f := fakeTV(t, func(ws *websocket.Conn) {
		connectEvent(ws, "tok123")
		ws.ReadMessage()
	})
	c, newTok, err := Dial(context.Background(), "ignored", "old-token", true, Options{URL: f.url})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if newTok != "tok123" {
		t.Errorf("new token = %q", newTok)
	}
	q := <-f.query
	if !strings.HasPrefix(q, "/api/v2/channels/samsung.remote.control?") {
		t.Errorf("path = %q", q)
	}
	// "=" in the base64 value is escaped by url.Values; the TV decodes it.
	if !strings.Contains(q, "name=T1BUSUM%3D") || !strings.Contains(q, "token=old-token") {
		t.Errorf("query = %q", q)
	}
}

func TestDialWithoutTokenOmitsParam(t *testing.T) {
	f := fakeTV(t, func(ws *websocket.Conn) { connectEvent(ws, ""); ws.ReadMessage() })
	c, tok, err := Dial(context.Background(), "ignored", "", true, Options{URL: f.url})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if tok != "" {
		t.Errorf("token = %q", tok)
	}
	if q := <-f.query; strings.Contains(q, "token") {
		t.Errorf("query = %q", q)
	}
}

func TestDialInsecureNeverSendsToken(t *testing.T) {
	f := fakeTV(t, func(ws *websocket.Conn) { connectEvent(ws, ""); ws.ReadMessage() })
	c, _, err := Dial(context.Background(), "ignored", "secret", false, Options{URL: f.url})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if q := <-f.query; strings.Contains(q, "token") {
		t.Errorf("query = %q", q)
	}
}

func TestDialTLSSelfSigned(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer ws.Close()
		connectEvent(ws, "t")
		ws.ReadMessage()
	}))
	defer srv.Close()
	c, tok, err := Dial(context.Background(), "ignored", "", true,
		Options{URL: "wss" + strings.TrimPrefix(srv.URL, "https")})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if tok != "t" {
		t.Errorf("token = %q", tok)
	}
}

func TestDialUnauthorized(t *testing.T) {
	f := fakeTV(t, func(ws *websocket.Conn) {
		ws.WriteJSON(map[string]any{"event": "ms.channel.unauthorized"})
		ws.ReadMessage()
	})
	_, _, err := Dial(context.Background(), "ignored", "", true, Options{URL: f.url})
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("err = %v", err)
	}
}

func TestDialTVTimeoutEvent(t *testing.T) {
	f := fakeTV(t, func(ws *websocket.Conn) {
		ws.WriteJSON(map[string]any{"event": "ms.channel.timeOut"})
		ws.ReadMessage()
	})
	_, _, err := Dial(context.Background(), "ignored", "", true, Options{URL: f.url})
	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("err = %v", err)
	}
}

func TestDialSilentTVTimesOut(t *testing.T) {
	f := fakeTV(t, func(ws *websocket.Conn) { ws.ReadMessage() })
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, _, err := Dial(ctx, "ignored", "", true, Options{URL: f.url})
	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("err = %v", err)
	}
	if time.Since(start) > 2*time.Second {
		t.Error("took too long")
	}
}

func TestDialErrorsNeverContainToken(t *testing.T) {
	// Nothing listens here: the dial fails with a transport error.
	_, _, err := Dial(context.Background(), "127.0.0.1", "supersecret", true,
		Options{URL: "ws://127.0.0.1:1", Timeout: time.Second})
	if err == nil {
		t.Fatal("expected error")
	}
	for _, bad := range []string{"token=", "supersecret", "T1BUSUM", "ws://"} {
		if strings.Contains(err.Error(), bad) {
			t.Errorf("error leaks %q: %v", bad, err)
		}
	}

	// A server that is not a websocket: bad handshake error.
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()
	_, _, err = Dial(context.Background(), "x", "supersecret", true,
		Options{URL: "ws" + strings.TrimPrefix(srv.URL, "http")})
	if err == nil {
		t.Fatal("expected error")
	}
	for _, bad := range []string{"token=", "supersecret", "T1BUSUM"} {
		if strings.Contains(err.Error(), bad) {
			t.Errorf("error leaks %q: %v", bad, err)
		}
	}
}

func TestDialDroppedBeforeHandshake(t *testing.T) {
	f := fakeTV(t, func(ws *websocket.Conn) {}) // closes at once
	_, _, err := Dial(context.Background(), "ignored", "tk", true, Options{URL: f.url})
	if err == nil || strings.Contains(err.Error(), "token=") {
		t.Fatalf("err = %v", err)
	}
}

func TestKeyFrame(t *testing.T) {
	got := make(chan map[string]any, 1)
	f := fakeTV(t, func(ws *websocket.Conn) {
		connectEvent(ws, "")
		got <- readMsg(t, ws)
		ws.ReadMessage()
	})
	tl := &traceLog{}
	c := dial(t, f, "", tl.fn)
	if err := c.Key("KEY_HOME"); err != nil {
		t.Fatal(err)
	}
	want := map[string]any{
		"method": "ms.remote.control",
		"params": map[string]any{
			"Cmd": "Click", "DataOfCmd": "KEY_HOME", "Option": "false", "TypeOfRemote": "SendRemoteKey",
		},
	}
	if m := <-got; !reflect.DeepEqual(m, want) {
		t.Errorf("frame = %v", m)
	}
	if !strings.Contains(tl.all(), "→ key KEY_HOME") {
		t.Errorf("trace = %q", tl.all())
	}
}

func TestTextFramesAndTraceOmitsText(t *testing.T) {
	got := make(chan map[string]any, 2)
	f := fakeTV(t, func(ws *websocket.Conn) {
		connectEvent(ws, "")
		got <- readMsg(t, ws)
		got <- readMsg(t, ws)
		ws.ReadMessage()
	})
	tl := &traceLog{}
	c := dial(t, f, "", tl.fn)
	if err := c.Text("olá mundo"); err != nil {
		t.Fatal(err)
	}
	first, second := <-got, <-got
	p := first["params"].(map[string]any)
	if p["TypeOfRemote"] != "SendInputString" || p["DataOfCmd"] != "base64" || p["Cmd"] != "b2zDoSBtdW5kbw==" {
		t.Errorf("first = %v", first)
	}
	want := map[string]any{"method": "ms.remote.control", "params": map[string]any{"TypeOfRemote": "SendInputEnd"}}
	if !reflect.DeepEqual(second, want) {
		t.Errorf("second = %v", second)
	}
	tr := tl.all()
	if strings.Contains(tr, "olá") || strings.Contains(tr, "b2zDoSB") {
		t.Errorf("trace leaks text: %q", tr)
	}
	if !strings.Contains(tr, "→ SendInputString [texto omitido]") {
		t.Errorf("trace = %q", tr)
	}
}

func TestApps(t *testing.T) {
	req := make(chan map[string]any, 1)
	f := fakeTV(t, func(ws *websocket.Conn) {
		connectEvent(ws, "")
		req <- readMsg(t, ws)
		ws.WriteJSON(map[string]any{"event": "ed.installedApp.get", "data": map[string]any{"data": []any{
			map[string]any{"appId": "111299001912", "name": "YouTube", "app_type": 2},
			map[string]any{"appId": "3201907018807", "name": "Netflix", "app_type": 4},
		}}})
		ws.ReadMessage()
	})
	tl := &traceLog{}
	c := dial(t, f, "", tl.fn)
	apps, err := c.Apps(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := []App{{"111299001912", "YouTube", 2}, {"3201907018807", "Netflix", 4}}
	if !reflect.DeepEqual(apps, want) {
		t.Errorf("apps = %v", apps)
	}
	wantReq := map[string]any{
		"method": "ms.channel.emit",
		"params": map[string]any{"event": "ed.installedApp.get", "to": "host"},
	}
	if m := <-req; !reflect.DeepEqual(m, wantReq) {
		t.Errorf("request = %v", m)
	}
	tr := tl.all()
	if !strings.Contains(tr, "→ emit ed.installedApp.get") || !strings.Contains(tr, "← event ed.installedApp.get") {
		t.Errorf("trace = %q", tr)
	}
}

func TestAppsFailsWhenSocketDrops(t *testing.T) {
	f := fakeTV(t, func(ws *websocket.Conn) {
		connectEvent(ws, "")
		ws.ReadMessage() // the emit, then close without answering
	})
	c := dial(t, f, "", nil)
	_, err := c.Apps(context.Background())
	if err == nil || errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v", err)
	}
}

func TestAppsDefaultTimeout(t *testing.T) {
	f := fakeTV(t, func(ws *websocket.Conn) {
		connectEvent(ws, "")
		for {
			if _, _, err := ws.ReadMessage(); err != nil {
				return
			}
		}
	})
	c, _, err := Dial(context.Background(), "ignored", "", true, Options{URL: f.url, Timeout: 100 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	_, err = c.Apps(context.Background())
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v", err)
	}
}

func TestLaunchActionType(t *testing.T) {
	got := make(chan map[string]any, 2)
	f := fakeTV(t, func(ws *websocket.Conn) {
		connectEvent(ws, "")
		got <- readMsg(t, ws)
		got <- readMsg(t, ws)
		ws.ReadMessage()
	})
	c := dial(t, f, "", nil)
	if err := c.Launch("111299001912", 2); err != nil {
		t.Fatal(err)
	}
	if err := c.Launch("3201907018807", 4); err != nil {
		t.Fatal(err)
	}
	for i, tc := range []struct{ id, action string }{{"111299001912", "DEEP_LINK"}, {"3201907018807", "NATIVE_LAUNCH"}} {
		want := map[string]any{
			"method": "ms.channel.emit",
			"params": map[string]any{
				"event": "ed.apps.launch", "to": "host",
				"data": map[string]any{"appId": tc.id, "action_type": tc.action},
			},
		}
		if m := <-got; !reflect.DeepEqual(m, want) {
			t.Errorf("launch %d = %v", i, m)
		}
	}
}

func TestIMEEvents(t *testing.T) {
	f := fakeTV(t, func(ws *websocket.Conn) {
		connectEvent(ws, "")
		ws.ReadMessage() // wait for the client's ping frame
		ws.WriteJSON(map[string]any{"event": "ms.remote.imeStart", "data": "x"})
		ws.WriteJSON(map[string]any{"event": "ms.remote.imeEnd"})
		ws.ReadMessage()
	})
	c := dial(t, f, "", nil)
	ch := make(chan bool, 4)
	c.OnIME(func(open bool) { ch <- open })
	if err := c.Key("KEY_ENTER"); err != nil {
		t.Fatal(err)
	}
	for _, want := range []bool{true, false} {
		select {
		case got := <-ch:
			if got != want {
				t.Errorf("ime = %v, want %v", got, want)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("no ime event")
		}
	}
}

func TestDropClosesDone(t *testing.T) {
	f := fakeTV(t, func(ws *websocket.Conn) { connectEvent(ws, "") })
	c := dial(t, f, "", nil)
	select {
	case <-c.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("Done not closed")
	}
	c.Close()
	c.Close()
}

func TestCloseIdempotent(t *testing.T) {
	f := fakeTV(t, func(ws *websocket.Conn) { connectEvent(ws, ""); ws.ReadMessage() })
	c := dial(t, f, "", nil)
	c.Close()
	c.Close()
	select {
	case <-c.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("Done not closed after Close")
	}
}

func TestWriteErrorsNeverContainToken(t *testing.T) {
	f := fakeTV(t, func(ws *websocket.Conn) { connectEvent(ws, "") })
	c := dial(t, f, "supersecret", nil)
	<-c.Done()
	c.Close()
	var err error
	for i := 0; i < 5 && err == nil; i++ {
		err = c.Key("KEY_UP")
	}
	if err == nil {
		t.Fatal("expected write error")
	}
	if strings.Contains(err.Error(), "token=") || strings.Contains(err.Error(), "supersecret") {
		t.Errorf("error leaks token: %v", err)
	}
}

func infoServer(t *testing.T, h http.HandlerFunc) string {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return srv.URL
}

func TestGetInfo(t *testing.T) {
	var path string
	base := infoServer(t, func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		json.NewEncoder(w).Encode(map[string]any{"device": map[string]any{
			"name": "Sala", "modelName": "UN55", "wifiMac": "AA:BB", "PowerState": "standby",
			"OS": "Tizen", "TokenAuthSupport": "true", "model": "18_KANTM2_QTV", "ImeSyncedSupport": "true",
		}})
	})
	got, err := GetInfo(context.Background(), &http.Client{Timeout: time.Second}, base)
	if err != nil {
		t.Fatal(err)
	}
	want := Info{Name: "Sala", Model: "UN55", WifiMAC: "AA:BB", PowerState: "standby", OS: "Tizen", TokenAuth: true, ModelCode: "18_KANTM2_QTV", Year: "2018", ImeSynced: true}
	if got != want {
		t.Errorf("info = %+v", got)
	}
	if path != "/api/v2/" {
		t.Errorf("path = %q", path)
	}
}

func TestGetInfoDefaults(t *testing.T) {
	base := infoServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"device":{"name":"Velha"}}`))
	})
	got, err := GetInfo(context.Background(), &http.Client{}, base)
	if err != nil {
		t.Fatal(err)
	}
	if got.PowerState != "on" || got.TokenAuth || got.ImeSynced || got.Year != "" {
		t.Errorf("info = %+v", got)
	}
}

func TestGetInfoRejectsRedirect(t *testing.T) {
	hit := false
	other := infoServer(t, func(w http.ResponseWriter, r *http.Request) { hit = true })
	base := infoServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, other, http.StatusFound)
	})
	// The caller's client follows redirects by default; GetInfo must not.
	if _, err := GetInfo(context.Background(), &http.Client{}, base); err == nil {
		t.Fatal("expected error")
	}
	if hit {
		t.Error("redirect was followed")
	}
}

func TestGetInfoCapsBody(t *testing.T) {
	base := infoServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"device":{"name":"`))
		w.Write([]byte(strings.Repeat("a", 2<<20)))
		w.Write([]byte(`"}}`))
	})
	if _, err := GetInfo(context.Background(), &http.Client{}, base); err == nil {
		t.Fatal("expected error for oversized body")
	}
}

func TestGetInfoBadStatusAndJSON(t *testing.T) {
	bad := infoServer(t, func(w http.ResponseWriter, r *http.Request) { http.Error(w, "x", 500) })
	if _, err := GetInfo(context.Background(), &http.Client{}, bad); err == nil {
		t.Error("expected status error")
	}
	junk := infoServer(t, func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("nope")) })
	if _, err := GetInfo(context.Background(), &http.Client{}, junk); err == nil {
		t.Error("expected json error")
	}
}

func TestDialUnauthorizedThenCloseIsDeterministic(t *testing.T) {
	f := fakeTV(t, func(ws *websocket.Conn) {
		ws.WriteJSON(map[string]any{"event": "ms.channel.unauthorized"})
	})
	for i := 0; i < 50; i++ {
		_, _, err := Dial(context.Background(), "ignored", "", true, Options{URL: f.url})
		if !errors.Is(err, ErrUnauthorized) {
			t.Fatalf("iteration %d: err = %v", i, err)
		}
	}
}

func TestDialConnectThenCloseIsDeterministic(t *testing.T) {
	f := fakeTV(t, func(ws *websocket.Conn) { connectEvent(ws, "tk") })
	for i := 0; i < 50; i++ {
		c, tok, err := Dial(context.Background(), "ignored", "", true, Options{URL: f.url})
		if err != nil || tok != "tk" {
			t.Fatalf("iteration %d: tok=%q err=%v", i, tok, err)
		}
		c.Close()
	}
}

func TestDialDefaultDeadlineWithoutCtxDeadline(t *testing.T) {
	f := fakeTV(t, func(ws *websocket.Conn) { ws.ReadMessage() })
	start := time.Now()
	_, _, err := Dial(context.Background(), "ignored", "", true, Options{URL: f.url, Timeout: 100 * time.Millisecond})
	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("err = %v", err)
	}
	if time.Since(start) > 2*time.Second {
		t.Error("took too long")
	}
}

func TestDialHandshakeStallIsErrTimeout(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			defer c.Close() // never answers the upgrade
		}
	}()
	_, _, err = Dial(context.Background(), "ignored", "", true,
		Options{URL: "ws://" + ln.Addr().String(), Timeout: 100 * time.Millisecond})
	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("err = %v", err)
	}
}

func TestBuildURLIPv6(t *testing.T) {
	if u := buildURL("fe80::1", "t", true, ""); !strings.HasPrefix(u, "wss://[fe80::1]:8002/") {
		t.Errorf("url = %q", u)
	}
	if u := buildURL("fe80::1", "", false, ""); !strings.HasPrefix(u, "ws://[fe80::1]:8001/") {
		t.Errorf("url = %q", u)
	}
}

func TestTraceTruncatesEventName(t *testing.T) {
	long := strings.Repeat("x", 500)
	f := fakeTV(t, func(ws *websocket.Conn) {
		connectEvent(ws, "")
		ws.WriteJSON(map[string]any{"event": long})
		ws.ReadMessage()
	})
	tl := &traceLog{}
	dial(t, f, "", tl.fn)
	deadline := time.Now().Add(2 * time.Second)
	for !strings.Contains(tl.all(), "xxxx") && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if strings.Contains(tl.all(), strings.Repeat("x", 100)) || !strings.Contains(tl.all(), "xxxx") {
		t.Errorf("trace = %q", tl.all())
	}
}

func TestGetInfoCapsClientTimeout(t *testing.T) {
	old := infoTimeout
	infoTimeout = 100 * time.Millisecond
	defer func() { infoTimeout = old }()
	release := make(chan struct{})
	base := infoServer(t, func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-release:
		case <-r.Context().Done():
		}
	})
	defer close(release)
	start := time.Now()
	_, err := GetInfo(context.Background(), &http.Client{Timeout: time.Hour}, base)
	if err == nil {
		t.Fatal("expected timeout")
	}
	if time.Since(start) > 2*time.Second {
		t.Errorf("client timeout not capped: %v", time.Since(start))
	}
}

func TestGetInfoIPv6Host(t *testing.T) {
	// A bare host with a colon must be bracketed, not split as host:port.
	_, err := GetInfo(context.Background(), &http.Client{Timeout: 200 * time.Millisecond}, "::1")
	if err == nil {
		t.Skip("unexpected listener on [::1]:8001")
	}
	if strings.Contains(err.Error(), "invalid port") || strings.Contains(err.Error(), "endereço inválido") {
		t.Errorf("err = %v", err)
	}
}

func TestModelYear(t *testing.T) {
	for code, want := range map[string]string{
		"18_KANTM2_QTV": "2018", "22_NIKEM_UHD": "2022", "": "", "UN55": "", "1_X": "", "AB_CD": "",
	} {
		if got := modelYear(code); got != want {
			t.Errorf("modelYear(%q) = %q, want %q", code, got, want)
		}
	}
}

func TestGetInfoImeSyncedFalseOrAbsent(t *testing.T) {
	for _, body := range []string{
		`{"device":{"ImeSyncedSupport":"false"}}`,
		`{"device":{"ImeSyncedSupport":false}}`,
		`{"device":{}}`,
	} {
		base := infoServer(t, func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(body)) })
		got, err := GetInfo(context.Background(), &http.Client{}, base)
		if err != nil || got.ImeSynced {
			t.Errorf("%s: info = %+v err = %v", body, got, err)
		}
	}
}
