# Changelog

Histórico do PalProxVoice, organizado por **milestone** (V1 → V1.5 → V2 → V3; ver
[docs/ROADMAP.md](docs/ROADMAP.md)). Formato inspirado em
[Keep a Changelog](https://keepachangelog.com/). Cada milestone lista
**Adicionado / Alterado / Corrigido**.

---

## Canais de voz · ETW · zero-config _(develop, mais recente)_

### Adicionado
- **Canais de voz — proximidade / guild / global.** O peer manda `meta{guild,channel}`; o servidor repassa (`peermeta`); o **companion mistura por canal** (proximidade = panner 3D; guild/global = som plano; guild só entre a mesma guild). **Hotkey global Alt+V** (`RegisterHotKey`) cicla o canal **in-game**, sem focar o app nem admin.
- **Guild — auto + código manual.** O companion prefere a guild que o mod escreve (`palproxvoice_guild.txt`) sobre um código digitado; `probe_guild.lua` descobre a reflexão da guild in-game (leitura real pendente dos logs).
- **Auto-connect zero-config — acha a PORTA sozinho.** Além do IP, o companion **testa as portas comuns** da voz (8765, 8766, …) e conecta na 1ª que responde como PalProxVoice. Zero digitação.
- **ETW — IP do servidor em tempo real.** Lê o UDP do processo do Palworld no kernel (`Microsoft-Windows-Kernel-Network`) via `golang-etw` (no-CGO), filtra pelo PID, pega o destino público mais frequente → `palproxvoice_server.txt` (o `DetectGameServerIP()` já consome com prioridade). **Precisa de admin**; degrada sem crashar.
- **Game Pass (WinGDK) confirmado** — UE4SS + o mod cliente rodam na versão Microsoft Store; o `DetectGameServerIP` lê o IP tanto de `WinGDK` quanto de `Windows`.

## V2 — server-side _(funciona · experimental)_

Posição **e yaw autoritativos do servidor** — o cliente nunca afirma posição. Validado in-game (o server escuta, o mod carrega, o feed grava). Roda o `.exe` Windows do PalServer **sob Proton** no Linux.

### Adicionado
- **mod server-side** (`mod-server/`) — lê pos+yaw+FGuid+nome de todos os `PlayerController` e escreve `palproxvoice_players.txt` (feed que a voz consome via `PPV_PLAYERS_FILE`). Hook de join (`ServerAcknowledgePossession`).
- **anti-spoof V2** (`server/antispoof_v2.go`) — lê o feed, casa por FGuid/nome, tem prioridade sobre a REST. Off por padrão.
- **deploy Proton** (`deploy/v2-experimental/linux`) — imagem debian + GE-Proton que auto-baixa o PalServer (depot Windows) + UE4SS Okaetsu + o mod; portas deslocadas. `docker-compose.v2.yml` pro Dokploy. Windows-nativo via `install.ps1`.

### Corrigido (a saga do Proton, em ordem)
- `container_name` conflitava no Dokploy → removido; `python3` faltava (launcher do Proton); prefixo Proton criado do zero (não copiar `default_pfx`); **symlinks do `steamclient.so`** em `~/.steam/sdk{32,64}` (Proton #9068, era o loop de restart); **GUI console do UE4SS off** + Xvfb (RE-UE4SS #497, travava o boot headless); **`PROTON_NO_ESYNC/NO_FSYNC`** (o esync deadlockava — o destrave final); mod **20Hz → 5Hz + cache de PlayerControllers** + **NetDriver timeout** (o loop saturava o game thread no spawn → disconnect).

## V1.5 — anti-spoof _(codado · off por padrão)_

Servidor de voz **agnóstico de fonte**: a posição pode vir do cliente OU da REST do Palworld, mesma forma. `PPV_AUTH_MODE = off | verify | strict`.

### Adicionado
- **`server/antispoof.go`** — reconcilia a posição do cliente com a REST (`GET /v1/api/players`), casa o peer por **FGuid → IP → IP+proximidade**, e ignora mentiras **sustentadas** (guard-rails contra falso-positivo de fast-travel/montaria). Ban escalonado opcional (política B). **Mantém `Z`+`yaw` do cliente** (a REST não tem) — anti-spoof no horizontal, direção do cliente. Deploy: [ANTI-SPOOF-DEPLOY.md](docs/ANTI-SPOOF-DEPLOY.md).

---

## V1 — auto-connect

O companion descobre o servidor pelo IP em que você está e conecta a voz sozinho.
Produto funcional ponta-a-ponta, validado com pessoas reais pela internet.

### Adicionado
- **interface multilíngue (i18n PT/EN)** — toda a UI **e os logs** do painel no idioma escolhido, com seletor 🌐; padrão = idioma do sistema (salvo). Extensível (basta somar ao dicionário).
- **abre e fecha junto com o Palworld** — um *watcher* leve ([`companion/cmd/watcher`](companion/cmd/watcher)) sobe na inicialização do Windows, lança o companion (oculto) quando o Palworld abre e o companion sai sozinho quando o jogo fecha. O instalador inicia o watcher **logo após instalar** (sem esperar o próximo login).
- **README bilíngue** — [EN](README.md) + [PT-BR](README.pt-BR.md).
- **captura de mic NATIVA (WASAPI/go-wca)** — fora do WebView2, com `AudioCategory_Other`. O `getUserMedia` do Chromium abria como categoria de **comunicação** e punha o codec em "modo voz", degradando TODO o áudio do sistema (música/jogo). Agora a captura é nativa (PCM → WS local → AudioWorklet → WebRTC), **sem degradar nada**. Forçada a **48 kHz** (autoconvert) → fim do áudio "robotizado" quando o device muda de taxa (ex.: Discord em modo comunicação).
- **áudio espacial reescrito** — listener fixo + posição por *bearing* relativo (corrige espelhamento esquerda/direita), **crossfade perto** (presente/centralizado) **↔ longe** (HRTF direcional), cutoff de proximidade (fade ao silêncio no alcance), abafamento de quem está **atrás** (reduz a confusão frente/trás do HRTF) e suavização (`setTargetAtTime`). Alinhado ao Palworld (UE5: cm + câmera via `ControlRotation`).
- **painel de processamento do mic** — passa-alta, compressor e **noise gate** (sensibilidade + medidor ao vivo) liga/desliga individual; **RNNoise (IA, Xiph BSD, local)** opcional; **🎧 ouvir meu microfone** (monitor). Seleção de **microfone e saída por nome** (mic nativo via WASAPI).
- **robustez pra internet ruim** — Opus **DTX** + **bitrate adaptativo automático** (48→16 kbps por perda/RTT, com histerese) + **reconexão automática** em queda de rede (backoff), além do FEC já existente.
- **mod UE4SS** (Lua) — lê posição (X,Y,Z) + yaw a ~20 Hz e escreve em `C:\Users\Public\palproxvoice_pos.txt`. Client-side, blindado contra estados ruins de objeto.
- **servidor de voz** (Go + pion/webrtc) — SFU: cada mic sobe uma vez e é repassado a todos; relay de posição por WebSocket. Sem sala (pool único + senha). `serverinfo` (nome/alcance) e senha opcional.
- **companion** (Wails, Go+WebView2) — app único: lê a posição do jogo e faz voz por proximidade **3D** (Web Audio/HRTF) na mesma janela.
  - dashboard + modo **overlay** compacto (janela frameless, fora da barra de tarefas, always-on-top).
  - multi-servidor na config; **conectar/desconectar manual**; fallback "Configurar/Ignorar".
  - controles de áudio: volume de entrada/saída, **mutar mic**, **deafen**; escolher microfone e saída.
  - fecha a voz sozinho ao sair do servidor; aparece/some junto com o jogo.
- **auto-detecção do IP do servidor** — `DetectGameServerIP()`: tenta o mod C++ (sessão atual) → `PalOptionSaveGame` → `GameUserSettings.ini`. Cobre **Direct Connect e join pela lista do Steam**. ([`companion/serverdetect.go`](companion/serverdetect.go), com teste)
- **voz mais limpa** — captura com `echoCancellation`/`noiseSuppression`/`autoGainControl`/mono; Opus com **FEC** (`useinbandfec=1`, corrige perda de pacote), mono e **bitrate 48 kbps** (`setParameters` + SDP).
- **mod C++** ([`mod-live/`](mod-live/)) — _scaffold_ que serve o IP da sessão atual por socket local via `LowLevelGetRemoteAddress` (build no Windows).
- **instalador 1-clique** (Inno Setup) — acha o Palworld em qualquer biblioteca Steam, instala UE4SS + mod + companion + config e configura auto-start.
- **CI** — GitHub Actions no push de tag `v*`: builda o companion (Windows) e anexa `palproxvoice.exe`, `PalProxVoice-mod.zip` e o bundle `PalProxVoice-UE4SS.zip` (UE4SS v3.0.1 + mod) na release.
- **docs** — [ARCHITECTURE.md](docs/ARCHITECTURE.md), [ROADMAP.md](docs/ROADMAP.md), READMEs por componente.

### Alterado
- transporte mod→companion por **arquivo** (UE4SS não traz LuaSocket); o companion lê direto, aposentando a bridge HTTP separada.
- auto-detecção do IP agora usa `DetectGameServerIP()` (live→save→ini) no lugar de só o `GameUserSettings.ini`.
- permissão de CI: `contents: write` pro `GITHUB_TOKEN` criar releases.

### Corrigido
- **auto-connect não pegava quem entrava pela lista do Steam** (só Direct Connect, via `GameUserSettings.ini`). Agora o `PalOptionSaveGame` cobre os dois.
- mic cai pro dispositivo padrão sem `OverconstrainedError`; não reconecta em dobro.
- **instalador** acha o Palworld dentro/ao redor da pasta escolhida (antes exigia a raiz exata) + botão **"Procurar nos discos"** que varre e lista todos os Palworlds.
- **áudio do sistema degradava ao conectar** (parecia "ducking", mas o volume não caía — a **qualidade de tudo** virava voz estreita: música perde os instrumentos, ambiente do jogo, ex. grilos, some; só voz sobra). **Causa real:** o `getUserMedia` do WebView2 abre a captura do mic como **`AudioCategory_Communications`**, que opta a sessão na "communications policy" do Windows e põe o **codec compartilhado** (mic + fone no mesmo chip, ex. Realtek/headset USB) em **"modo voz"** — degradando toda a saída. A categoria é **hard-coded no WebRTC do Chromium**; Discord/Teams escapam por usarem engine de áudio nativa, não `getUserMedia`. (Descartado por medição: não era ducking de volume, nem o ducking de Comunicação do Windows, nem Discord, nem enhancements do device.) **Fix (o jeito do Discord):** captura do mic **nativa via WASAPI** com **`AudioCategory_Other`** ([`companion/mic_capture_windows.go`](companion/mic_capture_windows.go), go-wca), entregue por um WebSocket local de PCM → **AudioWorklet** ([`companion/frontend/dist/mic-feed.js`](companion/frontend/dist/mic-feed.js)) → `MediaStreamDestination` → `RTCPeerConnection`. **Sem `getUserMedia`** → sem categoria de comunicação → não degrada mais.
- **janela sumia no Alt+Tab** em modo cheio: ela era *tool window* (fora do Alt+Tab/taskbar) e ia pra trás sem volta. Agora cheio = app window normal; *tool window* só no overlay compacto.
- **atualizar instalação existente**: o instalador fecha o companion em execução antes de copiar (o `.exe` travado fazia o update falhar) e **barra com aviso claro se o Palworld estiver aberto** (os DLLs do UE4SS ficam em uso → antes quebrava no meio).

### Notas / limites honestos
- posição é **client-reported** → spoofável. Anti-spoof vem na V1.5.
- requer **UE4SS no cliente** de cada jogador. Some na V2 (server-side).
- mod C++ ([`mod-live/`](mod-live/)) ainda não foi compilado/testado (precisa build no Windows).

---

## Como manter este changelog

A cada milestone (ou release), adicione no topo a seção do milestone com as
subseções **Adicionado / Alterado / Corrigido**. "Corrigido" é onde entram os
bugs resolvidos desde o anterior.
