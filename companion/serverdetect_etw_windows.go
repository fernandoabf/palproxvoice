//go:build windows

package main

// ETW: descobre, em TEMPO REAL, o IP do servidor de jogo que o Palworld esta conectado,
// lendo os pacotes UDP (provider Microsoft-Windows-Kernel-Network). UDP nao aparece em
// GetExtendedUdpTable -> por isso ETW. Escreve no serverFile (palproxvoice_server.txt) que
// o GameServerIPLive()/DetectGameServerIP() (serverdetect.go) ja consomem com prioridade.
//
// PRECISA DE ADMIN (sessao ETW de kernel). Sem admin: DEGRADA (loga e segue; as outras
// fontes de IP — save/ini — continuam valendo). Roda elevado pra ligar o ETW.

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/0xrawsec/golang-etw/etw"
	"golang.org/x/sys/windows"
)

// EventIDs UDP do provider Microsoft-Windows-Kernel-Network.
const (
	evUDPSendV4 = 42 // send IPv4 -> servidor = daddr/dport (caminho principal)
	evUDPRecvV4 = 43 // recv IPv4 -> servidor = saddr/sport
	evUDPSendV6 = 58
	evUDPRecvV6 = 59
)

var palPIDs atomic.Value // map[uint32]bool

// StartServerIPWatchETW liga a sessao ETW + consumidor. NAO bloqueia (spawna goroutines);
// chame com `go StartServerIPWatchETW(ctx)` no startup.
func StartServerIPWatchETW(ctx context.Context) {
	ses := etw.NewRealTimeSession("PalProxVoiceNet")
	// Microsoft-Windows-Kernel-Network = {7DD42A49-5329-4832-8DFD-43D979153A88}
	prov := etw.MustParseProvider("Microsoft-Windows-Kernel-Network")
	if err := ses.EnableProvider(prov); err != nil {
		ses.Stop()
		logServerIP("ETW indisponivel (rodar como Admin?): " + err.Error())
		return // degrada: as outras fontes de DetectGameServerIP seguem valendo
	}
	refreshPalPIDs()
	c := etw.NewRealTimeConsumer(ctx)
	c.FromSessions(ses)
	agg := newIPAgg()

	go func() { // janela: a cada 3s pega o vencedor e escreve se mudou
		t := time.NewTicker(3 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				refreshPalPIDs()
				if ip := agg.winner(); ip != "" {
					writeServerIP(ip)
				}
			}
		}
	}()
	go func() {
		for e := range c.Events {
			handleNetEvent(e, agg)
		}
	}()
	go func() {
		<-ctx.Done()
		c.Stop()
		ses.Stop()
	}()

	if err := c.Start(); err != nil {
		c.Stop()
		ses.Stop()
		logServerIP("ETW Start falhou: " + err.Error())
		return
	}
}

func handleNetEvent(e *etw.Event, agg *ipAgg) {
	var ipKey, portKey string
	switch e.System.EventID {
	case evUDPSendV4, evUDPSendV6:
		ipKey, portKey = "daddr", "dport"
	case evUDPRecvV4, evUDPRecvV6:
		ipKey, portKey = "saddr", "sport"
	default:
		// provider MOF antigo: o EventID pode chegar 0 -> infere pela presenca das chaves.
		if firstField(e.EventData, "daddr") != "" {
			ipKey, portKey = "daddr", "dport"
		} else if firstField(e.EventData, "saddr") != "" {
			ipKey, portKey = "saddr", "sport"
		} else {
			return
		}
	}
	pid, ok := fieldUint32(e.EventData, "PID")
	if !ok || !isPalPID(pid) {
		return // so o trafego do Palworld
	}
	ip := strings.TrimSpace(firstField(e.EventData, ipKey))
	if ip == "" {
		return
	}
	if parsed := net.ParseIP(ip); parsed == nil || isLocalOrLAN(parsed) {
		return // descarta loopback/LAN/CGNAT/multicast
	}
	if strings.TrimSpace(firstField(e.EventData, portKey)) == "53" {
		return // DNS
	}
	size, _ := fieldUint32(e.EventData, "size")
	agg.add(ip, strings.TrimSpace(firstField(e.EventData, portKey)), int(size))
}

// ----- agregador: vencedor por bytes (desempate por pacotes), janela ~3s -----

type ipStat struct{ pkts, bytes int }

type ipAgg struct {
	mu  sync.Mutex
	cur map[string]*ipStat // chave = "ip:port"
}

func newIPAgg() *ipAgg { return &ipAgg{cur: map[string]*ipStat{}} }

