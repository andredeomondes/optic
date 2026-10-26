// Package tizen is a client for the Samsung Tizen remote-control protocol
// (JSON over WebSocket) and its REST info endpoint.
package tizen

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	// ErrUnauthorized is returned when the TV refuses the connection (pairing denied).
	ErrUnauthorized = errors.New("tizen: conexão não autorizada pela TV")
	// ErrTimeout is returned when the TV does not answer the connection handshake in time.
	ErrTimeout = errors.New("tizen: a TV não respondeu")
)

const (
	remoteName   = "T1BUSUM=" // base64("OPTIC"), the name the TV shows in the prompt
	remotePath   = "/api/v2/channels/samsung.remote.control"
	secureAddr   = ":8002"
	plainAddr    = ":8001"
	maxFrameSize = 4 << 20
)

// Options tunes Dial.
type Options struct {
	// URL replaces the scheme and host:port Dial connects to (for tests,
	// e.g. "ws://127.0.0.1:1234"). The path and query are still computed.
	URL     string
	Timeout time.Duration // default 5s; per write and per request without a deadline
	// Trace receives a direction ("→" or "←") and a line. Payloads are never traced.
	Trace func(dir, line string)
}

// App is an app installed on the TV.
type App struct {
	ID   string
	Name string
	Type int
}

type message struct {
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data"`
}

// Conn is a remote-control connection. Safe for concurrent use.
type Conn struct {
	ws  *websocket.Conn
	opt Options

	wmu sync.Mutex // serializes writes

	mu      sync.Mutex
	first   chan message // first handshake event
	appsCh  chan []App   // set while Apps waits for the answer
	onIME   func(open bool)
	appsMu  sync.Mutex // one Apps request at a time
	done    chan struct{}
	closeFn sync.Once
}

