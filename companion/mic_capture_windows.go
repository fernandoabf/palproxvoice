//go:build windows

package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
	"unsafe"

	ole "github.com/go-ole/go-ole"
	"github.com/gorilla/websocket"
	"github.com/moutend/go-wca/pkg/wca"
)

// micLog: diagnostico da captura -> %TEMP%\ppv_mic.log (lido fora do app).
func micLog(format string, a ...any) {
	f, err := os.OpenFile(filepath.Join(os.TempDir(), "ppv_mic.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s "+format+"\n", append([]any{time.Now().Format("15:04:05")}, a...)...)
}

// Captura do mic NATIVA via WASAPI (go-wca) com AudioCategory_Other — NAO
// Communications. Isso evita que o codec entre em "modo voz" (que degrada o
// resto do audio do sistema), problema inerente ao getUserMedia do WebView2.
// O PCM (mono float32 @48k) sai por um WebSocket local; o front pluga num
// AudioWorklet -> MediaStreamDestination -> RTCPeerConnection, sem getUserMedia.

const (
	captureSilentFlag = 0x2
	captureRate       = 48000
	captureFrame      = 480 // 10ms de mono @48k
	deviceStateActive = 0x1 // DEVICE_STATE_ACTIVE
)

// MicDevice = um device de captura WASAPI (id de endpoint + nome amigavel).
type MicDevice struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

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

// ListMicDevices enumera os devices de captura WASAPI ativos (id + nome amigavel),
// pro front popular o seletor de microfone. Nao precisa de permissao (e' nativo).
func (a *App) ListMicDevices() []MicDevice {
	out := []MicDevice{}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED) != nil {
		return out
	}
	defer ole.CoUninitialize()

	var mmde *wca.IMMDeviceEnumerator
	if wca.CoCreateInstance(wca.CLSID_MMDeviceEnumerator, 0, wca.CLSCTX_ALL, wca.IID_IMMDeviceEnumerator, &mmde) != nil {
		return out
	}
	defer mmde.Release()

	var coll *wca.IMMDeviceCollection
	if mmde.EnumAudioEndpoints(wca.ECapture, deviceStateActive, &coll) != nil {
		return out
	}
	defer coll.Release()

	var n uint32
	coll.GetCount(&n)
	for i := uint32(0); i < n; i++ {
		var d *wca.IMMDevice
		if coll.Item(i, &d) != nil {
			continue
		}
		var id string
		d.GetId(&id)
		name := id
		var ps *wca.IPropertyStore
		if d.OpenPropertyStore(wca.STGM_READ, &ps) == nil {
			var pv wca.PROPVARIANT
			if ps.GetValue(&wca.PKEY_Device_FriendlyName, &pv) == nil {
				if s := pv.String(); s != "" {
					name = s
				}
			}
			ps.Release()
		}
		out = append(out, MicDevice{ID: id, Name: name})
		d.Release()
	}
	return out
}

// openCaptureDevice devolve o device de captura pedido (por id) ou o padrao
// (role Console, NAO Communications) se id vazio / nao encontrado.
func openCaptureDevice(mmde *wca.IMMDeviceEnumerator, deviceID string) *wca.IMMDevice {
	if deviceID != "" {
		var coll *wca.IMMDeviceCollection
		if mmde.EnumAudioEndpoints(wca.ECapture, deviceStateActive, &coll) == nil {
			defer coll.Release()
			var n uint32
			coll.GetCount(&n)
			for i := uint32(0); i < n; i++ {
				var d *wca.IMMDevice
				if coll.Item(i, &d) != nil {
					continue
				}
				var id string
				if d.GetId(&id) == nil && id == deviceID {
					return d
				}
				d.Release()
			}
		}
	}
	var d *wca.IMMDevice
	if mmde.GetDefaultAudioEndpoint(wca.ECapture, wca.EConsole, &d) != nil {
		return nil
	}
	return d
}

