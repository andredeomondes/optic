// Package tray shows OPTIC's icon in the system tray (the Windows
// notification area). It knows nothing about Wails: the desktop shell passes
// callbacks in and reflects state out, so the tray can be replaced or stubbed.
//
// Wails v2 has no tray API (it arrives in v3). On Windows the tray runs on
// energye/systray, pure Go, with its own hidden window and message loop on a
// dedicated OS thread. Other platforms get a no-op tray: on macOS a tray must
// own the main thread, which Wails already uses.
package tray

// Handlers are called from the tray's own thread when the user picks an item.
type Handlers struct {
	Open              func() // left click or "Abrir OPTIC"
	ToggleCompact     func(on bool)
	ToggleCloseToTray func(on bool)
	Quit              func()
}

// Tray is a running tray icon.
type Tray interface {
	// Ready reports whether the icon is up, i.e. hiding the window is safe.
	Ready() bool
	SetCompact(on bool)
	SetCloseToTray(on bool)
	// Stop removes the icon, waiting briefly so no ghost icon is left behind.
	Stop()
}

// Options configure Start.
type Options struct {
	Icon        []byte // .ico bytes on Windows
	Tooltip     string
	CloseToTray bool // initial state of the checkbox
	Handlers    Handlers
}

// noop is the tray on platforms without support.
type noop struct{}

func (noop) Ready() bool         { return false }
func (noop) SetCompact(bool)     {}
func (noop) SetCloseToTray(bool) {}
func (noop) Stop()               {}
