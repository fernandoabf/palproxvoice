//go:build windows

package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"net"
	"net/http"
	"runtime"
	"sync"
	"time"
	"unsafe"

	ole "github.com/go-ole/go-ole"
	"github.com/gorilla/websocket"
	"github.com/moutend/go-wca/pkg/wca"
)

// Captura do mic NATIVA via WASAPI (go-wca) com AudioCategory_Other — NAO
// Communications. Isso evita que o codec entre em "modo voz" (que degrada o
// resto do audio do sistema), problema inerente ao getUserMedia do WebView2.
// O PCM (mono float32 @48k) sai por um WebSocket local; o front pluga num
// AudioWorklet -> MediaStreamDestination -> RTCPeerConnection, sem getUserMedia.

const (
	captureSilentFlag = 0x2
	captureRate       = 48000
	captureFrame      = 480 // 10ms de mono @48k
)

type micCapture struct {
	mu      sync.Mutex
	running bool
	stop    chan struct{}
	url     string
	srv     *http.Server

	connMu sync.Mutex
	conn   *websocket.Conn
}

var micCap = &micCapture{}

var micUpgrader = websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}

// StartMicCapture inicia a captura nativa + um WS local que entrega PCM mono
// float32 @48k em frames de 10ms. Retorna "ws://127.0.0.1:PORT/pcm" ou "erro: ...".
func (a *App) StartMicCapture() string {
	micCap.mu.Lock()
	defer micCap.mu.Unlock()
	if micCap.running {
		return micCap.url
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "erro: listen: " + err.Error()
	}
	port := ln.Addr().(*net.TCPAddr).Port

	mux := http.NewServeMux()
	mux.HandleFunc("/pcm", func(w http.ResponseWriter, r *http.Request) {
		c, err := micUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		micCap.connMu.Lock()
		if micCap.conn != nil {
			micCap.conn.Close()
		}
		micCap.conn = c
		micCap.connMu.Unlock()
		// drena leituras pra detectar o fechamento do lado JS
		go func() {
			for {
				if _, _, err := c.ReadMessage(); err != nil {
					return
				}
			}
		}()
	})

	micCap.srv = &http.Server{Handler: mux}
	go micCap.srv.Serve(ln)

	micCap.stop = make(chan struct{})
	micCap.running = true
	micCap.url = fmt.Sprintf("ws://127.0.0.1:%d/pcm", port)
	go captureLoop(micCap.stop)
	return micCap.url
}

// StopMicCapture para a captura e fecha o WS (libera o mic).
func (a *App) StopMicCapture() {
	micCap.mu.Lock()
	defer micCap.mu.Unlock()
	if !micCap.running {
		return
	}
	close(micCap.stop)
	micCap.running = false
	micCap.connMu.Lock()
	if micCap.conn != nil {
		micCap.conn.Close()
		micCap.conn = nil
	}
	micCap.connMu.Unlock()
	if micCap.srv != nil {
		micCap.srv.Close()
		micCap.srv = nil
	}
}

func sendFrame(mono []float32) {
	micCap.connMu.Lock()
	c := micCap.conn
	micCap.connMu.Unlock()
	if c == nil {
		return
	}
	b := make([]byte, len(mono)*4)
	for i, s := range mono {
		binary.LittleEndian.PutUint32(b[i*4:], math.Float32bits(s))
	}
	_ = c.WriteMessage(websocket.BinaryMessage, b)
}

// captureLoop abre o device de captura padrao (role Console, NAO Communications)
// e empurra frames de PCM mono pro WS ate receber stop.
func captureLoop(stop chan struct{}) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if err := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED); err != nil {
		return
	}
	defer ole.CoUninitialize()

	var mmde *wca.IMMDeviceEnumerator
	if wca.CoCreateInstance(wca.CLSID_MMDeviceEnumerator, 0, wca.CLSCTX_ALL, wca.IID_IMMDeviceEnumerator, &mmde) != nil {
		return
	}
	defer mmde.Release()

	var mmd *wca.IMMDevice
	if mmde.GetDefaultAudioEndpoint(wca.ECapture, wca.EConsole, &mmd) != nil {
		return
	}
	defer mmd.Release()

	var ac2 *wca.IAudioClient2
	if mmd.Activate(wca.IID_IAudioClient2, wca.CLSCTX_ALL, nil, &ac2) != nil {
		return
	}
	defer ac2.Release()

	props := &wca.AudioClientProperties{
		CbSize:                uint32(unsafe.Sizeof(wca.AudioClientProperties{})),
		AUDIO_STREAM_CATEGORY: wca.AudioCategory_Other,
	}
	if ac2.SetClientProperties(props) != nil {
		return
	}

	var wfx *wca.WAVEFORMATEX
	if ac2.GetMixFormat(&wfx) != nil {
		return
	}
	ch := int(wfx.NChannels)
	isFloat := wfx.WBitsPerSample == 32 // mix format moderno = float32

	var hnsBuf wca.REFERENCE_TIME = 10_000_000 // 1s
	if ac2.Initialize(wca.AUDCLNT_SHAREMODE_SHARED, 0, hnsBuf, 0, wfx, nil) != nil {
		return
	}

	var acc *wca.IAudioCaptureClient
	if ac2.GetService(wca.IID_IAudioCaptureClient, &acc) != nil {
		return
	}
	defer acc.Release()

	if ac2.Start() != nil {
		return
	}
	defer ac2.Stop()

	mono := make([]float32, 0, captureFrame*4)
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
		}
		var packet uint32
		if acc.GetNextPacketSize(&packet) != nil {
			return
		}
		for packet > 0 {
			var data *byte
			var frames, flags uint32
			if acc.GetBuffer(&data, &frames, &flags, nil, nil) != nil {
				return
			}
			if frames > 0 && data != nil && flags&captureSilentFlag == 0 && isFloat {
				src := unsafe.Slice((*float32)(unsafe.Pointer(data)), int(frames)*ch)
				for i := 0; i < int(frames); i++ {
					var s float32
					if ch >= 2 {
						s = (src[i*ch] + src[i*ch+1]) * 0.5
					} else {
						s = src[i*ch]
					}
					mono = append(mono, s)
				}
			} else if frames > 0 {
				// silencio ou formato inesperado -> zeros (mantem o ritmo)
				for i := 0; i < int(frames); i++ {
					mono = append(mono, 0)
				}
			}
			acc.ReleaseBuffer(frames)
			if acc.GetNextPacketSize(&packet) != nil {
				return
			}
		}
		for len(mono) >= captureFrame {
			sendFrame(mono[:captureFrame])
			mono = mono[captureFrame:]
		}
	}
}
