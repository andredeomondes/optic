// Package domain holds OPTIC's core types and the ports (interfaces) that the
// use cases depend on. It imports nothing from the outside world: protocol,
// network discovery, storage and UI live in adapters that implement these
// ports, so they can be swapped or faked in tests.
package domain

import (
	"context"
	"errors"
	"time"
)

// Platform identifies the TV operating system, which decides the protocol
// adapter used to control it.
type Platform string

const (
	PlatformAndroidTV Platform = "androidtv"
	PlatformTizen     Platform = "tizen"
	PlatformWebOS     Platform = "webos"
	PlatformRoku      Platform = "roku"
	PlatformUnknown   Platform = ""
)

// Label is the name shown to the user for the platform.
func (p Platform) Label() string {
	switch p {
	case PlatformAndroidTV:
		return "Android TV"
	case PlatformTizen:
		return "Samsung"
	case PlatformWebOS:
		return "LG webOS"
	case PlatformRoku:
		return "Roku"
	default:
		return "Desconhecida"
	}
}

// Capabilities says which features a platform supports, so the UI can hide
// controls the TV cannot handle.
type Capabilities struct {
	Text           bool `json:"text"`
	AppList        bool `json:"appList"`
	AbsoluteVolume bool `json:"absoluteVolume"`
	PowerOn        bool `json:"powerOn"`
	Inputs         bool `json:"inputs"`
	Pointer        bool `json:"pointer"`
	Toast          bool `json:"toast"`
}

// ErrUnsupported means the platform is recognized but not controllable yet.
type ErrUnsupported struct {
	Platform Platform
}

func (e ErrUnsupported) Error() string {
	return e.Platform.Label() + " ainda não é suportada"
}

// TV is a device found on the local network.
type TV struct {
	Name     string   `json:"name"`
	Model    string   `json:"model"`
	IP       string   `json:"ip"`
	Remotev2 bool     `json:"remotev2"` // kept for frontend compatibility
	Platform Platform `json:"platform"`
	MAC      string   `json:"mac,omitempty"`
}

// KnownTV is a TV the user has used before, remembered across launches.
type KnownTV struct {
	IP       string   `json:"ip"`
	Name     string   `json:"name"`
	Platform Platform `json:"platform"`
	MAC      string   `json:"mac"`
}

// DeviceInfo is what a TV reports about itself once connected.
type DeviceInfo struct {
	Model        string       `json:"model"`
	Vendor       string       `json:"vendor"`
	Version      string       `json:"version"`
	AppVersion   string       `json:"appVersion"`
	Platform     Platform     `json:"platform"`
	Capabilities Capabilities `json:"capabilities"`
}

// State is the live state pushed by the TV.
type State struct {
	On        bool   `json:"on"`
	Volume    int    `json:"volume"`
	VolumeMax int    `json:"volumeMax"`
	Muted     bool   `json:"muted"`
	App       string `json:"app"`      // foreground package, e.g. "com.netflix.ninja"
	Keyboard  bool   `json:"keyboard"` // Android on-screen keyboard is open
}

var (
	// ErrNotPaired means the TV rejected our certificate; pair first.
	ErrNotPaired = errors.New("NOT_PAIRED")
	// ErrNotConnected means there is no active session.
	ErrNotConnected = errors.New("não conectado")
	// ErrNoPairing means FinishPairing was called without StartPairing.
	ErrNoPairing = errors.New("nenhum pairing em andamento")
	// ErrCodeTypo means the pairing code was wrong; the session stays open.
	ErrCodeTypo = errors.New("código não confere (erro de digitação?)")
	// ErrVolumeUnknown means the TV hasn't reported its volume yet.
	ErrVolumeUnknown = errors.New("volume atual desconhecido: mude o volume uma vez e tente de novo")
	// ErrNoMAC means Wake-on-LAN was requested for a TV whose MAC was never learned.
	ErrNoMAC = errors.New("MAC da TV desconhecido: conecte uma vez com ela ligada")
	// ErrNoReleaseSource means no release lookup is configured.
	ErrNoReleaseSource = errors.New("checagem de versão indisponível")
)

// Release is a published OPTIC version.
type Release struct {
	Tag         string    `json:"tag"`
	Name        string    `json:"name"`
	URL         string    `json:"url"`
	Notes       string    `json:"notes"`
	PublishedAt time.Time `json:"publishedAt"`
}

// ReleaseSource looks up the latest published release.
type ReleaseSource interface {
	Latest(ctx context.Context) (Release, error)
}

// Discoverer finds TVs on the local network.
type Discoverer interface {
	Discover(ctx context.Context, timeout time.Duration) ([]TV, error)
}

// SessionEvents receives what a live session reports. Called from the
// session's own goroutine.
type SessionEvents interface {
	OnState(State)
	OnTrace(dir, line string)
}

// Session is a live control connection to one TV.
type Session interface {
	Info() DeviceInfo
	State() State
	Done() <-chan struct{}
	SendKey(name string) error
	SendText(text string) error
	LaunchApp(nameOrURI string) error
	Close() error
}

// Connector opens sessions. It returns ErrNotPaired when pairing is needed.
type Connector interface {
	Connect(ctx context.Context, host string, events SessionEvents) (Session, error)
}

// PairingSession is an in-progress pairing: the TV is showing a code.
type PairingSession interface {
	ServerName() string
	// Finish submits the code. ErrCodeTypo keeps the session usable.
	Finish(code string) error
	Close() error
}

// Pairer starts pairing with a TV.
type Pairer interface {
	StartPairing(ctx context.Context, host string) (PairingSession, error)
}

// HostStore remembers the last TV used.
type HostStore interface {
	LastHost() string
	SaveLastHost(host string) error
}

// KnownTVs remembers the TVs the user has used before.
type KnownTVs interface {
	Get(ip string) (KnownTV, bool)
	Put(tv KnownTV) error
	List() []KnownTV
}

// MACResolver finds a device's MAC from its IP (the OS ARP table).
type MACResolver interface {
	LookupMAC(ip string) (string, bool)
}

// WakeSender powers a device on with Wake-on-LAN.
type WakeSender interface {
	Send(mac, ip string) error
}

// EventSink is how use cases notify the UI.
type EventSink interface {
	State(State)
	Trace(dir, line string)
	Disconnected()
}
