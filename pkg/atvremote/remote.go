package atvremote

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"strings"
	"sync"
	"time"

	"google.golang.org/protobuf/encoding/protowire"
)

// Control messages (port 6466), from the community-documented remotemessage.proto.
//
//	message RemoteMessage {
//	  RemoteConfigure            remote_configure        = 1;
//	  RemoteSetActive            remote_set_active       = 2;
//	  RemoteError                remote_error            = 3;
//	  RemotePingRequest          remote_ping_request     = 8;
//	  RemotePingResponse         remote_ping_response    = 9;
//	  RemoteKeyInject            remote_key_inject       = 10;
//	  RemoteStart                remote_start            = 40;
//	  RemoteSetVolumeLevel       remote_set_volume_level = 50;
//	  RemoteAppLinkLaunchRequest remote_app_link_launch_request = 90;
//	}
const (
	rmConfigure   protowire.Number = 1
	rmSetActive   protowire.Number = 2
	rmError       protowire.Number = 3
	rmPingRequest protowire.Number = 8
	rmPingResp    protowire.Number = 9
	rmKeyInject   protowire.Number = 10
	rmStart       protowire.Number = 40
	rmVolume      protowire.Number = 50
	rmAppLink     protowire.Number = 90

	// Feature bitmask we announce: ping, key, IME, power, volume, app link.
	clientFeatures = 622
)

// Key directions for RemoteKeyInject.
const (
	DirStartLong = 1
	DirEndLong   = 2
	DirShort     = 3
)

// DeviceInfo is what the TV reports about itself in RemoteConfigure.
type DeviceInfo struct {
	Model      string `json:"model"`
	Vendor     string `json:"vendor"`
	Version    string `json:"version"`
	AppVersion string `json:"appVersion"`
}

// State is pushed by the TV whenever power or volume changes.
type State struct {
	On        bool `json:"on"`
	Volume    int  `json:"volume"`
	VolumeMax int  `json:"volumeMax"`
	Muted     bool `json:"muted"`
	// App is the package in the foreground, e.g. "com.netflix.ninja".
	App string `json:"app"`
	// Keyboard is true while the TV has the Android on-screen keyboard open,
	// which is when SendText can replace the field's content in one go.
	Keyboard bool `json:"keyboard"`
}

// Remote is a live control session with a paired TV.
type Remote struct {
	conn *tls.Conn
	wmu  sync.Mutex

	mu    sync.Mutex
	info  DeviceInfo
	state State
	ime   imeState

	ready    chan struct{}
	done     chan struct{}
	err      error
	onState  func(State)
	trace    Tracer
	closeOne sync.Once
}

// ErrNotPaired is returned when the TV rejects our certificate.
var ErrNotPaired = errors.New("TV recusou a conexão: rode o pairing primeiro")

// Connect opens a control session and waits until the TV marks it active.
// onState (optional) is called from the read loop on every state change.
func Connect(ctx context.Context, host string, cert tls.Certificate, onState func(State), trace Tracer) (*Remote, error) {
	conn, err := dialTLS(ctx, host, ControlPort, cert)
	if err != nil {
		// A TLS alert means the TV doesn't know our certificate; anything
		// else (timeout, refused) is a plain network problem.
		if strings.Contains(err.Error(), "tls:") {
			return nil, ErrNotPaired
		}
		return nil, err
	}
	r := &Remote{
		conn:    conn,
		ready:   make(chan struct{}),
		done:    make(chan struct{}),
		onState: onState,
		trace:   trace,
	}
	go r.readLoop()

	select {
	case <-r.ready:
		return r, nil
	case <-r.done:
		if r.err != nil {
			return nil, r.err
		}
		return nil, ErrNotPaired
	case <-time.After(stepTimeout):
		r.Close()
		return nil, errors.New("TV não ativou a sessão de controle")
	case <-ctx.Done():
		r.Close()
		return nil, ctx.Err()
	}
}

func (r *Remote) Info() DeviceInfo {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.info
}

func (r *Remote) State() State {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.state
}

// Done is closed when the session ends; Err tells why.
func (r *Remote) Done() <-chan struct{} { return r.done }
func (r *Remote) Err() error           { return r.err }

