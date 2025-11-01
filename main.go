package main

import (
	"context"
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"

	"mib-to-the-future/backend/app"
	"mib-to-the-future/backend/services"
)

//go:embed frontend/dist
var assets embed.FS

func main() {
	application := app.NewApp()
	sys := &services.System{}
	log := &services.Logger{}

	err := wails.Run(&options.App{
		Title:  "MIB to the Future",
		Assets: assets,
		Bind: []interface{}{
			application,
			sys,
			log,
		},
		OnStartup: func(ctx context.Context) {
			application.Startup(ctx)
			log.SetContext(ctx)
			log.StartDemoLogs()
		},
		OnShutdown: func(ctx context.Context) {
			log.StopDemoLogs()
		},
	})

	if err != nil {
		println("Errore avvio:", err.Error())
	}
}
