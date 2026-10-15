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

// TV is a device found on the local network.
type TV struct {
	Name     string `json:"name"`
	Model    string `json:"model"`
	IP       string `json:"ip"`
	Remotev2 bool   `json:"remotev2"`
}

// DeviceInfo is what a TV reports about itself once connected.
type DeviceInfo struct {
	Model      string `json:"model"`
	Vendor     string `json:"vendor"`
	Version    string `json:"version"`
	AppVersion string `json:"appVersion"`
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
)

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

// EventSink is how use cases notify the UI.
type EventSink interface {
	State(State)
	Trace(dir, line string)
	Disconnected()
}
