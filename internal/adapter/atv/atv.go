// Package atv adapts pkg/atvremote (Android TV Remote protocol v2) to the
// domain ports Connector, Pairer and Session.
package atv

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"strings"

	"optic/internal/domain"
	"optic/pkg/atvremote"
)

// clientName is how OPTIC shows up in the TV's list of paired remotes.
const clientName = "OPTIC"

// traceMax caps the length of one protocol trace line sent to the UI.
const traceMax = 120

// Client connects and pairs using the client certificate in cfgDir.
type Client struct {
	cert    tls.Certificate
	certErr error
}

// New loads (or creates on first run) the client certificate. A failure is
// kept and returned by Connect/StartPairing, so the app still starts.
func New(cfgDir string) *Client {
	c := &Client{}
	c.cert, c.certErr = atvremote.LoadOrCreateCert(cfgDir)
	return c
}

func (c *Client) Connect(ctx context.Context, host string, ev domain.SessionEvents) (domain.Session, error) {
	if c.certErr != nil {
		return nil, c.certErr
	}
	onState := func(s atvremote.State) { ev.OnState(toState(s)) }
	r, err := atvremote.Connect(ctx, host, c.cert, onState, tracer(ev))
	if errors.Is(err, atvremote.ErrNotPaired) {
		return nil, domain.ErrNotPaired
	}
	if err != nil {
		return nil, err
	}
	return session{r}, nil
}

func (c *Client) StartPairing(ctx context.Context, host string) (domain.PairingSession, error) {
	if c.certErr != nil {
		return nil, c.certErr
	}
	p, err := atvremote.StartPairing(ctx, host, c.cert, clientName, nil)
	if err != nil {
		return nil, err
	}
	return pairing{p}, nil
}

// tracer turns raw frames into one-line summaries for the activity log.
func tracer(ev domain.SessionEvents) atvremote.Tracer {
	return func(dir string, msg []byte) {
		line := strings.Join(strings.Fields(atvremote.DumpRaw(msg)), " ")
		if len(line) > traceMax {
			line = line[:traceMax] + "…"
		}
		ev.OnTrace(dir, line)
	}
}

type session struct{ r *atvremote.Remote }

func (s session) Info() domain.DeviceInfo {
	i := s.r.Info()
	return domain.DeviceInfo{Model: i.Model, Vendor: i.Vendor, Version: i.Version, AppVersion: i.AppVersion}
}

func (s session) State() domain.State     { return toState(s.r.State()) }
func (s session) Done() <-chan struct{}   { return s.r.Done() }
func (s session) Close() error            { return s.r.Close() }
func (s session) SendText(t string) error { return s.r.SendText(t) }

func (s session) SendKey(name string) error {
	code, ok := atvremote.LookupKey(name)
	if !ok {
		return fmt.Errorf("tecla desconhecida: %s", name)
	}
	return s.r.SendKey(code)
}

// LaunchApp accepts a shortcut ("youtube") or a full deep link / market URI.
func (s session) LaunchApp(nameOrURI string) error {
	link := atvremote.Apps[nameOrURI]
	if link == "" {
		link = nameOrURI
	}
	return s.r.LaunchApp(link)
}

type pairing struct{ p *atvremote.Pairing }

func (p pairing) ServerName() string { return p.p.ServerName }
func (p pairing) Close() error       { return p.p.Close() }
func (p pairing) Finish(code string) error {
	err := p.p.Finish(code)
	if errors.Is(err, atvremote.ErrCodeTypo) {
		return domain.ErrCodeTypo
	}
	return err
}

func toState(s atvremote.State) domain.State {
	return domain.State{
		On: s.On, Volume: s.Volume, VolumeMax: s.VolumeMax, Muted: s.Muted,
		App: s.App, Keyboard: s.Keyboard,
	}
}
