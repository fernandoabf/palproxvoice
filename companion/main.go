package main

import (
	"context"
	"embed"
	"os"
	"path/filepath"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()

	// auto-start passa "-min" -> abre minimizado (escondido na barra de tarefas)
	startState := options.Normal
	for _, a := range os.Args[1:] {
		if a == "-min" || a == "--min" {
			startState = options.Minimised
		}
	}

	err := wails.Run(&options.App{
		Title:            "PalProxVoice",
		Width:            960,
		Height:           700,
		MinWidth:         640,
		MinHeight:        560,
		WindowStartState: startState,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}

// startup is called when the app boots. It stores the context for later
// EventsEmit calls and kicks off the position listener goroutine.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	go a.positionListener()
}

// shutdown is called when the app closes.
func (a *App) shutdown(ctx context.Context) {
	// nothing to clean up for now
}

// positionListener reads C:\Users\Public\palproxvoice_pos.txt every 50ms and
// emits the "pos" event to the frontend whenever the line changes. Caminho FIXO
// (mesma pasta pra todo processo) — %TEMP% varia entre o jogo e o companion.
func (a *App) positionListener() {
	pub := os.Getenv("PUBLIC")
	if pub == "" {
		pub = `C:\Users\Public`
	}
	posFile := filepath.Join(pub, "palproxvoice_pos.txt")

	var lastPos string
	var wasFresh bool
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		info, err := os.Stat(posFile)
		// "fresco" = o mod escreveu nos ultimos 2s (no mundo ele escreve a 20Hz).
		fresh := err == nil && time.Since(info.ModTime()) < 2*time.Second
		if fresh {
			if data, rerr := os.ReadFile(posFile); rerr == nil {
				if pos := string(data); pos != lastPos {
					lastPos = pos
					runtime.EventsEmit(a.ctx, "pos", pos)
				}
			}
			wasFresh = true
		} else if wasFresh {
			// posicao parou de atualizar -> saiu do mundo/servidor
			wasFresh = false
			lastPos = ""
			runtime.EventsEmit(a.ctx, "posLost")
		}
	}
}
