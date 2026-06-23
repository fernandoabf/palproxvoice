//go:build windows

package main

import (
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

const (
	gwlExStyle       = ^uintptr(19) // GWL_EXSTYLE (-20) sem estourar uintptr
	wsExToolWindow   = 0x00000080
	wsExAppWindow    = 0x00040000
	swHide           = 0
	swShow           = 5
	swShowNoActivate = 4
	hwndTopmost      = ^uintptr(0) // HWND_TOPMOST (-1)
	swpNoSize        = 0x0001
	swpNoMove        = 0x0002
	swpNoActivate    = 0x0010
	swpShowWindow    = 0x0040
)

var (
	user32                       = syscall.NewLazyDLL("user32.dll")
	kernel32                     = syscall.NewLazyDLL("kernel32.dll")
	procEnumWindows              = user32.NewProc("EnumWindows")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	procGetWindowLongPtr         = user32.NewProc("GetWindowLongPtrW")
	procSetWindowLongPtr         = user32.NewProc("SetWindowLongPtrW")
	procShowWindow               = user32.NewProc("ShowWindow")
	procIsWindowVisible          = user32.NewProc("IsWindowVisible")
	procIsIconic                 = user32.NewProc("IsIconic")
	procSetWindowPos             = user32.NewProc("SetWindowPos")
	procGetWindowTextW           = user32.NewProc("GetWindowTextW")
	procGetCurrentProcessId      = kernel32.NewProc("GetCurrentProcessId")
)

var (
	toolMu      sync.Mutex // serializa o uso das globais de EnumWindows
	enumPID     uint32
	enumWindows []uintptr
	overlayOn   int32 // atomic: 1 = estamos em modo overlay (HUD)
)

func enumProc(hwnd, _ uintptr) uintptr {
	var pid uint32
	procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	if pid == enumPID {
		enumWindows = append(enumWindows, hwnd)
	}
	return 1 // continua
}

// enumCB criado sob demanda (NewCallback na carga do pacote pode derrubar o boot).
var (
	enumCB     uintptr
	enumCBOnce sync.Once
)

func ensureEnumCB() uintptr {
	enumCBOnce.Do(func() { enumCB = syscall.NewCallback(enumProc) })
	return enumCB
}

func windowTitle(h uintptr) string {
	buf := make([]uint16, 64)
	n, _, _ := procGetWindowTextW.Call(h, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return syscall.UTF16ToString(buf[:n])
}

// mainWindow acha a janela principal DO NOSSO processo (nao depende do titulo,
// que a janela frameless pode nao expor): prefere a visivel; senao a com titulo
// "PalProxVoice"; senao a primeira top-level.
func mainWindow() uintptr {
	toolMu.Lock()
	defer toolMu.Unlock()
	pid, _, _ := procGetCurrentProcessId.Call()
	enumPID = uint32(pid)
	enumWindows = nil
	procEnumWindows.Call(ensureEnumCB(), 0)
	var first, titled uintptr
	for _, h := range enumWindows {
		if first == 0 {
			first = h
		}
		if v, _, _ := procIsWindowVisible.Call(h); v != 0 {
			return h
		}
		if titled == 0 && windowTitle(h) == "PalProxVoice" {
			titled = h
		}
	}
	if titled != 0 {
		return titled
	}
	return first
}

// applyToolWindow ajusta o "chrome" da janela conforme o modo (overlayOn):
//   overlay (compacto) -> tool window: FORA da barra de tarefas e do Alt-Tab.
//   cheio              -> app window:  NA barra de tarefas e no Alt-Tab. Sem isso,
//                          ao dar Alt-Tab a janela ia pra tras e nao havia como
//                          traze-la de volta (so relogando).
// Idempotente — o front chama de novo apos cada troca de modo / WindowShow, que
// e quando o refresh da taskbar pega de verdade.
func applyToolWindow() {
	overlay := atomic.LoadInt32(&overlayOn) == 1
	for i := 0; i < 20; i++ {
		if h := mainWindow(); h != 0 {
			ex, _, _ := procGetWindowLongPtr.Call(h, gwlExStyle)
			var want uintptr
			if overlay {
				want = (ex | wsExToolWindow) &^ wsExAppWindow
			} else {
				want = (ex | wsExAppWindow) &^ wsExToolWindow
			}
			if want != ex {
				visible, _, _ := procIsWindowVisible.Call(h)
				if visible != 0 { // a taskbar so re-le o estilo escondendo e mostrando
					procShowWindow.Call(h, swHide)
				}
				procSetWindowLongPtr.Call(h, gwlExStyle, want)
				if visible != 0 {
					procShowWindow.Call(h, swShow)
				}
			}
			return
		}
		time.Sleep(150 * time.Millisecond)
	}
}

func setOverlay(on bool) {
	if on {
		atomic.StoreInt32(&overlayOn, 1)
	} else {
		atomic.StoreInt32(&overlayOn, 0)
	}
}

// startOverlayWatch sobe o watchdog: em modo overlay, "Mostrar area de trabalho"
// (Win+D) / minimizar-tudo tira a janela — aqui revertemos na hora e reforcamos
// o always-on-top, sem roubar o foco do jogo.
func startOverlayWatch() { go overlayWatch() }

func overlayWatch() {
	for {
		time.Sleep(250 * time.Millisecond)
		if atomic.LoadInt32(&overlayOn) == 0 {
			continue
		}
		h := mainWindow()
		if h == 0 {
			continue
		}
		// "Mostrar area de trabalho" (Win+D) MINIMIZA janelas normais, mas a tool
		// window (sem aba) ele ESCONDE -> tratamos os dois e restauramos sem ativar.
		iconic, _, _ := procIsIconic.Call(h)
		visible, _, _ := procIsWindowVisible.Call(h)
		if iconic != 0 || visible == 0 {
			procShowWindow.Call(h, swShowNoActivate)
		}
		procSetWindowPos.Call(h, hwndTopmost, 0, 0, 0, 0, swpNoMove|swpNoSize|swpNoActivate|swpShowWindow)
	}
}
