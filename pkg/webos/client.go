// Package webos is a client for the LG webOS SSAP protocol (JSON over
// WebSocket), with prompt pairing and the pointer-input socket.
package webos

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// ErrRegisterRejected is returned when the TV refuses the pairing.
var ErrRegisterRejected = errors.New("webos: registro recusado pela TV")

// ErrPromptShown is returned by Register with a key when the TV asks the user
// to allow the connection again: the key is no longer accepted.
var ErrPromptShown = errors.New("webos: a TV pediu confirmação")

const (
	registerID = "register_0"
	insertText = "ssap://com.webos.service.ime/insertText"
	pointerURI = "ssap://com.webos.service.networkinput/getPointerInputSocket"
)

// Options tunes Dial.
type Options struct {
	// URLs replaces the default list (wss://host:3001, ws://host:3000).
	URLs    []string
	Timeout time.Duration // default 5s
	// Trace receives a direction ("→" or "←") and a line. Payloads are never traced.
	Trace func(dir, line string)
}

type message struct {
	Type    string          `json:"type"`
	ID      string          `json:"id"`
	Error   string          `json:"error"`
	Payload json.RawMessage `json:"payload"`
}

type pendingReq struct {
	ch chan message
}

type subscription struct {
	uri string
	fn  func(json.RawMessage)
}

// Conn is an SSAP connection. Safe for concurrent use.
type Conn struct {
	ws     *websocket.Conn
	dialer *websocket.Dialer
	opt    Options
	scheme string // scheme and host:port Dial connected to; the pointer socket reuses them
	host   string

	wmu sync.Mutex // serializes writes

	mu      sync.Mutex
	pending map[string]*pendingReq
	subs    map[string]subscription
	seq     int
	pointer *Pointer
	closed  bool

	done      chan struct{}
	closeOnce sync.Once
}

// Dial connects to the TV, trying each URL in order.
func Dial(ctx context.Context, host string, opt Options) (*Conn, error) {
	if opt.Timeout <= 0 {
		opt.Timeout = 5 * time.Second
	}
	urls := opt.URLs
	if len(urls) == 0 {
		urls = []string{"wss://" + host + ":3001", "ws://" + host + ":3000"}
	}
	d := newDialer(opt.Timeout)
	var last error
	for _, u := range urls {
		dctx, cancel := context.WithTimeout(ctx, opt.Timeout)
		ws, _, err := d.DialContext(dctx, u, nil)
		cancel()
		if err != nil {
			last = err
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			continue
		}
		pu, _ := url.Parse(u)
		c := &Conn{
			ws:      ws,
			dialer:  d,
			opt:     opt,
			scheme:  pu.Scheme,
			host:    pu.Host,
			pending: map[string]*pendingReq{},
			subs:    map[string]subscription{},
			done:    make(chan struct{}),
		}
		go c.readLoop()
		return c, nil
	}
	return nil, fmt.Errorf("webos: nao foi possivel conectar: %w", last)
}

func newDialer(timeout time.Duration) *websocket.Dialer {
	return &websocket.Dialer{
		HandshakeTimeout: timeout,
		// LG TVs use a self-signed certificate on the local network; the
		// pairing client-key is what authenticates the TV, not the certificate.
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
}

func (c *Conn) trace(dir, line string) {
	if c.opt.Trace != nil {
		c.opt.Trace(dir, line)
	}
}

func (c *Conn) readLoop() {
	defer func() {
		c.mu.Lock()
		p := c.pointer
		c.mu.Unlock()
		if p != nil {
			p.Close()
		}
		close(c.done)
	}()
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
		c.mu.Lock()
		sub, isSub := c.subs[m.ID]
		pr := c.pending[m.ID]
		c.mu.Unlock()
		if isSub {
			if m.Type == "error" {
				c.trace("←", "erro "+sub.uri)
				continue
			}
			callSafe(sub.fn, m.Payload)
			continue
		}
		if pr != nil {
			select {
			case pr.ch <- m:
			default:
			}
		}
	}
}

// callSafe keeps a panicking callback from killing the reader goroutine.
func callSafe(fn func(json.RawMessage), p json.RawMessage) {
	defer func() { _ = recover() }()
	fn(p)
}

