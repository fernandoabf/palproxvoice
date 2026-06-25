<div align="center">

# 🎙️ PalProxVoice

**Voz por proximidade 3D pro Palworld, self-hosted**

_Tipo o Simple Voice Chat do Minecraft, mas pro Palworld — você ouve quem está perto de você no mundo do jogo, mais alto quanto mais perto, com direção de verdade (áudio 3D / HRTF). Sem sala, sem canal, sem serviço de terceiro: tudo roda na sua infra._

![status](https://img.shields.io/badge/status-alpha%20ativo-2ea44f)
![plataforma](https://img.shields.io/badge/plataforma-Windows%20%C2%B7%20Linux-1f6feb)
![game](https://img.shields.io/badge/Palworld-Steam%20%C2%B7%20Game%20Pass-orange)
![self-hosted](https://img.shields.io/badge/self--hosted-sim-success)
![voz](https://img.shields.io/badge/audio-3D%20HRTF%20%2B%20Opus-blueviolet)
![licenca](https://img.shields.io/badge/licen%C3%A7a-ver%20LICENSE-lightgrey)

🇬🇧 **[Read in English](README.md)** · [Releases](../../releases) · [CHANGELOG](CHANGELOG.md) · [Arquitetura](docs/ARCHITECTURE.md) · [Roadmap](docs/ROADMAP.md)

</div>

> 🎥 **Demo:** _(em breve — clipe de ~20s da voz por proximidade 3D; ouça de **fone** 🎧)_

---

## 🧩 Versões / Marcos

O PalProxVoice cresce em camadas. A **base** (voz por proximidade) funciona em qualquer servidor; cada versão adiciona **anti-spoof melhor / menos delay** sem quebrar as de baixo.

| Tag | Status | O que adiciona | Anti-spoof | Precisa de | Onde mora |
|:---:|:---:|---|:---:|---|---|
| **Base + V1** | 🟢 **funciona** | voz por proximidade + **auto-connect** (o companion acha o servidor pelo seu IP e **descobre a porta da voz** sozinho) | — (confia no cliente) | mod cliente + companion | `companion/` · `server/` · `mod/` |
| **V1.5** | 🟢 **codado** _(off por padrão)_ | **anti-spoof via REST** — reconcilia a posição do cliente com a REST do Palworld (`verify`/`strict`). Trapaceiro não se teletransporta pros seus ouvidos. Mantém `Z`+`yaw` do cliente (a REST não tem). | ✅ horizontal | + REST do Palworld alcançável por dentro | `server/antispoof.go` · `PPV_AUTH_MODE` |
| **V2** | 🟢 **funciona** _(experimental)_ | **posição autoritativa server-side** (pos+yaw+FGuid @ 5Hz), lida por um **mod UE4SS no servidor** (`.exe` Windows sob **Proton** no Linux). Zero confiança na posição do cliente. | ✅✅ 3D completo | + UE4SS no servidor dedicado | `mod-server/` · `deploy/v2-experimental/` · `docker-compose.v2.yml` |
| **V3** | 🔬 **pesquisa** _(Fase 0)_ | a mesma posição autoritativa do V2, mas no servidor **NATIVO de Linux** — um **leitor de memória externo** (`process_vm_readv` + AOB scan), **sem UE4SS, sem Proton, sem Wine**. | ✅✅ 3D completo | + `CAP_SYS_PTRACE` | branch [`experimental/v3-linux-native`](../../tree/experimental/v3-linux-native) |

> **Qual eu quero?** A maioria: **Base + V1** (só funciona). Quer proximidade à prova de trapaça num servidor normal: liga o **V1.5**. Quer posição 3D autoritativa com delay mínimo: **V2** (validado ponta-a-ponta). **V3** é o experimento puro-Linux pros corajosos.

---

## ✨ Recursos

- **Voz posicional 3D** (Web Audio / HRTF) — esquerda/direita/frente/trás + distância, acompanhando a câmera do jogo.
- **Auto-connect zero-config** — o companion detecta o IP do servidor pelo jogo e **acha a porta da voz sozinho** (probe). Sem digitar endereço.
- **Captura de mic nativa de baixa latência** (WASAPI) — **não** degrada o resto do áudio do sistema (a pegadinha clássica do navegador/`getUserMedia`).
- **Supressão de ruído por IA** (RNNoise) opcional + noise gate + compressor + monitor "ouvir meu mic".
- **Escolha o microfone e a saída por nome.**
- **Feito pra internet ruim** — Opus FEC + DTX, **bitrate adaptativo** automático e **reconexão automática** em quedas.
- **Sem sala** — um pool único por servidor + senha; quem você ouve é 100% proximidade.
- **Anti-spoof em camadas** — da reconciliação por REST (V1.5) até posição totalmente autoritativa no servidor (V2/V3).
- **Instalador 1-clique** (UE4SS + mod + companion + auto-start). Funciona em **Steam e Game Pass** (WinGDK).

---

## 🔧 Como funciona

Três peças. O jogo nunca fala com o servidor de voz direto — quem faz a ponte é o companion.

| # | Peça | Onde | O que faz |
|---|------|------|-----------|
| **mod** | UE4SS (Lua) | PC de cada jogador (Windows/Game Pass) | lê posição + direção + FGuid, escreve arquivos locais |
| **companion** | app Wails (Go+WebView2) | PC de cada jogador | lê a posição, manda o mic, recebe os outros e **espacializa em 3D** (Web Audio/HRTF); **auto-conecta** |
| **server** | Go + pion/webrtc | VPS do dono | SFU: cada mic sobe uma vez e é repassado pra todos; relay de posição + anti-spoof. Sem sala. |

Cada track de áudio sai com `StreamID = id do peer`, pro companion casar **áudio ↔ posição**. No **V2/V3**, a posição vira **autoritativa do servidor** (um mod/leitor no servidor escreve um feed que a voz consome), então o cliente não consegue mentir onde está.

---

## 🎮 Para jogadores

1. Baixe o instalador da [última release](../../releases) (`PalProxVoice-Setup.exe`).
2. Rode — ele acha o Palworld (ou clique **Procurar em um disco** pra varrer), instala UE4SS + o mod + o companion e configura o auto-start. Funciona em **Steam** e **Game Pass / Microsoft Store** (WinGDK).
3. Entre no jogo. O companion **acha o servidor sozinho** (IP + porta da voz) e conecta. Use fone. 🎧

> **"O Windows protegeu o computador"?** É o SmartScreen avisando que o app ainda não tem assinatura reconhecida — normal pra binário novo/open-source. **Mais informações → Executar assim mesmo.** (Assinatura de código no roadmap.)

Sem instalador: pegue `PalProxVoice-UE4SS.zip` (UE4SS + mod, extrai em `Pal\Binaries\<Win64|WinGDK>\`) e `palproxvoice.exe` (companion) na release.

---

## 🖥️ Para donos de servidor

O servidor de voz é um container Go. Roda ao lado do seu Palworld (mesma VPS ou outra).

### Base / V1 (todo mundo)
```bash
cp .env.example .env          # defina VOICE_PASSWORD e PUBLIC_IP=<ip-publico-da-vps> (ou "auto")
docker compose up -d --build
```
- O companion **auto-detecta** o IP do jogo e **testa as portas comuns da voz** (`8765`, `8766`, …) — exponha a voz numa delas.
- **Áudio (mídia):** abra **UDP 50000–50010** no firewall. O `PUBLIC_IP` faz o pion anunciar seu IP (sem TURN). `PUBLIC_IP=auto` se descobre sozinho.
- **Mic no navegador** precisa de contexto seguro — ponha a voz atrás de um domínio com **TLS** (`wss://`). O companion desktop conecta por IP direto.

### V1.5 — anti-spoof (REST)
Defina `PPV_AUTH_MODE=verify` (ou `strict`) + `PPV_REST_URL`/`PPV_REST_PASS` pra voz alcançar a **REST do Palworld por dentro** (API de admin — **nunca** exponha pública). A voz reconcilia a posição reportada com a REST e ignora mentiras sustentadas. Ver [docs/ANTI-SPOOF-DEPLOY.md](docs/ANTI-SPOOF-DEPLOY.md).

### V2 — autoritativo server-side (experimental)
Um **mod UE4SS no servidor dedicado** escreve um feed `fguid,x,y,z,yaw,nome` @ 5Hz; a voz consome (zero confiança no cliente). No Linux o `.exe` Windows roda sob **Proton**. Deploy num comando (Dokploy ou local):
```bash
docker compose -f docker-compose.v2.yml up -d --build
```
Notas completas (as pegadinhas de Proton que resolvemos: symlink do steamclient, GUI console headless, esync/fsync, NetDriver timeout): [deploy/v2-experimental/README.md](deploy/v2-experimental/README.md).

### V3 — Linux nativo (pesquisa)
Sem Proton/UE4SS — um leitor de memória externo no servidor nativo (estilo `thijsvanloef`). Ver a branch [`experimental/v3-linux-native`](../../tree/experimental/v3-linux-native).

---

## 🧑‍💻 Para desenvolvedores

### Layout do repo
```
mod/PalProxVoice/        mod UE4SS (CLIENTE) — lê pos+yaw+FGuid -> C:\Users\Public\palproxvoice_*.txt
mod-server/              [V2] mod UE4SS (SERVIDOR) — feed pos+yaw+FGuid autoritativo @ 5Hz
companion/               app desktop Wails (Go+WebView2) — voz 3D + auto-connect (detecta IP + probe de porta)
server/                  servidor de voz Go — SFU + relay de posição + anti-spoof (V1.5 REST, V2 feed)
deploy/v2-experimental/  [V2] servidor Palworld MODADO (Proton+UE4SS): linux/ (Docker) + windows/ (install.ps1)
v3-linux-native/         [V3] leitor de memória nativo de Linux (Fase 0) — na branch experimental/v3-linux-native
installer/               instalador Inno Setup (UE4SS + mod + companion + auto-start)
docs/                    ARCHITECTURE · ROADMAP · ANTI-SPOOF-DEPLOY
docker-compose.yml       servidor de voz (Base / V1 / V1.5)
docker-compose.v2.yml    [V2] Palworld modado (Proton) + voz — pro Dokploy
```

### Testar local (sem jogo, sem VPS)
```bash
cp .env.example .env          # VOICE_PASSWORD=test, PUBLIC_IP vazio
docker compose up --build     # só o servidor de voz (API; sem cliente web embutido)
```
O servidor é **só-API** — conecte com o **companion** apontando pra `localhost:<HTTP_PORT>`. Precisa de dois clientes (duas máquinas) pra ouvir o SFU + espacialização.

### Build do companion
Windows (WebView2) ou via GitHub Actions no push de tag `v*`. Ver [companion/BUILD.md](companion/BUILD.md). Testes do Go: `go test ./...` em `companion/`.

### Contribuindo
Issues e PRs bem-vindos. O acoplamento com o jogo é fino e isolado de propósito (o Palworld 1.0 sai em 2026-07-10 e updates grandes quebram mods de UE4SS) — veja as decisões travadas no [ROADMAP](docs/ROADMAP.md) antes de mexer no `mod/`.

---

## 🗺️ Roadmap

| | Marco | Status |
|:---:|---|:---:|
| **V1** | auto-connect: o companion acha o servidor pelo seu IP + **descobre a porta da voz** | 🟢 feito |
| **V1.5** | anti-spoof: protocolo agnóstico de fonte + reconciliação por REST (`verify`/`strict`) | 🟢 codado |
| **V2** | server-side: mod UE4SS no servidor (Proton) com pos+yaw autoritativos; cliente sem UE4SS pra posição | 🟢 funciona |
| **V3** | Linux nativo: leitor de memória externo (`process_vm_readv`), sem Proton/Wine | 🔬 pesquisa |
| **próximos** | assinatura de código · push-to-talk · anúncio do endereço da voz in-game (RPC servidor → cliente) | 🔜 planejado |

Detalhe em [docs/ROADMAP.md](docs/ROADMAP.md).

## 📄 Licença

Ver [LICENSE](LICENSE).
