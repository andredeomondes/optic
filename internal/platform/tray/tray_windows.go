//go:build windows

package tray

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/energye/systray"
)

// stopWait bounds how long Stop waits for the icon to be removed.
const stopWait = time.Second

type winTray struct {
	ready       atomic.Bool
	done        chan struct{}

	mu          sync.Mutex // guards lang and the item handles
	lang        string
	open        *systray.MenuItem
	volume      *systray.MenuItem
	volUp       *systray.MenuItem
	volDown     *systray.MenuItem
	volMute     *systray.MenuItem
	compact     *systray.MenuItem
	closeToTray *systray.MenuItem
	quit        *systray.MenuItem
}

// Start shows the icon on a dedicated OS thread and returns immediately.
func Start(o Options) Tray {
	t := &winTray{done: make(chan struct{}), lang: o.Language}
	go func() {
		runtime.LockOSThread() // the tray window and its message loop share this thread
		systray.Run(func() { t.onReady(o) }, func() { close(t.done) })
	}()
	return t
}

func (t *winTray) onReady(o Options) {
	h := o.Handlers
	t.mu.Lock()
	defer t.mu.Unlock()
	l := trayLabels(t.lang)
	systray.SetIcon(o.Icon)
	systray.SetTooltip(o.Tooltip)
	systray.SetOnClick(func(systray.IMenu) { h.Open() })

	t.open = systray.AddMenuItem(l.Open, "")
	t.open.Click(h.Open)

	t.volume = systray.AddMenuItem(l.Volume, "")
	for _, k := range []struct {
		item  **systray.MenuItem
		label string
		key   string
	}{
		{&t.volUp, l.Up, "VOLUME_UP"}, {&t.volDown, l.Down, "VOLUME_DOWN"}, {&t.volMute, l.Mute, "MUTE"},
	} {
		key := k.key
		*k.item = t.volume.AddSubMenuItem(k.label, "")
		(*k.item).Click(func() {
			if h.Key != nil {
				h.Key(key)
			}
		})
	}

	t.compact = systray.AddMenuItemCheckbox(l.Compact, "", false)
	t.compact.Click(func() { h.ToggleCompact(!t.compact.Checked()) })

	t.closeToTray = systray.AddMenuItemCheckbox(l.CloseToTray, "", o.CloseToTray)
	t.closeToTray.Click(func() { h.ToggleCloseToTray(!t.closeToTray.Checked()) })

	systray.AddSeparator()
	t.quit = systray.AddMenuItem(l.Quit, "")
	t.quit.Click(h.Quit)

	t.ready.Store(true)
}

func (t *winTray) Ready() bool { return t.ready.Load() }

func (t *winTray) SetCompact(on bool)     { t.check(t.compact, on) }
func (t *winTray) SetCloseToTray(on bool) { t.check(t.closeToTray, on) }

// SetLanguage relabels the menu; before the icon is up it only records the language.
func (t *winTray) SetLanguage(lang string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.lang = lang
	if !t.ready.Load() {
		return
	}
	l := trayLabels(lang)
	for item, title := range map[*systray.MenuItem]string{
		t.open: l.Open, t.volume: l.Volume, t.volUp: l.Up, t.volDown: l.Down, t.volMute: l.Mute,
		t.compact: l.Compact, t.closeToTray: l.CloseToTray, t.quit: l.Quit,
	} {
		item.SetTitle(title)
	}
}

func (t *winTray) check(item *systray.MenuItem, on bool) {
	if !t.ready.Load() {
		return
	}
	if on {
		item.Check()
	} else {
		item.Uncheck()
	}
}

func (t *winTray) Stop() {
	systray.Quit()
	select {
	case <-t.done:
	case <-time.After(stopWait):
	}
}