func (c *Conn) write(v any) error {
	c.wmu.Lock()
	defer c.wmu.Unlock()
	c.ws.SetWriteDeadline(time.Now().Add(c.opt.Timeout))
	return c.ws.WriteJSON(v)
}

// withTimeout applies the default timeout when ctx has no deadline.
func (c *Conn) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, c.opt.Timeout)
}

func (c *Conn) addPending(id string) *pendingReq {
	pr := &pendingReq{ch: make(chan message, 4)}
	c.mu.Lock()
	c.pending[id] = pr
	c.mu.Unlock()
	return pr
}

func (c *Conn) dropPending(id string) {
	c.mu.Lock()
	delete(c.pending, id)
	c.mu.Unlock()
}

func (c *Conn) wait(ctx context.Context, pr *pendingReq) (message, error) {
	select {
	case m := <-pr.ch:
		return m, nil
	case <-ctx.Done():
		return message{}, ctx.Err()
	case <-c.done:
		return message{}, errors.New("webos: conexao encerrada")
	}
}

// Register pairs with the TV. With an empty key the TV shows a prompt and
// Register waits for the user; ctx bounds that wait.
func (c *Conn) Register(ctx context.Context, key string) (string, error) {
	payload := map[string]any{
		"forcePairing": false,
		"pairingType":  "PROMPT",
		"manifest":     manifest,
	}
	if key != "" {
		payload["client-key"] = key
	}
	pr := c.addPending(registerID)
	defer c.dropPending(registerID)
	c.trace("→", "register")
	err := c.write(map[string]any{"type": "register", "id": registerID, "payload": payload})
	if err != nil {
		return "", fmt.Errorf("webos: enviar registro: %w", err)
	}
	for {
		m, err := c.wait(ctx, pr)
		if err != nil {
			return "", err
		}
		switch m.Type {
		case "response":
			if key != "" && isPrompt(m.Payload) {
				c.trace("←", "register pediu confirmação")
				return "", ErrPromptShown
			}
			continue // PROMPT: waiting for the user
		case "registered":
			var p struct {
				Key string `json:"client-key"`
			}
			if err := json.Unmarshal(m.Payload, &p); err != nil || p.Key == "" {
				return "", errors.New("webos: resposta de registro sem chave")
			}
			c.trace("←", "registered")
			return p.Key, nil
		case "error":
			c.trace("←", "register erro")
			return "", fmt.Errorf("%w: %s", ErrRegisterRejected, m.Error)
		}
	}
}

func isPrompt(payload json.RawMessage) bool {
	var p struct {
		PairingType string `json:"pairingType"`
	}
	return json.Unmarshal(payload, &p) == nil && p.PairingType == "PROMPT"
}

func (c *Conn) nextID() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.seq++
	return strconv.Itoa(c.seq)
}

// tracedURI is the name used in traces and errors; it hides the typed-text URI.
func tracedURI(uri string) string {
	if uri == insertText {
		return "insertText [texto omitido]"
	}
	return uri
}

// Request sends a request and waits for the matching response.
func (c *Conn) Request(ctx context.Context, uri string, payload any) (json.RawMessage, error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	id := c.nextID()
	pr := c.addPending(id)
	defer c.dropPending(id)
	if uri == insertText {
		c.trace("→", tracedURI(uri))
	} else {
		c.trace("→", "request "+uri)
	}
	msg := map[string]any{"type": "request", "id": id, "uri": uri}
	if payload != nil {
		msg["payload"] = payload
	}
	if err := c.write(msg); err != nil {
		return nil, fmt.Errorf("webos: enviar %s: %w", tracedURI(uri), err)
	}
	m, err := c.wait(ctx, pr)
	if err != nil {
		return nil, err
	}
	if text := errorText(m); text != "" {
		c.trace("←", "response "+tracedURI(uri)+" erro")
		return nil, fmt.Errorf("webos: %s: %s", tracedURI(uri), text)
	}
	c.trace("←", "response "+tracedURI(uri)+" ok")
	return m.Payload, nil
}

