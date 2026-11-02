package main

import (
	"context"
	"time"

	"optic/internal/control"
	"optic/internal/domain"
	"optic/internal/update"
)

// App is the Wails binding: every exported method is callable from React as
// window.go.main.App.<Method> (see frontend/wailsjs/go/main/App.js, kept by
// hand). It holds no logic: TV control is delegated to control.Service and
// window/tray behaviour to Shell (shell.go).
type App struct {
	*Shell
	svc *control.Service
	upd *updater
}

// startup runs once Wails has a window: the shell first, then the outcome
// of the last update is read (only the instance that got the window does it).
func (a *App) startup(ctx context.Context) {
	a.Shell.startup(ctx)
	a.upd.load()
}

// Discover scans the network for a few seconds.
func (a *App) Discover() ([]domain.TV, error) { return a.svc.Discover(a.ctx) }

// LastHost returns the TV used last time, so the app can reconnect on launch.
func (a *App) LastHost() string { return a.svc.LastHost() }

// Connect opens a control session. The frontend matches the "NOT_PAIRED"
// error to open the pairing dialog.
func (a *App) Connect(ip string) (domain.DeviceInfo, error) { return a.svc.Connect(a.ctx, ip) }

func (a *App) Disconnect() { a.svc.Disconnect() }

// StartPairing makes the TV show a code or a prompt; the result says which.
func (a *App) StartPairing(ip string) (domain.PairingInfo, error) {
	return a.svc.StartPairing(a.ctx, ip)
}

// FinishPairing submits the code; on success the frontend calls Connect.
func (a *App) FinishPairing(code string) error { return a.svc.FinishPairing(code) }

func (a *App) CancelPairing() { a.svc.CancelPairing() }

func (a *App) SendKey(name string) error { return a.svc.SendKey(name) }

// SendText types into the focused text field on the TV.
func (a *App) SendText(text string) error { return a.svc.SendText(text) }

func (a *App) SetVolume(level int) error { return a.svc.SetVolume(level) }

func (a *App) LaunchApp(name string) error { return a.svc.LaunchApp(name) }

// PowerOn wakes the TV at ip with Wake-on-LAN. It fails with a Portuguese
// message when the TV's MAC was never learned.
func (a *App) PowerOn(ip string) error { return a.svc.PowerOn(ip) }

// LatestRelease returns the newest published OPTIC release.
func (a *App) LatestRelease() (domain.Release, error) { return a.svc.LatestRelease(a.ctx) }

// UpdateSupport says whether the latest release can be installed from here.
func (a *App) UpdateSupport() UpdateSupport { return a.upd.support(a.ctx) }

// StartUpdate downloads, verifies and installs the latest release in the
// background, reporting through the update:progress and update:state
// events. A second call while one runs does nothing.
func (a *App) StartUpdate() error {
	a.upd.start(a.ctx)
	return nil
}

// LastUpdateResult is the outcome of the previous update, once (then nil).
func (a *App) LastUpdateResult() *update.Result { return a.upd.lastResult() }

// quitForUpdate closes any TV session or pairing and exits so the update
// script can replace the app.
func (a *App) quitForUpdate() {
	a.svc.CancelPairing()
	a.svc.Disconnect()
	a.Shell.QuitApp()
}

// InstalledApps lists the apps installed on the connected TV.
func (a *App) InstalledApps() ([]domain.App, error) {
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()
	return a.svc.InstalledApps(ctx)
}

// Inputs lists the external inputs of the connected TV.
func (a *App) Inputs() ([]domain.Input, error) {
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()
	return a.svc.Inputs(ctx)
}

// SwitchInput changes the TV to the input with the given id.
func (a *App) SwitchInput(id string) error {
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()
	return a.svc.SwitchInput(ctx, id)
}

// Toast shows a message on the TV.
func (a *App) Toast(msg string) error {
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()
	return a.svc.Toast(ctx, msg)
}

// shutdown runs when the app quits for real.
func (a *App) shutdown(context.Context) {
	a.svc.Disconnect()
	a.Shell.shutdown()
}
