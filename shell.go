package main

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"optic/internal/adapter/filestore"
	"optic/internal/domain"
	"optic/internal/platform/hotkey"
	"optic/internal/platform/tray"
)

// Events sent to the frontend (see frontend/src/hooks).
const (
	evState        = "state"
	evTrace        = "trace"
	evDisconnected = "disconnected"
	evTrayCompact  = "tray:compact"
	evCloseToTray  = "prefs:closeToTray"
	evTrayKey      = "tray:key"
)

// Shell is the desktop side of the app: window, tray and preferences. Its
// exported methods are bound to the frontend through App.
type Shell struct {
	ctx   context.Context
	store filestore.Store
	icon  []byte
	tray  tray.Tray

	closeToTray atomic.Bool
	quitting    atomic.Bool
	hidden      atomic.Bool // window hidden to the tray

	// keys sends a key to the connected TV; injected by main.
	keys func(string) error

	hkMu   sync.Mutex // guards hk, hkErrs and the prefs read-modify-write
	hk     hotkey.Manager
	hkErrs map[string]string
}

func newShell(store filestore.Store, icon []byte) *Shell {
	s := &Shell{store: store, icon: icon}
	s.closeToTray.Store(store.LoadPrefs().CloseToTray)
	return s
}

// startup runs once Wails has a window; ctx is needed by every runtime call.
func (s *Shell) startup(ctx context.Context) {
	s.ctx = ctx
	s.tray = tray.Start(tray.Options{
		Icon:        s.icon,
		Tooltip:     "OPTIC",
		CloseToTray: s.closeToTray.Load(),
		Handlers: tray.Handlers{
			Open: s.ShowWindow,
			ToggleCompact: func(on bool) {
				s.ShowWindow()
				runtime.EventsEmit(s.ctx, evTrayCompact, on)
			},
			ToggleCloseToTray: s.SetCloseToTray,
			Key:               func(k string) { runtime.EventsEmit(s.ctx, evTrayKey, k) },
			Quit:              s.QuitApp,
		},
	})
	s.hkMu.Lock()
	s.hk = hotkey.Start(s.onHotkey)
	s.hkErrs = hotkeyErrors(s.hk.Set(hotkeyBindings(s.store.LoadPrefs().Hotkeys)))
	s.hkMu.Unlock()
}

func (s *Shell) shutdown() {
	s.hkMu.Lock()
	if s.hk != nil {
		s.hk.Close()
	}
	s.hkMu.Unlock()
	s.tray.Stop()
}

// onHotkey runs on its own goroutine for each fired global shortcut.
func (s *Shell) onHotkey(id string) {
	dispatchHotkey(id, s.keys, hotkeyEffects{
		visible: func() bool { return !s.hidden.Load() },
		show:    s.ShowWindow,
		emit:    func(ev string, data ...any) { runtime.EventsEmit(s.ctx, ev, data...) },
	})
}

// GetHotkeys returns the shortcut settings and the result of the last registration.
func (s *Shell) GetHotkeys() HotkeyState {
	s.hkMu.Lock()
	defer s.hkMu.Unlock()
	return buildHotkeyState(hotkey.Supported(), s.store.LoadPrefs().Hotkeys, s.hkErrs)
}

// SetHotkeys validates, saves and registers the shortcuts. Invalid input is
// returned as an error and nothing changes. Unknown action ids are ignored.
func (s *Shell) SetHotkeys(enabled, media bool, bindings map[string]string) (HotkeyState, error) {
	s.hkMu.Lock()
	defer s.hkMu.Unlock()
	prefs := s.store.LoadPrefs()
	valid, err := validateHotkeys(bindings, prefs.Hotkeys.Bindings)
	if err != nil {
		return HotkeyState{}, err
	}
	prefs.Hotkeys = filestore.HotkeyPrefs{Enabled: enabled, Media: media, Bindings: valid}
	if err := s.store.SavePrefs(prefs); err != nil {
		return HotkeyState{}, err
	}
	if s.hk != nil {
		s.hkErrs = hotkeyErrors(s.hk.Set(hotkeyBindings(prefs.Hotkeys)))
	}
	return buildHotkeyState(hotkey.Supported(), prefs.Hotkeys, s.hkErrs), nil
}

