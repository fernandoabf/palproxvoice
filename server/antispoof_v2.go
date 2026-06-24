// PalProxVoice — anti-spoof V2 (EXPERIMENTAL)
//
// Fonte autoritativa = o FEED do mod SERVER-SIDE (UE4SS) que escreve
// `palproxvoice_players.txt` no host do servidor de jogo, 1 linha por player:
//
//	<fguid>,<x>,<y>,<z>,<yaw>
//
// Diferente do verify/strict (que usam a REST: 2D, ~1s, SEM yaw), o feed traz a
// posicao a 20Hz E o YAW de verdade -> espacializacao direcional autoritativa, com
// delay minimo. O cliente nunca afirma posicao (zero spoof).
//
// Liga so quando PPV_PLAYERS_FILE aponta pro arquivo (o voz e o servidor de jogo
// precisam compartilhar esse arquivo — mesmo host / volume docker compartilhado).
// Off por padrao. Tem prioridade sobre o caminho da REST quando o peer esta no feed.
package main

import (
	"bufio"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	playersFile string // PPV_PLAYERS_FILE; vazio = V2 desligado
	feedPollMS  = 100

	feedMu      sync.RWMutex
	feedByFGuid = map[string]feedPos{} // FGuid (MAIUSCULO) -> posicao
	feedAt      time.Time
)

type feedPos struct{ x, y, z, yaw string }

func initV2() {
	playersFile = strings.TrimSpace(os.Getenv("PPV_PLAYERS_FILE"))
	if playersFile == "" {
		return
	}
	if v, err := strconv.Atoi(os.Getenv("PPV_FEED_POLL_MS")); err == nil && v >= 20 {
		feedPollMS = v
	}
	log.Printf("anti-spoof V2 (EXPERIMENTAL): feed=%s poll=%dms (posicao+yaw autoritativos do mod server-side)",
		playersFile, feedPollMS)
	go feedPollLoop()
}

func feedPollLoop() {
	t := time.NewTicker(time.Duration(feedPollMS) * time.Millisecond)
	defer t.Stop()
	for {
		pollFeed()
		<-t.C
	}
}

func pollFeed() {
	f, err := os.Open(playersFile)
	if err != nil {
		return // sem feed -> mantem o anterior; staleness em v2Pos cuida do resto
	}
	defer f.Close()
	m := make(map[string]feedPos)
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		parts := strings.Split(strings.TrimSpace(sc.Text()), ",")
		if len(parts) < 5 || parts[0] == "" || parts[0] == "?" {
			continue
		}
		m[strings.ToUpper(parts[0])] = feedPos{x: parts[1], y: parts[2], z: parts[3], yaw: parts[4]}
	}
	feedMu.Lock()
	feedByFGuid, feedAt = m, time.Now()
	feedMu.Unlock()
}

// v2Pos: "x,y,z,yaw" autoritativo do feed pro peer (casado por FGuid). ok=false se V2
// off, feed stale, peer sem FGuid, ou peer fora do feed -> cai pro verify/REST/raw.
func v2Pos(p *peerState) (string, bool) {
	if playersFile == "" || p.userID == "" {
		return "", false
	}
	feedMu.RLock()
	defer feedMu.RUnlock()
	if feedAt.IsZero() || time.Since(feedAt) > time.Duration(feedPollMS*8)*time.Millisecond {
		return "", false // feed parado/ausente -> nunca usa stale
	}
	fp, ok := feedByFGuid[strings.ToUpper(p.userID)]
	if !ok {
		return "", false
	}
	return fp.x + "," + fp.y + "," + fp.z + "," + fp.yaw, true
}