// StartMicCapture inicia a captura nativa + um WS local que entrega PCM mono
// float32 @48k em frames de 10ms. deviceID vazio = device padrao do Windows.
// Retorna "ws://127.0.0.1:PORT/pcm" ou "erro: ...".
func (a *App) StartMicCapture(deviceID string) string {
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
	go captureLoop(micCap.stop, deviceID)
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
func captureLoop(stop chan struct{}, deviceID string) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	micLog("captureLoop start deviceID=%q", deviceID)
	if err := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED); err != nil {
		micLog("CoInitializeEx FALHOU: %v", err)
		return
	}
	defer ole.CoUninitialize()

	var mmde *wca.IMMDeviceEnumerator
	if hr := wca.CoCreateInstance(wca.CLSID_MMDeviceEnumerator, 0, wca.CLSCTX_ALL, wca.IID_IMMDeviceEnumerator, &mmde); hr != nil {
		micLog("CoCreateInstance FALHOU: %v", hr)
		return
	}
	defer mmde.Release()

	mmd := openCaptureDevice(mmde, deviceID)
	if mmd == nil {
		micLog("openCaptureDevice retornou nil (device nao achado e default falhou)")
		return
	}
	defer mmd.Release()

	var ac2 *wca.IAudioClient2
	if hr := mmd.Activate(wca.IID_IAudioClient2, wca.CLSCTX_ALL, nil, &ac2); hr != nil {
		micLog("Activate IAudioClient2 FALHOU: %v", hr)
		return
	}
	defer ac2.Release()

	// Best-effort: alguns drivers (devices virtuais tipo VoiceMeeter) rejeitam
	// SetClientProperties. Nao e' fatal — a categoria PADRAO de um stream WASAPI
	// ja e' Other (nao-Communications), que e' justamente o que queremos. Entao
	// se falhar, seguimos assim mesmo.
	props := &wca.AudioClientProperties{
		CbSize:                uint32(unsafe.Sizeof(wca.AudioClientProperties{})),
		AUDIO_STREAM_CATEGORY: wca.AudioCategory_Other,
	}
	if hr := ac2.SetClientProperties(props); hr != nil {
		micLog("SetClientProperties falhou (segue mesmo assim; categoria padrao ja e' Other): %v", hr)
	}

	var wfx *wca.WAVEFORMATEX
	if hr := ac2.GetMixFormat(&wfx); hr != nil {
		micLog("GetMixFormat FALHOU: %v", hr)
		return
	}
	ch := int(wfx.NChannels)
	isFloat := wfx.WBitsPerSample == 32 // mix format moderno = float32
	micLog("formato nativo: ch=%d bits=%d rate=%d blockAlign=%d tag=0x%X isFloat=%t",
		wfx.NChannels, wfx.WBitsPerSample, wfx.NSamplesPerSec, wfx.NBlockAlign, wfx.WFormatTag, isFloat)

	// FORCA 48kHz: o worklet/AudioContext e o RNNoise sao fixos em 48k (frame 480).
	// Se o device estiver em outra taxa (ex.: Discord poe o mic em modo comunicacao),
	// capturar na taxa nativa sairia "robotizado" (pitch/drift). Pedimos 48k e deixamos
	// o WASAPI reamostrar (AUTOCONVERTPCM). Fallback p/ taxa nativa se o driver recusar.
	const autoConv = uint32(0x80000000 | 0x08000000) // AUTOCONVERTPCM | SRC_DEFAULT_QUALITY
	var hnsBuf wca.REFERENCE_TIME = 10_000_000        // 1s
	if wfx.NSamplesPerSec != 48000 {
		orig := *wfx
		wfx.NSamplesPerSec = 48000
		wfx.NAvgBytesPerSec = 48000 * uint32(wfx.NBlockAlign)
		if hr := ac2.Initialize(wca.AUDCLNT_SHAREMODE_SHARED, autoConv, hnsBuf, 0, wfx, nil); hr != nil {
			micLog("Initialize 48k+autoconvert falhou (%v); usando taxa nativa", hr)
			*wfx = orig
			if hr2 := ac2.Initialize(wca.AUDCLNT_SHAREMODE_SHARED, 0, hnsBuf, 0, wfx, nil); hr2 != nil {
				micLog("Initialize FALHOU: %v", hr2)
				return
			}
		} else {
			micLog("captura forcada p/ 48kHz (autoconvert)")
		}
	} else if hr := ac2.Initialize(wca.AUDCLNT_SHAREMODE_SHARED, 0, hnsBuf, 0, wfx, nil); hr != nil {
		micLog("Initialize FALHOU: %v", hr)
		return
	}

	var acc *wca.IAudioCaptureClient
	if hr := ac2.GetService(wca.IID_IAudioCaptureClient, &acc); hr != nil {
		micLog("GetService FALHOU: %v", hr)
		return
	}
	defer acc.Release()

	if hr := ac2.Start(); hr != nil {
		micLog("Start FALHOU: %v", hr)
		return
	}
	defer ac2.Stop()
	micLog("captura iniciada OK")

	mono := make([]float32, 0, captureFrame*4)
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			micLog("stop recebido")
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
