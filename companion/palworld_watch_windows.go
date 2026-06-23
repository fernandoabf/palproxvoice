//go:build windows

package main

import (
	"context"
	"strings"
	"time"
	"unsafe"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/sys/windows"
)

var palworldExes = []string{"palworld-win64-shipping.exe", "palworld-wingdk-shipping.exe"}

// processRunning: true se algum processo com um desses nomes (lowercase) existe.
func processRunning(names ...string) bool {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return false
	}
	defer windows.CloseHandle(snap)
	var e windows.ProcessEntry32
	e.Size = uint32(unsafe.Sizeof(e))
	if windows.Process32First(snap, &e) != nil {
		return false
	}
	for {
		n := strings.ToLower(windows.UTF16ToString(e.ExeFile[:]))
		for _, want := range names {
			if n == want {
				return true
			}
		}
		if windows.Process32Next(snap, &e) != nil {
			break
		}
	}
	return false
}

// startPalworldQuitWatch: o companion SAI quando o Palworld fecha (lifecycle colado
// ao jogo, via o watcher). So fecha se o Palworld chegou a rodar — abrir o companion
// manualmente (sem o jogo, p/ configurar) NAO o fecha sozinho.
func startPalworldQuitWatch(ctx context.Context) {
	go func() {
		seen := false
		t := time.NewTicker(3 * time.Second)
		defer t.Stop()
		for range t.C {
			if processRunning(palworldExes...) {
				seen = true
			} else if seen {
				runtime.Quit(ctx)
				return
			}
		}
	}()
}
