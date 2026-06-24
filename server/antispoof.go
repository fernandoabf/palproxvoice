// PalProxVoice — anti-spoof (V1.5)
//
// A posicao que o cliente reporta ("x,y,z,yaw") e CLIENT-REPORTED -> spoofavel.
// Aqui a gente reconcilia com a fonte AUTORITATIVA: a REST API do servidor
// dedicado do Palworld (GET /v1/api/players), que devolve, por jogador:
// userId, ip e location_x/location_y. DESCOBERTA importante: location_x/y vem na
// MESMA escala do mundo (cm) que o mod escreve -> reconciliar e' so distancia 2D,
// sem conversao de coordenadas.
//
// AUTH_MODE:
//
//	off     -> repassa a posicao do cliente como veio (comportamento legado).
//	verify  -> repassa a do cliente, mas se ela divergir da REST acima da
//	           tolerancia por N polls SEGUIDOS, trata como spoof:
//	             politica A (padrao): IGNORA a mentira -> repassa a ultima
//	               posicao valida (cai pra REST se nao houver) -> o trapaceiro
//	               nao consegue se "teletransportar" pros teus ouvidos.
//	             politica B (PPV_BAN=1): alem de ignorar, derruba o peer e
//	               bloqueia (userId+ip) no NIVEL DA VOZ (nao bane do jogo).
//	strict  -> sempre repassa a posicao da REST (cliente nunca afirma posicao).
//	           OBS: yaw continua do cliente (a REST nao tem direcao de camera).
//
// Correlacao peer <-> jogador da REST:
//  1. por userId, se o cliente mandou um (mais robusto; cobre mesmo-IP).
//  2. senao por IP: 1 jogador naquele IP -> casa direto.
//  3. mesmo IP com varios jogadores (mesma casa) -> casa pelo MAIS PROXIMO da
//     posicao reportada (e o verify cuida de manter honesto dali pra frente).
//
// O servidor de voz deve rodar na MESMA rede Docker do Palworld e falar com a
// REST por dentro (ex.: PPV_REST_URL=http://palworld:8212) — a REST e' API de
// ADMIN e NUNCA deve ser publica.
package main

