package main

import (
	"context"
	"time"

	"optic/internal/control"
	"optic/internal/domain"
)

// App is the Wails binding: every exported method is callable from React as
// window.go.main.App.<Method> (see frontend/wailsjs/go/main/App.js, kept by
// hand). It holds no logic: TV control is delegated to control.Service and
// window/tray behaviour to Shell (shell.go).
type App struct {
	*Shell
	svc *control.Service
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
