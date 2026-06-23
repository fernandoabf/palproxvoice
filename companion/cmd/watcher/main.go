//go:build windows

// palproxvoice-watcher: vigia minúsculo que roda na inicialização do Windows.
// Quando o Palworld abre, lança o companion (palproxvoice.exe -min); o companion
// sai sozinho quando o Palworld fecha. Resultado: o companion "abre e fecha junto
// com o Palworld". Sem janela (build com -H=windowsgui).
package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	palworldExes  = []string{"palworld-win64-shipping.exe", "palworld-wingdk-shipping.exe"}
	companionExe  = "palproxvoice.exe"
)

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

func main() {
	self, err := os.Executable()
	if err != nil {
		return
	}
	companion := filepath.Join(filepath.Dir(self), companionExe)
	for {
		// Palworld rodando E companion ainda nao -> sobe o companion (escondido).
		if processRunning(palworldExes...) && !processRunning(companionExe) {
			c := exec.Command(companion, "-min")
			c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
			_ = c.Start()
		}
		time.Sleep(4 * time.Second)
	}
}
