//go:build windows

package main

import (
	"context"
	"runtime"
	"syscall"
	"unsafe"

	wr "github.com/wailsapp/wails/v2/pkg/runtime"
)

// Hotkey GLOBAL pra ciclar o canal de voz (proximidade -> guild -> global) IN-GAME,
// sem focar a janela do companion. RegisterHotKey "rouba" so a combinacao Alt+V; o V
// puro do jogo segue normal. NAO precisa de admin.
var (
	user32hk           = syscall.NewLazyDLL("user32.dll")
	procRegisterHotKey = user32hk.NewProc("RegisterHotKey")
	procGetMessageW    = user32hk.NewProc("GetMessageW")
)

const (
	hkModAlt      = 0x0001
	hkModNoRepeat = 0x4000
	hkWMHotkey    = 0x0312
	hkVKV         = 0x56 // tecla 'V'
	hkID          = 1
)

// layout do MSG do Win32 (x86_64; o padding antes de wParam o Go insere sozinho).
type hkMSG struct {
	hwnd    uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      struct{ x, y int32 }
}

// startChannelHotkey registra Alt+V e emite "cyclechannel" pro frontend a cada toque.
// RegisterHotKey + GetMessage TEM que rodar na MESMA thread -> LockOSThread.
func startChannelHotkey(ctx context.Context) {
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		if r, _, _ := procRegisterHotKey.Call(0, hkID, hkModAlt|hkModNoRepeat, hkVKV); r == 0 {
			return // ja em uso / falhou -> segue sem hotkey, sem crashar
		}
		var m hkMSG
		for {
			ret, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
			if int32(ret) <= 0 { // 0 = WM_QUIT, -1 = erro
				return
			}
			if m.message == hkWMHotkey {
				wr.EventsEmit(ctx, "cyclechannel")
			}
		}
	}()
}
