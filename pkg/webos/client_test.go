package webos

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{}

// fakeTV starts a websocket server running h for each connection.
func fakeTV(t *testing.T, h func(ws *websocket.Conn)) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer ws.Close()
		h(ws)
	}))
	t.Cleanup(srv.Close)
	return "ws" + strings.TrimPrefix(srv.URL, "http")
}

func dial(t *testing.T, url string, tr func(string, string)) *Conn {
	t.Helper()
	c, err := Dial(context.Background(), "ignored", Options{URLs: []string{url}, Trace: tr})
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

func TestRegisterPromptThenRegistered(t *testing.T) {
	got := make(chan map[string]any, 1)
	url := fakeTV(t, func(ws *websocket.Conn) {
		m := readMsg(t, ws)
		got <- m
		ws.WriteJSON(map[string]any{"type": "response", "id": "register_0", "payload": map[string]any{"pairingType": "PROMPT"}})
		ws.WriteJSON(map[string]any{"type": "registered", "id": "register_0", "payload": map[string]any{"client-key": "abc"}})
		ws.ReadMessage()
	})
	c := dial(t, url, nil)
	key, err := c.Register(context.Background(), "")
	if err != nil || key != "abc" {
		t.Fatalf("key=%q err=%v", key, err)
	}
	m := <-got
	p := m["payload"].(map[string]any)
	if _, ok := p["client-key"]; ok {
		t.Error("client-key must be omitted when empty")
	}
	if _, ok := p["manifest"]; !ok {
		t.Error("manifest missing")
	}
	if p["pairingType"] != "PROMPT" {
		t.Errorf("pairingType = %v", p["pairingType"])
	}
}

func TestRegisterWithKey(t *testing.T) {
	got := make(chan map[string]any, 1)
	url := fakeTV(t, func(ws *websocket.Conn) {
		got <- readMsg(t, ws)
		ws.WriteJSON(map[string]any{"type": "registered", "id": "register_0", "payload": map[string]any{"client-key": "abc"}})
		ws.ReadMessage()
	})
	c := dial(t, url, nil)
	if _, err := c.Register(context.Background(), "abc"); err != nil {
		t.Fatal(err)
	}
	p := (<-got)["payload"].(map[string]any)
	if p["client-key"] != "abc" {
		t.Errorf("client-key = %v", p["client-key"])
	}
}

func TestRegisterRejected(t *testing.T) {
	url := fakeTV(t, func(ws *websocket.Conn) {
		readMsg(t, ws)
		ws.WriteJSON(map[string]any{"type": "error", "id": "register_0", "error": "403 user rejected"})
		ws.ReadMessage()
	})
	c := dial(t, url, nil)
	_, err := c.Register(context.Background(), "")
	if !errors.Is(err, ErrRegisterRejected) {
		t.Fatalf("err = %v", err)
	}
	if !strings.Contains(err.Error(), "403 user rejected") {
		t.Errorf("TV text missing: %v", err)
	}
}

func TestRegisterHonorsCtx(t *testing.T) {
	url := fakeTV(t, func(ws *websocket.Conn) {
		for {
			if _, _, err := ws.ReadMessage(); err != nil {
				return
			}
		}
	})
	c := dial(t, url, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := c.Register(ctx, "")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v", err)
	}
}

func TestRequestMatchesID(t *testing.T) {
	url := fakeTV(t, func(ws *websocket.Conn) {
		a := readMsg(t, ws)
		b := readMsg(t, ws)
		for _, m := range []map[string]any{b, a} { // out of order
			ws.WriteJSON(map[string]any{"type": "response", "id": m["id"], "payload": map[string]any{"returnValue": true, "echo": m["uri"]}})
		}
		ws.ReadMessage()
	})
	c := dial(t, url, nil)
	var wg sync.WaitGroup
	for _, uri := range []string{"ssap://a", "ssap://b"} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			raw, err := c.Request(context.Background(), uri, nil)
			if err != nil {
				t.Errorf("%s: %v", uri, err)
				return
			}
			var p struct{ Echo string }
			json.Unmarshal(raw, &p)
			if p.Echo != uri {
				t.Errorf("%s got %q", uri, p.Echo)
			}
		}()
	}
	wg.Wait()
}

func TestRequestErrorReturnValueFalse(t *testing.T) {
	url := fakeTV(t, func(ws *websocket.Conn) {
		m := readMsg(t, ws)
		ws.WriteJSON(map[string]any{"type": "response", "id": m["id"], "payload": map[string]any{"returnValue": false, "errorText": "x"}})
		ws.ReadMessage()
	})
	c := dial(t, url, nil)
	_, err := c.Request(context.Background(), "ssap://a", nil)
	if err == nil || !strings.Contains(err.Error(), "x") {
		t.Fatalf("err = %v", err)
	}
}

