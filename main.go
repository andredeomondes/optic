// Desktop app for OPTIC (Wails + React). The CLI lives in cmd/optic.
//
// This file is the composition root: it builds the adapters, injects them
// into the use cases and hands the result to Wails. Nothing else wires
// dependencies.
package main

import (
	"embed"
	"log"
	"net/http"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"

	"optic/internal/adapter/atv"
	"optic/internal/adapter/compose"
	"optic/internal/adapter/filestore"
	"optic/internal/adapter/github"
	"optic/internal/adapter/mdns"
	"optic/internal/adapter/registry"
	rokuadapter "optic/internal/adapter/roku"
	"optic/internal/adapter/ssdp"
	"optic/internal/control"
	"optic/internal/domain"
	"optic/internal/platform/arp"
	"optic/pkg/atvremote"
	"optic/pkg/wol"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/windows/icon.ico
var trayIcon []byte

// singleInstanceID identifies OPTIC so a second launch focuses the first.
const singleInstanceID = "optic-desktop-8b5cf6"

func main() {
	cfgDir, err := atvremote.DefaultConfigDir()
	if err != nil {
		log.Fatal(err)
	}
	store := filestore.Store{Dir: cfgDir}
	shell := newShell(store, trayIcon)
	tv := atv.New(cfgDir)
	router := registry.Registry{
		Adapters: map[domain.Platform]registry.Adapter{
			domain.PlatformAndroidTV: tv,
			domain.PlatformRoku:      rokuadapter.Adapter{},
		},
		Known: store,
	}
	app := &App{
		Shell: shell,
		svc: control.New(control.Deps{
			Discoverer: compose.Discoverer{Sources: []domain.Discoverer{mdns.Discoverer{}, ssdp.Discoverer{}}},
			Connector:  router,
			Pairer:     router,
			Hosts:      store,
			Known:      store,
			MACs:       arp.Resolver{},
			Waker:      wol.Sender{},
			Power:      router,
			Events:     uiEvents{shell},
			Releases:   github.Client{HTTP: &http.Client{Timeout: 5 * time.Second}, Repo: "andredeomondes/optic", UserAgent: "OPTIC"},
		}),
	}

	err = wails.Run(&options.App{
		Title:            "OPTIC",
		Width:            820,
		Height:           880,
		MinWidth:         460,
		MinHeight:        640,
		Frameless:        true,
		BackgroundColour: &options.RGBA{R: 0, G: 0, B: 0, A: 0},
		AssetServer:      &assetserver.Options{Assets: assets},
		OnStartup:        shell.startup,
		OnShutdown:       app.shutdown,
		OnBeforeClose:    shell.beforeClose,
		Bind:             []interface{}{app},
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId:               singleInstanceID,
			OnSecondInstanceLaunch: shell.onSecondInstance,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
			BackdropType:         windows.Acrylic,
		},
		Mac: &mac.Options{
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
			About:                &mac.AboutInfo{Title: "OPTIC", Message: "Control your TV from your PC."},
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}