import (
	"encoding/json"
	"log"
	"math"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ---------- config (env) ----------

var (
	authMode    = "off" // off | verify | strict
	restURL     string  // ex.: http://palworld:8212
	restUser    string  // ex.: admin
	restPass    string
	restPollMS  = 2000
	posTolCM    = 5000.0 // tolerancia de divergencia (cm); ~50 m
	divergeMax  = 4      // polls seguidos divergindo p/ considerar spoof
	banEnabled  bool     // politica B
	restTimeout = 6 * time.Second
)

func initAntispoof() {
	authMode = strings.ToLower(envOr("PPV_AUTH_MODE", "off"))
	restURL = strings.TrimRight(os.Getenv("PPV_REST_URL"), "/")
	restUser = envOr("PPV_REST_USER", "admin")
	restPass = os.Getenv("PPV_REST_PASS")
	if v, err := strconv.Atoi(os.Getenv("PPV_REST_POLL_MS")); err == nil && v >= 250 {
		restPollMS = v
	}
	if v, err := strconv.ParseFloat(os.Getenv("PPV_POS_TOL_CM"), 64); err == nil && v > 0 {
		posTolCM = v
	}
	if v, err := strconv.Atoi(os.Getenv("PPV_DIVERGE_POLLS")); err == nil && v >= 1 {
		divergeMax = v
	}
	banEnabled = os.Getenv("PPV_BAN") == "1"

	if authMode == "off" {
		log.Printf("anti-spoof: OFF (posicao confiada cegamente)")
		return
	}
	if restURL == "" {
		log.Printf("anti-spoof: %s pedido mas PPV_REST_URL vazio -> caindo pra OFF", authMode)
		authMode = "off"
		return
	}
	log.Printf("anti-spoof: %s  rest=%s poll=%dms tol=%.0fcm diverge=%d ban=%v",
		authMode, restURL, restPollMS, posTolCM, divergeMax, banEnabled)
	go restPollLoop()
}

// ---------- snapshot da REST ----------

type restPlayer struct {
	Name     string  `json:"name"`
	UserID   string  `json:"userId"`
	PlayerID string  `json:"playerId"`
	IP       string  `json:"ip"`
	X        float64 `json:"location_x"`
	Y        float64 `json:"location_y"`
}

type restPlayersResp struct {
	Players []restPlayer `json:"players"`
}

var (
	restMu     sync.RWMutex
	restByUser = map[string]restPlayer{}   // userId -> jogador
	restByIP   = map[string][]restPlayer{} // ip -> jogadores (pode ser >1 na mesma casa)
	restAt     time.Time                   // quando o ultimo poll OK chegou
)

func restPollLoop() {
	client := &http.Client{Timeout: restTimeout}
	t := time.NewTicker(time.Duration(restPollMS) * time.Millisecond)
	defer t.Stop()
	for {
		pollREST(client)
		<-t.C
	}
}

func pollREST(client *http.Client) {
	req, err := http.NewRequest(http.MethodGet, restURL+"/v1/api/players", nil)
	if err != nil {
		return
	}
	req.SetBasicAuth(restUser, restPass)
	resp, err := client.Do(req)
	if err != nil {
		return // erro/lag da REST -> NUNCA pune; mantem o snapshot anterior
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return
	}
	var pr restPlayersResp
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		return
	}
	byUser := make(map[string]restPlayer, len(pr.Players))
	byIP := make(map[string][]restPlayer)
	for _, p := range pr.Players {
		// indexado por userId E por playerId — o cliente pode mandar qualquer um
		// (o mod le mais facil o playerId/FGuid; o userId e' da plataforma).
		if p.UserID != "" {
			byUser[p.UserID] = p
		}
		if p.PlayerID != "" {
			byUser[p.PlayerID] = p
			// FGuid e' hex (case-insensitive); o mod escreve em MAIUSCULO. Indexa a
			// versao upper tambem pra casar independente do case que a REST devolver.
			byUser[strings.ToUpper(p.PlayerID)] = p
		}
		if p.IP != "" {
			byIP[p.IP] = append(byIP[p.IP], p)
		}
	}
	restMu.Lock()
	restByUser, restByIP, restAt = byUser, byIP, time.Now()
	restMu.Unlock()
}

// restStale: sem poll OK recente -> trata como indisponivel (nunca pune).
func restStale() bool {
	restMu.RLock()
	defer restMu.RUnlock()
	return restAt.IsZero() || time.Since(restAt) > time.Duration(restPollMS*4)*time.Millisecond
}

// ---------- correlacao + reconciliacao ----------

// lookupAuthoritative acha a posicao autoritativa pro peer (por userId -> IP ->
// IP+proximidade). ok=false quando a REST nao conhece esse peer (ainda).
func lookupAuthoritative(p *peerState, cliX, cliY float64) (rp restPlayer, ok bool) {
	restMu.RLock()
	defer restMu.RUnlock()
	if p.userID != "" {
		if rp, ok = restByUser[p.userID]; ok {
			return rp, true
		}
		// fallback case-insensitive (FGuid em case diferente). userId de plataforma
		// ("gdk_/steam_") em upper nao colide com nada -> sem falso positivo.
		if up := strings.ToUpper(p.userID); up != p.userID {
			if rp, ok = restByUser[up]; ok {
				return rp, true
			}
		}
	}
	cands := restByIP[p.ip]
	switch len(cands) {
	case 0:
		return restPlayer{}, false
	case 1:
		return cands[0], true
	default: // mesma casa (mesmo IP): casa pelo mais proximo do reportado
		best, bestD := cands[0], math.Inf(1)
		for _, c := range cands {
			if d := dist2(cliX, cliY, c.X, c.Y); d < bestD {
				best, bestD = c, d
			}
		}
		return best, true
	}
}

