// PalProxVoice — servidor de voz (SFU + sinalizacao + relay de posicao)
//
// SEM SALA: todos os conectados sao um pool unico. Uma senha compartilhada
// (VOICE_PASSWORD) gateia a entrada. O audio de cada um e repassado pra todos
// os outros (SFU); a POSICAO (que vem do mod M1, via o app companion) tambem e
// repassada, pra cada cliente espacializar localmente (M3).
//
// Cada track de audio sai com StreamID = id do peer, pro cliente correlacionar
// "este audio" <-> "esta posicao".
package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v4"
)

// ---------- estado global (pool unico, sem sala) ----------

var (
	listLock    sync.Mutex
	peers       []*peerState
	trackLocals = map[string]*webrtc.TrackLocalStaticRTP{}

	api        *webrtc.API
	password   string
	serverName string // nome do servidor (env), mandado pro cliente
	voiceRange string // alcance recomendado (env), mandado pro cliente

	upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
)

type peerState struct {
	id   string
	conn *threadSafeWriter
	pc   *webrtc.PeerConnection

	// anti-spoof (V1.5): identidade + estado de reconciliacao com a REST
	ip       string  // IP publico do cliente (respeita X-Forwarded-For)
	userID   string  // userId do Palworld, se o cliente mandou (cobre mesmo-IP)
	// canais de voz: guild (auto do mod / codigo manual) + canal que o peer fala AGORA.
	guild   string // grupo de voz de guild (mesmo guild = se ouvem no canal guild)
	channel string // canal atual deste peer: "proximity" | "guild" | "global"
	lastX    float64 // ultima posicao considerada valida
	lastY    float64
	hasLast  bool
	divergeN int // polls seguidos divergindo da REST (verify)
}

type wsMsg struct {
	Event string `json:"event"`
	Data  string `json:"data"`
	ID    string `json:"id,omitempty"`
	User  string `json:"user,omitempty"` // userId do Palworld no auth (anti-spoof)
}

// ---------- main ----------

func main() {
	password = os.Getenv("VOICE_PASSWORD")
	publicIP := os.Getenv("PUBLIC_IP") // IP publico do host (pro audio WebRTC)
	if strings.EqualFold(strings.TrimSpace(publicIP), "auto") {
		publicIP = detectPublicIP() // descobre sozinho (nao precisa fixar na mao)
	}
	serverName = envOr("SERVER_NAME", "PalProxVoice")
	voiceRange = envOr("VOICE_RANGE", "50") // alcance recomendado (m)
	wsPort := envOr("WS_PORT", "8080")

	// Uma faixa fixa de portas UDP pro audio + anuncia o IP publico (sem TURN).
	se := webrtc.SettingEngine{}
	se.SetEphemeralUDPPortRange(50000, 50010)
	if publicIP != "" {
		se.SetNAT1To1IPs([]string{publicIP}, webrtc.ICECandidateTypeHost)
	}
	api = webrtc.NewAPI(webrtc.WithSettingEngine(se))

	initAntispoof() // V1.5: liga o poll da REST se PPV_AUTH_MODE=verify|strict
	initV2()        // V2 (experimental): feed do mod server-side se PPV_PLAYERS_FILE setado

	http.HandleFunc("/ws", websocketHandler)

	// renegociacao periodica de seguranca
	go func() {
		for range time.NewTicker(3 * time.Second).C {
			signalPeerConnections()
		}
	}()

	log.Printf("PalProxVoice: ws=:%s  media-udp=50000-50010  publicIP=%q", wsPort, publicIP)
	log.Fatal(http.ListenAndServe(":"+wsPort, nil))
}

// detectPublicIP descobre o IP publico do host (PUBLIC_IP=auto). E' o endereco que o
// WebRTC anuncia como candidato ICE -> o audio UDP vai DIRETO nesse IP (nao da pra ser
// nome de container: o cliente remoto esta fora do Docker). "" se nao conseguir.
func detectPublicIP() string {
	client := &http.Client{Timeout: 5 * time.Second}
	for _, url := range []string{"https://api.ipify.org", "https://ifconfig.me/ip", "https://icanhazip.com"} {
		resp, err := client.Get(url)
		if err != nil {
			continue
		}
		b, err := io.ReadAll(io.LimitReader(resp.Body, 64))
		resp.Body.Close()
		if err != nil {
			continue
		}
		ip := strings.TrimSpace(string(b))
		if net.ParseIP(ip) != nil {
			log.Printf("PUBLIC_IP=auto -> detectado %s", ip)
			return ip
		}
	}
	log.Printf("PUBLIC_IP=auto -> nao consegui detectar; seguindo sem NAT1To1 (audio pode falhar de fora)")
	return ""
}

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func randID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// ---------- websocket / peer ----------