func TestRequestErrorType(t *testing.T) {
	url := fakeTV(t, func(ws *websocket.Conn) {
		m := readMsg(t, ws)
		ws.WriteJSON(map[string]any{"type": "error", "id": m["id"], "error": "401 insufficient permissions"})
		ws.ReadMessage()
	})
	c := dial(t, url, nil)
	_, err := c.Request(context.Background(), "ssap://a", nil)
	if err == nil || !strings.Contains(err.Error(), "401 insufficient permissions") {
		t.Fatalf("err = %v", err)
	}
}

func TestSubscribeDelivers(t *testing.T) {
	url := fakeTV(t, func(ws *websocket.Conn) {
		m := readMsg(t, ws)
		if m["type"] != "subscribe" {
			t.Errorf("type = %v", m["type"])
		}
		for i := 0; i < 2; i++ {
			ws.WriteJSON(map[string]any{"type": "response", "id": m["id"], "payload": map[string]any{"n": i}})
		}
		ws.ReadMessage()
	})
	c := dial(t, url, nil)
	ch := make(chan json.RawMessage, 4)
	if err := c.Subscribe(context.Background(), "ssap://s", func(r json.RawMessage) { ch <- r }); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		select {
		case <-ch:
		case <-time.After(time.Second):
			t.Fatalf("only %d deliveries", i)
		}
	}
}

func TestPointerButton(t *testing.T) {
	frames := make(chan string, 1)
	ptrURL := fakeTV(t, func(ws *websocket.Conn) {
		_, data, err := ws.ReadMessage()
		if err == nil {
			frames <- string(data)
		}
	})
	url := fakeTV(t, func(ws *websocket.Conn) {
		m := readMsg(t, ws)
		if m["uri"] != "ssap://com.webos.service.networkinput/getPointerInputSocket" {
			t.Errorf("uri = %v", m["uri"])
		}
		ws.WriteJSON(map[string]any{"type": "response", "id": m["id"], "payload": map[string]any{"returnValue": true, "socketPath": ptrURL}})
		ws.ReadMessage()
	})
	c := dial(t, url, nil)
	p, err := c.Pointer(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Button("UP"); err != nil {
		t.Fatal(err)
	}
	select {
	case f := <-frames:
		if f != "type:button\nname:UP\n\n" {
			t.Errorf("frame = %q", f)
		}
	case <-time.After(time.Second):
		t.Fatal("no frame")
	}
}

func TestFallbackURL(t *testing.T) {
	connected := make(chan struct{}, 1)
	ok := fakeTV(t, func(ws *websocket.Conn) {
		connected <- struct{}{}
		ws.ReadMessage()
	})
	c, err := Dial(context.Background(), "x", Options{URLs: []string{"ws://127.0.0.1:1", ok}, Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	select {
	case <-connected:
	case <-time.After(time.Second):
		t.Fatal("did not reach fallback")
	}
}

func TestDropClosesDone(t *testing.T) {
	url := fakeTV(t, func(ws *websocket.Conn) {
		readMsg(t, ws)
		// close without answering
	})
	c := dial(t, url, nil)
	errc := make(chan error, 1)
	go func() {
		_, err := c.Request(context.Background(), "ssap://a", nil)
		errc <- err
	}()
	select {
	case <-c.Done():
	case <-time.After(time.Second):
		t.Fatal("Done not closed")
	}
	select {
	case err := <-errc:
		if err == nil {
			t.Error("expected error")
		}
	case <-time.After(time.Second):
		t.Fatal("request hung")
	}
}

func TestCloseIdempotent(t *testing.T) {
	url := fakeTV(t, func(ws *websocket.Conn) { ws.ReadMessage() })
	c := dial(t, url, nil)
	c.Close()
	c.Close()
	select {
	case <-c.Done():
	case <-time.After(time.Second):
		t.Fatal("Done not closed")
	}
}

func TestTextNotTraced(t *testing.T) {
	url := fakeTV(t, func(ws *websocket.Conn) {
		m := readMsg(t, ws)
		ws.WriteJSON(map[string]any{"type": "response", "id": m["id"], "payload": map[string]any{"returnValue": true}})
		ws.ReadMessage()
	})
	var mu sync.Mutex
	var lines []string
	c := dial(t, url, func(dir, line string) {
		mu.Lock()
		lines = append(lines, dir+" "+line)
		mu.Unlock()
	})
	if _, err := c.Request(context.Background(), "ssap://com.webos.service.ime/insertText", map[string]any{"text": "segredo"}); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	joined := strings.Join(lines, "\n")
	if strings.Contains(joined, "segredo") {
		t.Errorf("text leaked: %s", joined)
	}
	if !strings.Contains(joined, "→ insertText [texto omitido]") {
		t.Errorf("trace = %s", joined)
	}
}