func dist2(ax, ay, bx, by float64) float64 {
	dx, dy := ax-bx, ay-by
	return math.Sqrt(dx*dx + dy*dy)
}

// resolveOutgoingPos decide o "x,y,z,yaw" que vai ser repassado pelo sender p.
// drop=true (so na politica B) sinaliza que o peer deve ser derrubado/bloqueado.
func resolveOutgoingPos(p *peerState, raw string) (out string, drop bool) {
	// V2 (EXPERIMENTAL): se o feed do mod server-side tem esse peer, usa a posicao+yaw
	// autoritativos e IGNORA o que o cliente reportou. Prioridade sobre a REST.
	if v2, ok := v2Pos(p); ok {
		return v2, false
	}
	if authMode == "off" {
		return raw, false
	}
	f := strings.Split(raw, ",")
	if len(f) < 4 {
		return raw, false // formato inesperado -> nao mexe
	}
	cliX, ex := strconv.ParseFloat(strings.TrimSpace(f[0]), 64)
	cliY, ey := strconv.ParseFloat(strings.TrimSpace(f[1]), 64)
	if ex != nil || ey != nil {
		return raw, false
	}
	z, yaw := f[2], f[3] // z e yaw seguem do cliente (REST nao tem yaw)

	if restStale() {
		return raw, false // REST indisponivel/laggada -> nunca pune
	}
	rp, ok := lookupAuthoritative(p, cliX, cliY)
	if !ok {
		// a REST ainda nao "ve" esse peer (acabou de entrar / trocando de mundo).
		// verify: deixa passar (sem fonte p/ punir). strict: idem ate aparecer.
		return raw, false
	}

	if authMode == "strict" {
		// posicao 100% da REST; mantem z/yaw do cliente.
		p.lastX, p.lastY, p.hasLast = rp.X, rp.Y, true
		return fmtPos(rp.X, rp.Y, z, yaw), false
	}

	// verify
	if dist2(cliX, cliY, rp.X, rp.Y) <= posTolCM {
		p.divergeN = 0
		p.lastX, p.lastY, p.hasLast = cliX, cliY, true
		return raw, false // honesto -> repassa o do cliente (baixa latencia)
	}
	// divergiu: so age se for SUSTENTADO (fast-travel/montaria se resolve sozinho)
	p.divergeN++
	if p.divergeN < divergeMax {
		return raw, false
	}
	// SPOOF sustentado -> politica A: ignora a mentira (ultima valida, senao REST)
	gx, gy := rp.X, rp.Y
	if p.hasLast {
		gx, gy = p.lastX, p.lastY
	}
	if banEnabled { // politica B: derruba + bloqueia no nivel da voz
		blockAdd(p.userID, p.ip)
		log.Printf("anti-spoof: SPOOF de %s (ip=%s user=%q) -> bloqueado (politica B)", p.id, p.ip, p.userID)
		return fmtPos(gx, gy, z, yaw), true
	}
	return fmtPos(gx, gy, z, yaw), false
}

func fmtPos(x, y float64, z, yaw string) string {
	return strconv.FormatFloat(x, 'f', 1, 64) + "," +
		strconv.FormatFloat(y, 'f', 1, 64) + "," + z + "," + yaw
}

// ---------- blocklist (nivel da voz; em memoria) ----------

var (
	blockMu      sync.RWMutex
	blockedUsers = map[string]bool{}
	blockedIPs   = map[string]bool{}
)

func blockAdd(userID, ip string) {
	blockMu.Lock()
	defer blockMu.Unlock()
	if userID != "" {
		blockedUsers[userID] = true
	}
	if ip != "" {
		blockedIPs[ip] = true
	}
}

func isBlocked(userID, ip string) bool {
	if !banEnabled {
		return false
	}
	blockMu.RLock()
	defer blockMu.RUnlock()
	return (userID != "" && blockedUsers[userID]) || (ip != "" && blockedIPs[ip])
}

// ---------- helper: IP do cliente (respeita reverse proxy) ----------

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.IndexByte(xff, ','); i >= 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}