// Dial connects to the TV and waits for the handshake result. With secure it
// uses wss on 8002 and the token (omitted when empty); otherwise ws on 8001.
// It returns the connection and the token the TV sent (may be empty or the
// same as token). ctx bounds the wait: use ~60s when pairing, ~10s otherwise.
func Dial(ctx context.Context, host, token string, secure bool, opt Options) (*Conn, string, error) {
	if opt.Timeout <= 0 {
		opt.Timeout = 5 * time.Second
	}
	u := buildURL(host, token, secure, opt.URL)
	d := &websocket.Dialer{
		HandshakeTimeout: opt.Timeout,
		// Samsung TVs use a self-signed certificate on the local network; the
		// token the TV hands out is what authenticates the remote, not the certificate.
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	dctx, cancel := context.WithTimeout(ctx, opt.Timeout)
	ws, _, err := d.DialContext(dctx, u, nil)
	cancel()
	if err != nil {
		if ctx.Err() != nil {
			return nil, "", ErrTimeout
		}
		return nil, "", dialError(err, u)
	}
	ws.SetReadLimit(maxFrameSize)
	c := &Conn{
		ws:    ws,
		opt:   opt,
		first: make(chan message, 1),
		done:  make(chan struct{}),
	}
	go c.readLoop()

	select {
	case m := <-c.first:
		switch m.Event {
		case "ms.channel.connect":
			var d struct {
				Token string `json:"token"`
			}
			_ = json.Unmarshal(m.Data, &d)
			return c, d.Token, nil
		case "ms.channel.unauthorized":
			c.Close()
			return nil, "", ErrUnauthorized
		default: // ms.channel.timeOut
			c.Close()
			return nil, "", ErrTimeout
		}
	case <-ctx.Done():
		c.Close()
		return nil, "", ErrTimeout
	case <-c.done:
		c.Close()
		return nil, "", errors.New("tizen: conexão encerrada")
	}
}

// buildURL composes the ws URL; the token travels in the query, so the URL
// must never reach an error string or a trace.
func buildURL(host, token string, secure bool, override string) string {
	scheme, addr := "ws", host+plainAddr
	if secure {
		scheme, addr = "wss", host+secureAddr
	}
	if override != "" {
		if ou, err := url.Parse(override); err == nil && ou.Host != "" {
			scheme, addr = ou.Scheme, ou.Host
		}
	}
	q := url.Values{}
	q.Set("name", remoteName)
	if secure && token != "" {
		q.Set("token", token)
	}
	u := url.URL{Scheme: scheme, Host: addr, Path: remotePath, RawQuery: q.Encode()}
	return u.String()
}

// dialError describes a connection failure without the URL (it holds the token).
func dialError(err error, u string) error {
	var ue *url.Error
	if errors.As(err, &ue) {
		err = ue.Err
	}
	var ne *net.OpError
	if errors.As(err, &ne) && ne.Err != nil {
		err = ne.Err
	}
	msg := strings.ReplaceAll(err.Error(), u, "")
	return errors.New("tizen: não foi possível conectar: " + msg)
}

func (c *Conn) trace(dir, line string) {
	if c.opt.Trace != nil {
		c.opt.Trace(dir, line)
	}
}

func (c *Conn) readLoop() {
	defer close(c.done)
	for {
		var m message
		if err := c.ws.ReadJSON(&m); err != nil {
			var syn *json.SyntaxError
			var ute *json.UnmarshalTypeError
			if errors.As(err, &syn) || errors.As(err, &ute) {
				continue
			}
			return
		}
		if m.Event == "" {
			continue
		}
		c.trace("←", "event "+m.Event)
		switch m.Event {
		case "ms.channel.connect", "ms.channel.unauthorized", "ms.channel.timeOut":
			select {
			case c.first <- m:
			default:
			}
		case "ed.installedApp.get":
			c.deliverApps(m.Data)
		case "ms.remote.imeStart":
			c.callIME(true)
		case "ms.remote.imeEnd":
			c.callIME(false)
		}
	}
}

func (c *Conn) deliverApps(data json.RawMessage) {
	var d struct {
		Data []struct {
			AppID   string `json:"appId"`
			Name    string `json:"name"`
			AppType int    `json:"app_type"`
		} `json:"data"`
	}
	_ = json.Unmarshal(data, &d)
	apps := make([]App, 0, len(d.Data))
	for _, a := range d.Data {
		apps = append(apps, App{ID: a.AppID, Name: a.Name, Type: a.AppType})
	}
	c.mu.Lock()
	ch := c.appsCh
	c.mu.Unlock()
	if ch != nil {
		select {
		case ch <- apps:
		default:
		}
	}
}

func (c *Conn) callIME(open bool) {
	c.mu.Lock()
	fn := c.onIME
	c.mu.Unlock()
	if fn == nil {
		return
	}
	defer func() { _ = recover() }() // a panicking callback must not kill the reader
	fn(open)
}

// OnIME registers fn for the TV keyboard opening (true) and closing (false).
// fn runs on the reader goroutine; it must not block.
func (c *Conn) OnIME(fn func(open bool)) {
	c.mu.Lock()
	c.onIME = fn
	c.mu.Unlock()
}

// write sends the frames in order, atomically with respect to other writers.
func (c *Conn) write(frames ...any) error {
	c.wmu.Lock()
	defer c.wmu.Unlock()
	for _, f := range frames {
		c.ws.SetWriteDeadline(time.Now().Add(c.opt.Timeout))
		if err := c.ws.WriteJSON(f); err != nil {
			return errors.New("tizen: enviar: " + err.Error())
		}
	}
	return nil
}

// Key presses a remote key (KEY_HOME, KEY_ENTER, ...).
func (c *Conn) Key(name string) error {
	c.trace("→", "key "+name)
	return c.write(map[string]any{
		"method": "ms.remote.control",
		"params": map[string]any{
			"Cmd":          "Click",
			"DataOfCmd":    name,
			"Option":       "false",
			"TypeOfRemote": "SendRemoteKey",
		},
	})
}

// Text types s into the TV keyboard that is open.
func (c *Conn) Text(s string) error {
	c.trace("→", "SendInputString [texto omitido]")
	return c.write(
		map[string]any{
			"method": "ms.remote.control",
			"params": map[string]any{
				"Cmd":          base64.StdEncoding.EncodeToString([]byte(s)),
				"DataOfCmd":    "base64",
				"TypeOfRemote": "SendInputString",
			},
		},
		map[string]any{
			"method": "ms.remote.control",
			"params": map[string]any{"TypeOfRemote": "SendInputEnd"},
		},
	)
}

func (c *Conn) emit(event string, data any) error {
	c.trace("→", "emit "+event)
	params := map[string]any{"event": event, "to": "host"}
	if data != nil {
		params["data"] = data
	}
	return c.write(map[string]any{"method": "ms.channel.emit", "params": params})
}

// Apps lists the apps installed on the TV. Without a ctx deadline it waits
// the default timeout.
func (c *Conn) Apps(ctx context.Context) ([]App, error) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.opt.Timeout)
		defer cancel()
	}
	c.appsMu.Lock()
	defer c.appsMu.Unlock()
	ch := make(chan []App, 1)
	c.mu.Lock()
	c.appsCh = ch
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		c.appsCh = nil
		c.mu.Unlock()
	}()
	if err := c.emit("ed.installedApp.get", nil); err != nil {
		return nil, err
	}
	select {
	case apps := <-ch:
		return apps, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-c.done:
		return nil, errors.New("tizen: conexão encerrada")
	}
}

// Launch starts an app. Type 2 apps are launched by deep link.
func (c *Conn) Launch(appID string, appType int) error {
	action := "NATIVE_LAUNCH"
	if appType == 2 {
		action = "DEEP_LINK"
	}
	return c.emit("ed.apps.launch", map[string]any{"appId": appID, "action_type": action})
}

// Done is closed when the connection drops.
func (c *Conn) Done() <-chan struct{} { return c.done }

// Close closes the connection. Idempotent.
func (c *Conn) Close() error {
	var err error
	c.closeFn.Do(func() { err = c.ws.Close() })
	return err
}
