// Package roku is a small client for the Roku External Control Protocol (ECP).
package roku

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Port is the ECP HTTP port.
const Port = 8060

const (
	defaultTimeout = 3 * time.Second
	maxBody        = 1 << 20
)

// noRedirect is the default client: ECP never redirects, and following one
// could send commands to another host.
var noRedirect = &http.Client{
	CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
}

// ErrForbidden means the Roku refused network control (HTTP 403).
var ErrForbidden = errors.New("A Roku bloqueou o controle pela rede. Na TV: Configurações › Sistema › Configurações avançadas do sistema › Controle por apps móveis › Acesso à rede: Padrão.")

// Client talks to one Roku. BaseURL, when set, replaces http://Host:8060.
type Client struct {
	HTTP    *http.Client
	Host    string
	BaseURL string
	Timeout time.Duration
}

// App is an installed channel. The home screen has no ID.
type App struct {
	ID, Name, Type string
}

// DeviceInfo is the subset of /query/device-info that OPTIC uses.
type DeviceInfo struct {
	Vendor       string `xml:"vendor-name"`
	Model        string `xml:"model-name"`
	FriendlyName string `xml:"friendly-device-name"`
	UserName     string `xml:"user-device-name"`
	Software     string `xml:"software-version"`
	PowerMode    string `xml:"power-mode"`
	WifiMAC      string `xml:"wifi-mac"`
	EthernetMAC  string `xml:"ethernet-mac"`
	IsTV         bool   `xml:"is-tv"`
}

// Name is the best human-readable name for the device.
func (d DeviceInfo) Name() string {
	switch {
	case d.UserName != "":
		return d.UserName
	case d.FriendlyName != "":
		return d.FriendlyName
	}
	return "Roku " + d.Model
}

// On reports whether the device is fully powered on.
func (d DeviceInfo) On() bool { return d.PowerMode == "PowerOn" }

// MAC prefers the ethernet address over wifi.
func (d DeviceInfo) MAC() string {
	if d.EthernetMAC != "" {
		return d.EthernetMAC
	}
	return d.WifiMAC
}

// Keypress sends one remote key, e.g. "Home" or "VolumeUp".
func (c Client) Keypress(ctx context.Context, key string) error {
	return c.post(ctx, "/keypress/"+url.PathEscape(key))
}

// Text types s one character at a time, stopping at the first error.
func (c Client) Text(ctx context.Context, s string) error {
	for _, r := range s {
		if err := c.post(ctx, "/keypress/Lit_"+strings.ReplaceAll(url.PathEscape(string(r)), "+", "%2B")); err != nil {
			return err
		}
	}
	return nil
}

// Launch starts the channel with the given id.
func (c Client) Launch(ctx context.Context, appID string) error {
	return c.post(ctx, "/launch/"+url.PathEscape(appID))
}

// Apps lists the installed channels.
func (c Client) Apps(ctx context.Context) ([]App, error) {
	var v struct {
		Apps []appXML `xml:"app"`
	}
	if err := c.get(ctx, "/query/apps", &v); err != nil {
		return nil, err
	}
	apps := make([]App, 0, len(v.Apps))
	for _, a := range v.Apps {
		apps = append(apps, a.app())
	}
	return apps, nil
}

// ActiveApp returns the running channel; on the home screen only Name is set.
func (c Client) ActiveApp(ctx context.Context) (App, error) {
	var v struct {
		App appXML `xml:"app"`
	}
	if err := c.get(ctx, "/query/active-app", &v); err != nil {
		return App{}, err
	}
	return v.App.app(), nil
}

// DeviceInfo returns the device description.
func (c Client) DeviceInfo(ctx context.Context) (DeviceInfo, error) {
	var d DeviceInfo
	err := c.get(ctx, "/query/device-info", &d)
	return d, err
}

// Player is the media player state from /query/media-player.
type Player struct {
	State string // play, pause, stop, close, buffer...
}

// MediaPlayer returns the playback state. Older Rokus answer 404.
func (c Client) MediaPlayer(ctx context.Context) (Player, error) {
	var v struct {
		State string `xml:"state,attr"`
	}
	if err := c.get(ctx, "/query/media-player", &v); err != nil {
		return Player{}, err
	}
	return Player{State: v.State}, nil
}

type appXML struct {
	ID   string `xml:"id,attr"`
	Type string `xml:"type,attr"`
	Name string `xml:",chardata"`
}

func (a appXML) app() App { return App{ID: a.ID, Name: strings.TrimSpace(a.Name), Type: a.Type} }

func (c Client) base() string {
	if c.BaseURL != "" {
		return c.BaseURL
	}
	return "http://" + net.JoinHostPort(c.Host, strconv.Itoa(Port))
}

func (c Client) do(ctx context.Context, method, path string) (*http.Response, context.CancelFunc, error) {
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	req, err := http.NewRequestWithContext(ctx, method, c.base()+path, nil)
	if err != nil {
		cancel()
		return nil, nil, fmt.Errorf("roku: %s: requisição inválida", method)
	}
	if method == http.MethodPost {
		req.ContentLength = 0
		req.Body = http.NoBody
	}
	hc := c.HTTP
	if hc == nil {
		hc = noRedirect
	}
	resp, err := hc.Do(req)
	if err != nil {
		cancel()
		// url.Error embeds the full URL, which would leak typed characters.
		var ue *url.Error
		if errors.As(err, &ue) {
			err = fmt.Errorf("roku: %s: %w", method, ue.Err)
		}
		return nil, nil, err
	}
	if resp.StatusCode == http.StatusForbidden {
		resp.Body.Close()
		cancel()
		return nil, nil, ErrForbidden
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		resp.Body.Close()
		cancel()
		return nil, nil, fmt.Errorf("roku: HTTP %d", resp.StatusCode)
	}
	return resp, cancel, nil
}

func (c Client) post(ctx context.Context, path string) error {
	resp, cancel, err := c.do(ctx, http.MethodPost, path)
	if err != nil {
		return err
	}
	defer cancel()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxBody))
	return resp.Body.Close()
}

func (c Client) get(ctx context.Context, path string, out any) error {
	resp, cancel, err := c.do(ctx, http.MethodGet, path)
	if err != nil {
		return err
	}
	defer cancel()
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBody+1))
	if err != nil {
		return err
	}
	if len(data) > maxBody {
		return errors.New("roku: response too large")
	}
	if err := xml.Unmarshal(data, out); err != nil {
		return fmt.Errorf("roku: bad response: %w", err)
	}
	return nil
}
