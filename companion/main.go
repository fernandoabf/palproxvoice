package main

import (
	"context"
	"embed"
	"os"
	"path/filepath"
	"strings"
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
	// O "chrome-wide echo cancellation" do WebView2 mantem um stream de referencia
	// de COMUNICACAO ativo, e isso faz o Windows ducar (abaixar) todos os outros
	// sons enquanto a voz roda. Desliga essa feature (junto com echoCancellation:false
	// no getUserMedia). WebView2 le essa env var na inicializacao.
	// O mic agora e capturado NATIVO (WASAPI/go-wca, AudioCategory_Other) fora do
	// WebView2 — o getUserMedia do Chromium abria a captura como AudioCategory_
	// Communications, que poe o codec em "modo voz" e degrada todo o audio do
	// sistema. Mantemos ChromeWideEchoCancellation off por garantia.
	os.Setenv("WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS", "--disable-features=ChromeWideEchoCancellation")

	app := NewApp()

	// auto-start passa "-min" -> inicia OCULTO (sem aba na barra de tarefas);
	// aparece como overlay so quando voce entra no jogo.
	startHidden := false
	for _, a := range os.Args[1:] {
		if a == "-min" || a == "--min" {
			startHidden = true
		}
	}

	err := wails.Run(&options.App{
		Title:       "PalProxVoice",
		Width:       960,
		Height:      700,
		MinWidth:    210,
		MinHeight:   44,
		Frameless:   true, // sem barra de titulo -> HUD overlay limpo
		StartHidden: startHidden,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		OnDomReady:       app.domReady,
		OnShutdown:       app.shutdown,
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId:               "palproxvoice-companion",
			OnSecondInstanceLaunch: app.onSecondInstance,
		},
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
	startOverlayWatch()          // mantem o overlay vivo contra "mostrar area de trabalho"
	go fixAudioDucking()         // desliga o ducking de comunicacao no HKCU do usuario
	startPalworldQuitWatch(ctx)  // sai junto com o Palworld (lifecycle colado ao jogo)
	startChannelHotkey(ctx)      // Alt+V global cicla o canal de voz in-game
	go StartServerIPWatchETW(ctx) // ETW: IP do server em tempo real (so com admin; degrada sem)
}

// domReady roda com a janela ja criada -> vira "tool window" (some da barra de
// tarefas e do Alt-Tab, mas segue visivel como overlay).
func (a *App) domReady(ctx context.Context) {
	go applyToolWindow()
}

// ApplyOverlayStyle: o front chama apos cada WindowShow pra garantir que a janela
// fique fora da barra de tarefas (e ai o refresh da taskbar pega).
func (a *App) ApplyOverlayStyle() { applyToolWindow() }

// SetOverlayMode: o front avisa quando entra/sai do HUD; em overlay o watchdog
// reverte minimizar-tudo e reforca o always-on-top.
func (a *App) SetOverlayMode(on bool) { setOverlay(on) }

// FixAudioDucking: o front chama ANTES de conectar pra reaplicar o "nao ducar"
// no HKCU do usuario, pra a sessao de comunicacao do Windows ler o valor fresco.
func (a *App) FixAudioDucking() { fixAudioDucking() }

// onSecondInstance: abrir o .exe de novo (atalho/instalador) traz a janela de
// config de volta, em vez de subir uma 2a instancia.
func (a *App) onSecondInstance(_ options.SecondInstanceData) {
	runtime.WindowShow(a.ctx)
	runtime.WindowUnminimise(a.ctx)
	runtime.EventsEmit(a.ctx, "showConfig")
}

// shutdown is called when the app closes.
func (a *App) shutdown(ctx context.Context) {
	// nothing to clean up for now
}

// PlayerID le o FGuid do player que o mod escreve em palproxvoice_id.txt e o
// devolve pro frontend mandar no auth (campo "user"). O servidor de voz usa isso
// pra correlacionar o peer com a REST do Palworld (anti-spoof) — cobre o caso de
// varios jogadores no mesmo IP (mesma casa). "" se o arquivo ainda nao existe
// (ainda nao entrou no mundo) -> o servidor cai pra correlacao por IP+proximidade.
func (a *App) PlayerID() string {
	pub := os.Getenv("PUBLIC")
	if pub == "" {
		pub = `C:\Users\Public`
	}
	data, err := os.ReadFile(filepath.Join(pub, "palproxvoice_id.txt"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// PlayerGuild le a guild do player que o mod escreve em palproxvoice_guild.txt (auto).
// O companion usa isso pra voz de GUILD; "" -> cai no codigo manual de guild.
func (a *App) PlayerGuild() string {
	pub := os.Getenv("PUBLIC")
	if pub == "" {
		pub = `C:\Users\Public`
	}
	data, err := os.ReadFile(filepath.Join(pub, "palproxvoice_guild.txt"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
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
			if !wasFresh { // acabou de (re)entrar num mundo -> traz o HUD de volta, mesmo se minimizado
				runtime.EventsEmit(a.ctx, "gameEnter")
			}
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
