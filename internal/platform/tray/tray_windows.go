//go:build windows

package tray

import (
	"runtime"
	"sync/atomic"
	"time"

	"github.com/energye/systray"
)

// stopWait bounds how long Stop waits for the icon to be removed.
const stopWait = time.Second

type winTray struct {
	ready       atomic.Bool
	done        chan struct{}
	compact     *systray.MenuItem
	closeToTray *systray.MenuItem
}

// Start shows the icon on a dedicated OS thread and returns immediately.
func Start(o Options) Tray {
	t := &winTray{done: make(chan struct{})}
	go func() {
		runtime.LockOSThread() // the tray window and its message loop share this thread
		systray.Run(func() { t.onReady(o) }, func() { close(t.done) })
	}()
	return t
}

func (t *winTray) onReady(o Options) {
	h := o.Handlers
	systray.SetIcon(o.Icon)
	systray.SetTooltip(o.Tooltip)
	systray.SetOnClick(func(systray.IMenu) { h.Open() })

	systray.AddMenuItem("Abrir OPTIC", "").Click(h.Open)

	volume := systray.AddMenuItem("Volume", "")
	for _, k := range []struct{ label, key string }{
		{"Aumentar", "VOLUME_UP"}, {"Diminuir", "VOLUME_DOWN"}, {"Mudo", "MUTE"},
	} {
		key := k.key
		volume.AddSubMenuItem(k.label, "").Click(func() {
			if h.Key != nil {
				h.Key(key)
			}
		})
	}

	t.compact = systray.AddMenuItemCheckbox("Modo compacto", "", false)
	t.compact.Click(func() { h.ToggleCompact(!t.compact.Checked()) })

	t.closeToTray = systray.AddMenuItemCheckbox("Fechar esconde na bandeja", "", o.CloseToTray)
	t.closeToTray.Click(func() { h.ToggleCloseToTray(!t.closeToTray.Checked()) })

	systray.AddSeparator()
	systray.AddMenuItem("Sair", "").Click(h.Quit)

	t.ready.Store(true)
}

func (t *winTray) Ready() bool { return t.ready.Load() }

func (t *winTray) SetCompact(on bool)     { t.check(t.compact, on) }
func (t *winTray) SetCloseToTray(on bool) { t.check(t.closeToTray, on) }

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
