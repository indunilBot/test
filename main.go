package main

import (
	"embed"
	"log"
	"strings"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/logger"
	"github.com/wailsapp/wails/v2/pkg/options"
)

//go:embed all:frontend/dist
var assets embed.FS

// CustomLogger wraps the default logger to filter out runtime:ready errors
type CustomLogger struct {
	logger.Logger
}

func (c *CustomLogger) Error(message string) {
	// Filter out the harmless runtime:ready error
	if !strings.Contains(message, "runtime:ready") && !strings.Contains(message, "Unknown message from front end: runtime:ready") {
		c.Logger.Error(message)
	}
}

func main() {
	// Create an instance of the app structure
	app := NewApp()

	// Create custom logger
	customLogger := &CustomLogger{
		Logger: logger.NewDefaultLogger(),
	}

	// Create application with options
	err := wails.Run(&options.App{
		Title:            "GPaw Explorer",
		Width:            1024,
		Height:           768,
		Assets:           assets,
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Logger:           customLogger,
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}