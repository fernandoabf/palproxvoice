# Changelog

Histórico do PalProxVoice, organizado por **milestone** (V1 → V1.5 → V2; ver
[docs/ROADMAP.md](docs/ROADMAP.md)). Formato inspirado em
[Keep a Changelog](https://keepachangelog.com/). Cada milestone lista
**Adicionado / Alterado / Corrigido**.

---

## V1 — auto-connect _(atual)_

O companion descobre o servidor pelo IP em que você está e conecta a voz sozinho.
Produto funcional ponta-a-ponta, validado com pessoas reais pela internet.

### Adicionado
- **mod UE4SS** (Lua) — lê posição (X,Y,Z) + yaw a ~20 Hz e escreve em `C:\Users\Public\palproxvoice_pos.txt`. Client-side, blindado contra estados ruins de objeto.
- **servidor de voz** (Go + pion/webrtc) — SFU: cada mic sobe uma vez e é repassado a todos; relay de posição por WebSocket. Sem sala (pool único + senha). `serverinfo` (nome/alcance) e senha opcional.
- **companion** (Wails, Go+WebView2) — app único: lê a posição do jogo e faz voz por proximidade **3D** (Web Audio/HRTF) na mesma janela.
  - dashboard + modo **overlay** compacto (janela frameless, fora da barra de tarefas, always-on-top).
  - multi-servidor na config; **conectar/desconectar manual**; fallback "Configurar/Ignorar".
  - controles de áudio: volume de entrada/saída, **mutar mic**, **deafen**; escolher microfone e saída.
  - fecha a voz sozinho ao sair do servidor; aparece/some junto com o jogo.
- **auto-detecção do IP do servidor** — `DetectGameServerIP()`: tenta o mod C++ (sessão atual) → `PalOptionSaveGame` → `GameUserSettings.ini`. Cobre **Direct Connect e join pela lista do Steam**. ([`companion/serverdetect.go`](companion/serverdetect.go), com teste)
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
- **ducking de comunicação do Windows** (mic abaixava todos os outros sons): o **.exe** grava `UserDuckingPreference=3` no HKCU do **usuário** — o instalador, rodando como admin, podia gravar no hive errado.
- **janela sumia no Alt+Tab** em modo cheio: ela era *tool window* (fora do Alt+Tab/taskbar) e ia pra trás sem volta. Agora cheio = app window normal; *tool window* só no overlay compacto.
- **atualizar instalação existente**: o instalador fecha o companion em execução antes de copiar (o `.exe` travado fazia o update falhar).

### Notas / limites honestos
- posição é **client-reported** → spoofável. Anti-spoof vem na V1.5.
- requer **UE4SS no cliente** de cada jogador. Some na V2 (server-side).
- mod C++ ([`mod-live/`](mod-live/)) ainda não foi compilado/testado (precisa build no Windows).

---

## V1.5 — anti-spoof _(planejado)_

Servidor de voz **agnóstico de fonte**: posição do cliente OU do servidor, mesma
forma. `authoritativeMode: off | verify | strict`, identidade por `userId` +
IP-match, reconciliação com a REST API e ban escalonado. Detalhe no
[ROADMAP](docs/ROADMAP.md#próximas-releases--v1--v15--v2).

## V2 — server-side _(planejado)_

Mod no servidor (UE4SS sob Proton) com posição **e** yaw autoritativos. Cliente
não precisa mais de UE4SS. Anti-spoof real.

---

## Como manter este changelog

A cada milestone (ou release), adicione no topo a seção do milestone com as
subseções **Adicionado / Alterado / Corrigido**. "Corrigido" é onde entram os
bugs resolvidos desde o anterior.