func (a *ipAgg) add(ip, port string, size int) {
	key := ip
	if port != "" {
		key = ip + ":" + port
	}
	a.mu.Lock()
	s := a.cur[key]
	if s == nil {
		s = &ipStat{}
		a.cur[key] = s
	}
	s.pkts++
	s.bytes += size
	a.mu.Unlock()
}

// winner: "ip:port" vencedor da janela; zera o acumulador. Exige minimo de pacotes (mata
// ruido de matchmaking/EOS).
func (a *ipAgg) winner() string {
	const minPkts = 10
	a.mu.Lock()
	defer a.mu.Unlock()
	best, bb, bp := "", -1, 0
	for k, s := range a.cur {
		if s.bytes > bb || (s.bytes == bb && s.pkts > bp) {
			best, bb, bp = k, s.bytes, s.pkts
		}
	}
	a.cur = map[string]*ipStat{}
	if bp < minPkts {
		return ""
	}
	return best
}

// ----- PID do Palworld (Toolhelp32, reusa palworldExes do palworld_watch_windows.go) -----

func refreshPalPIDs() {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return
	}
	defer windows.CloseHandle(snap)
	m := map[uint32]bool{}
	var e windows.ProcessEntry32
	e.Size = uint32(unsafe.Sizeof(e))
	if windows.Process32First(snap, &e) != nil {
		return
	}
	for {
		n := strings.ToLower(windows.UTF16ToString(e.ExeFile[:]))
		for _, want := range palworldExes {
			if n == want {
				m[e.ProcessID] = true
			}
		}
		if windows.Process32Next(snap, &e) != nil {
			break
		}
	}
	palPIDs.Store(m)
}

func isPalPID(pid uint32) bool {
	m, _ := palPIDs.Load().(map[uint32]bool)
	return m[pid]
}

// ----- helpers de campo (defensivo: nome/casing/tipo variam por lib/versao) -----

func fieldVariants(name string) []string {
	switch strings.ToLower(name) {
	case "pid":
		return []string{"PID", "Pid", "pid", "ProcessId"}
	case "daddr":
		return []string{"daddr", "DestAddr", "destaddr", "Daddr"}
	case "saddr":
		return []string{"saddr", "SourceAddr", "sourceaddr", "Saddr"}
	case "dport":
		return []string{"dport", "DestPort", "destport", "Dport"}
	case "sport":
		return []string{"sport", "SourcePort", "sourceport", "Sport"}
	case "size":
		return []string{"size", "Size"}
	}
	return []string{name}
}

func firstField(d map[string]interface{}, primary string) string {
	for _, k := range fieldVariants(primary) {
		if v, ok := d[k]; ok {
			return toStr(v)
		}
	}
	return ""
}

func fieldUint32(d map[string]interface{}, primary string) (uint32, bool) {
	for _, k := range fieldVariants(primary) {
		v, ok := d[k]
		if !ok {
			continue
		}
		switch t := v.(type) {
		case string:
			n, err := strconv.ParseUint(strings.TrimSpace(t), 10, 32)
			return uint32(n), err == nil
		case float64:
			return uint32(t), true
		case uint32:
			return t, true
		case uint64:
			return uint32(t), true
		case int:
			return uint32(t), true
		}
	}
	return 0, false
}

func toStr(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	default:
		return ""
	}
}

func isLocalOrLAN(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() || ip.IsUnspecified() || ip.IsPrivate() {
		return true
	}
	if v4 := ip.To4(); v4 != nil && v4[0] == 100 && v4[1] >= 64 && v4[1] <= 127 {
		return true // CGNAT 100.64/10
	}
	return false
}

// ----- escrita do IP (atomico, so se mudou) + log -----

var lastWrittenIP atomic.Value

func writeServerIP(ipPort string) {
	if last, _ := lastWrittenIP.Load().(string); last == ipPort {
		return
	}
	lastWrittenIP.Store(ipPort)
	tmp := serverFile + ".tmp"
	if os.WriteFile(tmp, []byte(ipPort), 0o644) == nil {
		_ = os.Rename(tmp, serverFile) // GameServerIPLive() le e faz hostOnly() (tira a porta)
		logServerIP("ETW: server = " + ipPort)
	}
}

func logServerIP(msg string) {
	pub := os.Getenv("PUBLIC")
	if pub == "" {
		pub = `C:\Users\Public`
	}
	if f, err := os.OpenFile(filepath.Join(pub, "palproxvoice_etw.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644); err == nil {
		defer f.Close()
		_, _ = f.WriteString(msg + "\n")
	}
}