func websocketHandler(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade:", err)
		return
	}
	conn := &threadSafeWriter{Conn: c}
	defer conn.Close()

	// 1a mensagem precisa ser {event:"auth", data:"<senha>"}
	var first wsMsg
	if err := conn.ReadJSON(&first); err != nil {
		return
	}
	// password vazio no server = sem senha (modo publico / auto-descoberta)
	if first.Event != "auth" || (password != "" && first.Data != password) {
		_ = conn.WriteJSON(&wsMsg{Event: "error", Data: "auth failed"})
		return
	}

	// anti-spoof: identidade do peer (IP publico + userId opcional do Palworld)
	peerIP := clientIP(r)
	peerUser := first.User
	if isBlocked(peerUser, peerIP) { // politica B: bloqueado no nivel da voz
		_ = conn.WriteJSON(&wsMsg{Event: "error", Data: "blocked"})
		return
	}

	id := randID()

	pc, err := api.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{{URLs: []string{"stun:stun.l.google.com:19302"}}},
	})
	if err != nil {
		log.Println("newpc:", err)
		return
	}
	defer func() { _ = pc.Close() }()

	// so audio: server recebe a track do cliente
	if _, err := pc.AddTransceiverFromKind(webrtc.RTPCodecTypeAudio,
		webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly}); err != nil {
		log.Println("transceiver:", err)
		return
	}

	pc.OnICECandidate(func(i *webrtc.ICECandidate) {
		if i == nil {
			return
		}
		b, _ := json.Marshal(i.ToJSON())
		_ = conn.WriteJSON(&wsMsg{Event: "candidate", Data: string(b)})
	})

	pc.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		if s == webrtc.PeerConnectionStateFailed || s == webrtc.PeerConnectionStateClosed {
			_ = pc.Close()
		}
	})

	// audio que chega do cliente -> vira track local (StreamID=id) -> fan-out pros outros
	pc.OnTrack(func(t *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
		local := addTrack(t, id)
		if local == nil {
			return
		}
		defer removeTrack(local)
		buf := make([]byte, 1500)
		for {
			n, _, readErr := t.Read(buf)
			if readErr != nil {
				return
			}
			if _, writeErr := local.Write(buf[:n]); writeErr != nil {
				return
			}
		}
	})

	// registra o peer e renegocia
	self := &peerState{id: id, conn: conn, pc: pc, ip: peerIP, userID: peerUser, channel: "proximity"}
	listLock.Lock()
	peers = append(peers, self)
	listLock.Unlock()
	signalPeerConnections()
	defer removePeer(id)

	_ = conn.WriteJSON(&wsMsg{Event: "hello", Data: id})
	// manda a config do servidor (nome + alcance) — "no compose e o padrao"
	if info, err := json.Marshal(map[string]string{"name": serverName, "range": voiceRange}); err == nil {
		_ = conn.WriteJSON(&wsMsg{Event: "serverinfo", Data: string(info)})
	}
	// manda a meta (guild/canal) dos peers JA conectados pro novo, pra ele ja mixar certo.
	listLock.Lock()
	for _, q := range peers {
		if q.id == id || (q.guild == "" && q.channel == "") {
			continue
		}
		if b, e := json.Marshal(map[string]string{"guild": q.guild, "channel": q.channel}); e == nil {
			_ = conn.WriteJSON(&wsMsg{Event: "peermeta", ID: q.id, Data: string(b)})
		}
	}
	listLock.Unlock()

	// loop de leitura
	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			return
		}
		var m wsMsg
		if err := json.Unmarshal(raw, &m); err != nil {
			continue
		}
		switch m.Event {
		case "candidate":
			var cand webrtc.ICECandidateInit
			if json.Unmarshal([]byte(m.Data), &cand) == nil {
				_ = pc.AddICECandidate(cand)
			}
		case "answer":
			var ans webrtc.SessionDescription
			if json.Unmarshal([]byte(m.Data), &ans) == nil {
				_ = pc.SetRemoteDescription(ans)
			}
		case "identify":
			// anti-spoof: o FGuid do player replica ~6s depois de entrar no mundo, mas
			// o companion auto-conecta em ~50ms -> o auth costuma vir com user vazio.
			// Aqui o companion completa a identidade quando o id fica pronto. self.userID
			// so e lido nesta mesma goroutine (case "pos" -> resolveOutgoingPos), sem corrida.
			if m.User != "" {
				self.userID = m.User
				// identidade mudou no meio da sessao -> re-deriva a reconciliacao do
				// zero contra o novo ancora (nao carrega divergeN/lastX do match por IP).
				self.divergeN, self.hasLast = 0, false
				if isBlocked(self.userID, peerIP) { // pode ter sido bloqueado (politica B)
					_ = conn.WriteJSON(&wsMsg{Event: "error", Data: "blocked"})
					return
				}
			}
		case "pos":
			// anti-spoof: reconcilia com a REST (off=passa direto). out = o que
			// vai pros outros; drop=true (politica B) -> derruba este peer.
			out, drop := resolveOutgoingPos(self, m.Data)
			broadcastPos(id, out) // "x,y,z,yaw" (do cliente, da REST, ou ultima valida)
			if drop {
				return
			}
		case "meta":
			// canais de voz: o companion seta seu guild + canal atual. O servidor so
			// REPASSA a meta; quem decide ouvir (proximidade/guild/global) e o companion.
			var meta struct {
				Guild   string `json:"guild"`
				Channel string `json:"channel"`
			}
			if json.Unmarshal([]byte(m.Data), &meta) == nil {
				self.guild = meta.Guild
				if meta.Channel != "" {
					self.channel = meta.Channel
				}
				broadcastMeta(self)
			}
		}
	}
}

