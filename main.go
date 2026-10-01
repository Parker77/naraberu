package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()
	dataRoot := resolveDataRoot()
	if err := ensureDataLayout(dataRoot); err != nil {
		log.Printf("data layout: %v", err)
	}
	app.initStorage(dataRoot)

	wailsApp := application.New(application.Options{
		Name:        "Naraberu",
		Description: "Anime series tracker",
		Services: []application.Service{
			application.NewService(app),
		},
		Assets: application.AssetOptions{
			Handler: application.BundledAssetFileServer(assets),
		},
		Windows: application.WindowsOptions{
			WebviewUserDataPath: webviewDirIn(dataRoot),
		},
	})

	app.setApp(wailsApp)

	wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:             "Naraberu",
		Width:             1200,
		Height:            800,
		BackgroundColour:  application.RGBA{Red: 18, Green: 18, Blue: 24, Alpha: 255},
	})

	if err := wailsApp.Run(); err != nil {
		log.Fatal(err)
	}
}