// errorText returns the TV's error text, or "" when the message is a success.
func errorText(m message) string {
	if m.Type == "error" {
		if m.Error != "" {
			return m.Error
		}
		return "erro desconhecido"
	}
	var p struct {
		ReturnValue *bool  `json:"returnValue"`
		ErrorText   string `json:"errorText"`
	}
	if json.Unmarshal(m.Payload, &p) == nil && p.ReturnValue != nil && !*p.ReturnValue {
		if p.ErrorText != "" {
			return p.ErrorText
		}
		return "erro desconhecido"
	}
	return ""
}

// Subscribe registers fn for every message the TV sends for the subscription.
// fn runs on the reader goroutine; it must not block or call Request.
// A panic in fn is recovered and ignored. Error messages are not delivered.
func (c *Conn) Subscribe(ctx context.Context, uri string, fn func(json.RawMessage)) error {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return err
	}
	id := c.nextID()
	c.mu.Lock()
	c.subs[id] = subscription{uri: uri, fn: fn}
	c.mu.Unlock()
	c.trace("→", "subscribe "+uri)
	err := c.write(map[string]any{"type": "subscribe", "id": id, "uri": uri})
	if err != nil {
		c.mu.Lock()
		delete(c.subs, id)
		c.mu.Unlock()
		return fmt.Errorf("webos: assinar %s: %w", uri, err)
	}
	return nil
}

// Done is closed when the connection drops.
func (c *Conn) Done() <-chan struct{} { return c.done }

// Close closes the connection and the pointer socket. Idempotent.
func (c *Conn) Close() error {
	var err error
	c.closeOnce.Do(func() {
		c.mu.Lock()
		c.closed = true
		p := c.pointer
		c.mu.Unlock()
		if p != nil {
			p.Close()
		}
		err = c.ws.Close()
	})
	return err
}

// Pointer is the pointer-input socket used for remote buttons.
type Pointer struct {
	ws        *websocket.Conn
	timeout   time.Duration
	wmu       sync.Mutex
	closeOnce sync.Once
}

// Pointer opens the pointer-input socket. A returned *Pointer is invalidated
// by the next call to Pointer.
func (c *Conn) Pointer(ctx context.Context) (*Pointer, error) {
	raw, err := c.Request(ctx, pointerURI, nil)
	if err != nil {
		return nil, err
	}
	var p struct {
		SocketPath string `json:"socketPath"`
	}
	if err := json.Unmarshal(raw, &p); err != nil || p.SocketPath == "" {
		return nil, errors.New("webos: TV nao informou o socket de ponteiro")
	}
	// Only the path and query are trusted; the TV host and scheme are the ones
	// Dial connected to.
	su, err := url.Parse(p.SocketPath)
	if err != nil {
		return nil, errors.New("webos: socket de ponteiro invalido")
	}
	target := url.URL{Scheme: c.scheme, Host: c.host, Path: su.Path, RawQuery: su.RawQuery}
	dctx, cancel := context.WithTimeout(ctx, c.opt.Timeout)
	defer cancel()
	ws, _, err := c.dialer.DialContext(dctx, target.String(), nil)
	if err != nil {
		return nil, errors.New("webos: nao foi possivel conectar ao socket de ponteiro")
	}
	ptr := &Pointer{ws: ws, timeout: c.opt.Timeout}
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		ws.Close()
		return nil, errors.New("webos: conexao encerrada")
	}
	old := c.pointer
	c.pointer = ptr
	c.mu.Unlock()
	if old != nil {
		old.Close()
	}
	// Drain so the socket notices when the TV closes it.
	go func() {
		for {
			if _, _, err := ws.ReadMessage(); err != nil {
				return
			}
		}
	}()
	return ptr, nil
}

// Button presses a remote button (UP, DOWN, ENTER, ...).
func (p *Pointer) Button(name string) error {
	p.wmu.Lock()
	defer p.wmu.Unlock()
	p.ws.SetWriteDeadline(time.Now().Add(p.timeout))
	return p.ws.WriteMessage(websocket.TextMessage, []byte("type:button\nname:"+name+"\n\n"))
}

// Close closes the socket. Idempotent.
func (p *Pointer) Close() error {
	var err error
	p.closeOnce.Do(func() { err = p.ws.Close() })
	return err
}
