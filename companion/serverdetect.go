package main

import (
	"bytes"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Deteccao do IP do servidor de jogo, em 3 fontes (da mais precisa pra menos):
//   1. GameServerIPLive()     — arquivo que o mod C++ escreve com o IP da sessao ATUAL (LowLevelGetRemoteAddress)
//   2. GameServerIPFromSave() — PalOptionSaveGame: pega Direct Connect E join pela lista do Steam
//   3. GameServerIP()         — GameUserSettings.ini (app.go): so Direct Connect
//
// DetectGameServerIP() tenta as tres em ordem. O outro chat (que monta o companion)
// chama isso no fluxo de auto-connect; aqui nao mexo no main.go.

// serverFile e onde o mod C++ grava o IP da sessao atual (mesma pasta do pos do mod Lua).
const serverFile = `C:\Users\Public\palproxvoice_server.txt`

// GameServerIPLive le o IP da sessao atual escrito pelo mod C++ (PalProxVoiceLive).
// Formato do arquivo: "ip:porta" ou so "ip". Retorna so o host. "" se nao existir.
func (a *App) GameServerIPLive() string {
	data, err := os.ReadFile(serverFile)
	if err != nil {
		return ""
	}
	return hostOnly(strings.TrimSpace(string(data)))
}

// GameServerIPFromSave le o endereco do servidor mais jogado do PalOptionSaveGame
// (HistoryServerWorld[].ServerAddress). Cobre Direct Connect E join pela lista,
// ao contrario do GameUserSettings.ini. "" se nao achar.
func (a *App) GameServerIPFromSave() string {
	for _, f := range palOptionSaveFiles() {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		if ip := latestServerAddress(data); ip != "" {
			return ip
		}
	}
	return ""
}

// DetectGameServerIP: live (mod C++) -> save -> GameUserSettings.ini. "" se nada.
func (a *App) DetectGameServerIP() string {
	if ip := a.GameServerIPLive(); ip != "" {
		return ip
	}
	if ip := a.GameServerIPFromSave(); ip != "" {
		return ip
	}
	return a.GameServerIP()
}

func hostOnly(v string) string {
	if v == "" {
		return ""
	}
	if i := strings.LastIndex(v, ":"); i > 0 { // tira a porta
		v = v[:i]
	}
	return v
}

// palOptionSaveFiles lista os candidatos a save de opcoes (GDK/Xbox + Steam).
// GDK guarda em blobs WGS com nomes GUID; Steam em Saved/SaveGames. So leitura
// barata: arquivos pequenos (o save de opcoes tem ~40 KB; Level.sav e grande -> pulado).
func palOptionSaveFiles() []string {
	local := os.Getenv("LOCALAPPDATA")
	if local == "" {
		return nil
	}
	var roots []string
	// GDK/Xbox: %LOCALAPPDATA%\Packages\PocketpairInc.Palworld_*\SystemAppData\wgs
	gdk, _ := filepath.Glob(filepath.Join(local, "Packages", "PocketpairInc.Palworld_*", "SystemAppData", "wgs"))
	roots = append(roots, gdk...)
	// Steam: %LOCALAPPDATA%\Pal\Saved\SaveGames
	roots = append(roots, filepath.Join(local, "Pal", "Saved", "SaveGames"))

	var files []string
	for _, root := range roots {
		filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			info, e := d.Info()
			if e != nil || info.Size() > 2<<20 { // pula arquivos grandes (mundos)
				return nil
			}
			files = append(files, p)
			return nil
		})
	}
	return files
}

var ipv4re = regexp.MustCompile(`(?:\d{1,3}\.){3}\d{1,3}`)

// latestServerAddress acha o ServerAddress mais frequente no blob GVAS.
// Heuristica: cada entrada tem "...ServerAddress <IP> ServerPort..."; pego o IP
// na janela entre os dois marcadores e valido octetos (rejeita VersionString tipo
// 0.7.3.904). Mais frequente vence.
// ponytail: "mais frequente", nao "mais recente" — a ordem do historico nao e
// garantida. Pra precisao real-time use o mod C++ (GameServerIPLive). Hostnames
// (nao-IP) nao sao cobertos aqui; se precisar, parsear o StrProperty do GVAS.
func latestServerAddress(data []byte) string {
	marker := []byte("ServerAddress")
	end := []byte("ServerPort")
	counts := map[string]int{}
	for i := 0; ; {
		j := bytes.Index(data[i:], marker)
		if j < 0 {
			break
		}
		start := i + j + len(marker)
		hi := start + 80
		if hi > len(data) {
			hi = len(data)
		}
		win := data[start:hi]
		if k := bytes.Index(win, end); k >= 0 {
			win = win[:k]
		}
		for _, m := range ipv4re.FindAll(win, -1) {
			ip := string(m)
			if p := net.ParseIP(ip); p != nil && p.To4() != nil {
				counts[ip]++
			}
		}
		i = start
	}
	best, bn := "", 0
	for ip, n := range counts {
		if n > bn {
			best, bn = ip, n
		}
	}
	return best
}