// ---------- tracks / SFU ----------

func addTrack(t *webrtc.TrackRemote, peerID string) *webrtc.TrackLocalStaticRTP {
	listLock.Lock()
	local, err := webrtc.NewTrackLocalStaticRTP(t.Codec().RTPCodecCapability, t.ID(), peerID)
	if err != nil {
		listLock.Unlock()
		return nil
	}
	trackLocals[t.ID()] = local
	listLock.Unlock()
	signalPeerConnections()
	return local
}

func removeTrack(t *webrtc.TrackLocalStaticRTP) {
	if t == nil {
		return
	}
	listLock.Lock()
	delete(trackLocals, t.ID())
	listLock.Unlock()
	signalPeerConnections()
}

func removePeer(id string) {
	listLock.Lock()
	for i := range peers {
		if peers[i].id == id {
			peers = append(peers[:i], peers[i+1:]...)
			break
		}
	}
	listLock.Unlock()
	signalPeerConnections()
}

func broadcastPos(fromID, data string) {
	listLock.Lock()
	defer listLock.Unlock()
	for _, p := range peers {
		if p.id == fromID {
			continue
		}
		_ = p.conn.WriteJSON(&wsMsg{Event: "pos", ID: fromID, Data: data})
	}
}

// broadcastMeta repassa o guild + canal atual de um peer pros outros — o companion usa
// pra decidir o mix (proximidade=panner 3D; guild=plano SE mesmo guild; global=plano).
func broadcastMeta(p *peerState) {
	b, err := json.Marshal(map[string]string{"guild": p.guild, "channel": p.channel})
	if err != nil {
		return
	}
	listLock.Lock()
	defer listLock.Unlock()
	for _, q := range peers {
		if q.id == p.id {
			continue
		}
		_ = q.conn.WriteJSON(&wsMsg{Event: "peermeta", ID: p.id, Data: string(b)})
	}
}

// signalPeerConnections sincroniza as tracks de cada peer e renegocia.
// (adaptado do exemplo sfu-ws do pion; audio-only)
func signalPeerConnections() {
	listLock.Lock()
	defer listLock.Unlock()

	attemptSync := func() (tryAgain bool) {
		for i := range peers {
			if peers[i].pc.ConnectionState() == webrtc.PeerConnectionStateClosed {
				peers = append(peers[:i], peers[i+1:]...)
				return true // lista mudou, recomeca
			}

			already := map[string]bool{}
			for _, sender := range peers[i].pc.GetSenders() {
				if sender.Track() == nil {
					continue
				}
				already[sender.Track().ID()] = true
				if _, ok := trackLocals[sender.Track().ID()]; !ok {
					if err := peers[i].pc.RemoveTrack(sender); err != nil {
						return true
					}
				}
			}
			// nao devolver pro peer a track que ele mesmo manda
			for _, recv := range peers[i].pc.GetReceivers() {
				if recv.Track() == nil {
					continue
				}
				already[recv.Track().ID()] = true
			}
			for trackID := range trackLocals {
				if !already[trackID] {
					if _, err := peers[i].pc.AddTrack(trackLocals[trackID]); err != nil {
						return true
					}
				}
			}

			offer, err := peers[i].pc.CreateOffer(nil)
			if err != nil {
				return true
			}
			if err = peers[i].pc.SetLocalDescription(offer); err != nil {
				return true
			}
			b, _ := json.Marshal(offer)
			if err = peers[i].conn.WriteJSON(&wsMsg{Event: "offer", Data: string(b)}); err != nil {
				return true
			}
		}
		return false
	}

	for sync := 0; ; sync++ {
		if sync == 25 {
			go func() { time.Sleep(3 * time.Second); signalPeerConnections() }()
			return
		}
		if !attemptSync() {
			return
		}
	}
}

// ---------- helper: writer de ws thread-safe ----------

type threadSafeWriter struct {
	*websocket.Conn
	sync.Mutex
}

func (t *threadSafeWriter) WriteJSON(v interface{}) error {
	t.Lock()
	defer t.Unlock()
	return t.Conn.WriteJSON(v)
}