func (r *Remote) Close() error {
	r.closeOne.Do(func() { r.conn.Close() })
	return nil
}

// SendKey presses and releases an Android keycode (see keys.go).
func (r *Remote) SendKey(code int) error {
	return r.sendKeyDir(code, DirShort)
}

func (r *Remote) sendKeyDir(code, dir int) error {
	var p []byte
	p = appendUint(p, 1, uint64(code))
	p = appendUint(p, 2, uint64(dir))
	return r.send(rmKeyInject, p)
}

// LaunchApp opens a deep link or "market://launch?id=<package>" URI.
func (r *Remote) LaunchApp(link string) error {
	return r.send(rmAppLink, appendString(nil, 1, link))
}

func (r *Remote) send(field protowire.Number, payload []byte) error {
	select {
	case <-r.done:
		return errors.New("sessão de controle encerrada")
	default:
	}
	msg := appendMsg(nil, field, payload)
	if r.trace != nil {
		r.trace(">", msg)
	}
	r.wmu.Lock()
	defer r.wmu.Unlock()
	r.conn.SetWriteDeadline(time.Now().Add(stepTimeout))
	return writeFrame(r.conn, msg)
}

func (r *Remote) readLoop() {
	defer close(r.done)
	defer r.Close()
	br := bufio.NewReader(r.conn)
	readyOnce := sync.Once{}

	for {
		msg, err := readFrame(br)
		if err != nil {
			r.err = err
			return
		}
		if r.trace != nil {
			r.trace("<", msg)
		}
		f, err := parseFields(msg)
		if err != nil {
			continue
		}

		if b, ok := f.bytes(rmConfigure); ok {
			r.handleConfigure(b)
		}
		if _, ok := f[rmSetActive]; ok {
			r.send(rmSetActive, appendUint(nil, 1, clientFeatures))
			readyOnce.Do(func() { close(r.ready) })
		}
		if b, ok := f.bytes(rmPingRequest); ok {
			pf, _ := parseFields(b)
			v, _ := pf.uint(1)
			r.send(rmPingResp, appendUint(nil, 1, v))
		}
		if b, ok := f.bytes(rmStart); ok {
			sf, _ := parseFields(b)
			on, _ := sf.uint(1)
			r.updateState(func(s *State) { s.On = on != 0 })
		}
		if b, ok := f.bytes(rmVolume); ok {
			// RemoteSetVolumeLevel: 6 = volume_max, 7 = volume_level, 8 = volume_muted
			vf, _ := parseFields(b)
			r.updateState(func(s *State) {
				if v, ok := vf.uint(6); ok {
					s.VolumeMax = int(v)
				}
				if v, ok := vf.uint(7); ok {
					s.Volume = int(v)
				}
				m, _ := vf.uint(8)
				s.Muted = m != 0
			})
		}
		r.handleIme(f)
		if _, ok := f[rmError]; ok {
			r.err = errors.New("TV reportou erro na sessão")
		}
	}
}

func (r *Remote) handleConfigure(b []byte) {
	cf, err := parseFields(b)
	if err == nil {
		if di, ok := cf.bytes(2); ok {
			df, _ := parseFields(di)
			str := func(n protowire.Number) string { v, _ := df.bytes(n); return string(v) }
			r.mu.Lock()
			r.info = DeviceInfo{Model: str(1), Vendor: str(2), Version: str(4), AppVersion: str(6)}
			r.mu.Unlock()
		}
	}

	// Answer with our own RemoteConfigure.
	var dev []byte
	dev = appendString(dev, 1, "OPTIC")
	dev = appendString(dev, 2, "OPTIC")
	dev = appendUint(dev, 3, 1)
	dev = appendString(dev, 4, "1")
	dev = appendString(dev, 5, "optic")
	dev = appendString(dev, 6, "0.1.0")
	var p []byte
	p = appendUint(p, 1, clientFeatures)
	p = appendMsg(p, 2, dev)
	r.send(rmConfigure, p)
}

func (r *Remote) updateState(fn func(*State)) {
	r.mu.Lock()
	fn(&r.state)
	s := r.state
	r.mu.Unlock()
	if r.onState != nil {
		r.onState(s)
	}
}