// beforeClose turns the close button (and Alt+F4) into "hide to tray" when
// that preference is on and the tray is there to bring the window back.
// Quitting for real is the tray menu's "Sair". Minimize stays the normal one.
func (s *Shell) beforeClose(context.Context) bool {
	if s.quitting.Load() || !s.closeToTray.Load() || !s.tray.Ready() {
		return false
	}
	runtime.WindowHide(s.ctx)
	s.hidden.Store(true)
	return true
}

// onSecondInstance runs when the exe is launched again: no second window,
// the running one comes back instead.
func (s *Shell) onSecondInstance(options.SecondInstanceData) { s.ShowWindow() }

// ShowWindow brings the window back from the tray or the taskbar.
func (s *Shell) ShowWindow() {
	s.hidden.Store(false)
	runtime.WindowShow(s.ctx)
	runtime.WindowUnminimise(s.ctx)
	runtime.WindowSetAlwaysOnTop(s.ctx, true)  // Windows won't hand focus to a background process;
	runtime.WindowSetAlwaysOnTop(s.ctx, false) // a topmost flip raises it anyway
}

// QuitApp exits for real, even when closing hides to the tray.
func (s *Shell) QuitApp() {
	s.quitting.Store(true)
	runtime.Quit(s.ctx)
}

// ToggleFullscreen switches fullscreen and returns the new state. A maximized
// window is restored first: Wails would otherwise keep the work-area size.
func (s *Shell) ToggleFullscreen() bool {
	if runtime.WindowIsFullscreen(s.ctx) {
		runtime.WindowUnfullscreen(s.ctx)
		return false
	}
	runtime.WindowUnmaximise(s.ctx)
	runtime.WindowFullscreen(s.ctx)
	return true
}

// ResizeWindow sets the min size, then the size. Sent from JS as two runtime
// calls they race (each runs in its own goroutine), and SetMinSize, which
// clamps the current size, could undo SetSize.
func (s *Shell) ResizeWindow(minW, minH, w, h int) {
	runtime.WindowUnfullscreen(s.ctx) // a size only applies to a normal window
	runtime.WindowUnmaximise(s.ctx)
	runtime.WindowSetMinSize(s.ctx, minW, minH)
	runtime.WindowSetSize(s.ctx, w, h)
}

// SetCompactMode mirrors the window mode (owned by the frontend) in the tray.
func (s *Shell) SetCompactMode(on bool) { s.tray.SetCompact(on) }

// TrayAvailable lets the UI offer "close to tray" only when it is safe.
func (s *Shell) TrayAvailable() bool { return s.tray.Ready() }

func (s *Shell) CloseToTray() bool { return s.closeToTray.Load() }

func (s *Shell) SetCloseToTray(on bool) {
	s.closeToTray.Store(on)
	s.hkMu.Lock()
	p := s.store.LoadPrefs()
	p.CloseToTray = on
	_ = s.store.SavePrefs(p)
	s.hkMu.Unlock()
	s.tray.SetCloseToTray(on)
	runtime.EventsEmit(s.ctx, evCloseToTray, on)
}

// uiEvents implements domain.EventSink by emitting Wails events.
type uiEvents struct{ shell *Shell }

func (e uiEvents) State(st domain.State)  { runtime.EventsEmit(e.shell.ctx, evState, st) }
func (e uiEvents) Trace(dir, line string) { runtime.EventsEmit(e.shell.ctx, evTrace, dir, line) }
func (e uiEvents) Disconnected()          { runtime.EventsEmit(e.shell.ctx, evDisconnected) }
