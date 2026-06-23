# PalProxVoice

**Voz por proximidade 3D pro Palworld, self-hosted** — tipo o Simple Voice Chat
do Minecraft, mas pro Palworld. Você ouve quem está perto de você no mundo do
jogo, mais alto quanto mais perto, com direção (áudio 3D/HRTF). Sem sala, sem
canal: um pool único por servidor + senha + proximidade. Nada de serviço de
terceiro — tudo roda na sua infra.

🇬🇧 **[Read in English](README.md)**

> Status: **alpha ativo** (milestone **V1**) — validado ponta-a-ponta com pessoas
> reais pela internet. Baixe a [última release](../../releases) · histórico em
> [CHANGELOG.md](CHANGELOG.md) · desenho em [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)
> · plano em [docs/ROADMAP.md](docs/ROADMAP.md).

> 🎥 **Demo:** _(em breve — clipe de ~20s da voz por proximidade 3D; ouça de **fone** 🎧)_

## Recursos

- **Voz posicional 3D** (Web Audio / HRTF) — esquerda/direita/frente/trás + distância, acompanhando a câmera do jogo.
- **Captura de mic nativa de baixa latência** (WASAPI) — **não** degrada o resto do áudio do sistema (a pegadinha clássica do navegador/`getUserMedia`).
- **Supressão de ruído por IA** (RNNoise) opcional + noise gate + compressor + monitor "ouvir meu mic".
- **Escolha o microfone e a saída por nome.**
- **Feito pra internet ruim** — Opus FEC + DTX, **bitrate adaptativo** automático e **reconexão automática** em quedas.
- **Sem sala** — um pool único por servidor + senha; quem você ouve é 100% proximidade.
- **Instalador 1-clique** (UE4SS + mod + companion + auto-start).

## Como funciona

Três peças. O jogo nunca fala com o servidor de voz direto — quem faz a ponte é o companion.

| # | Peça | Onde | O que faz |
|---|------|------|-----------|
| **mod** | UE4SS (Lua) | PC de cada jogador (Windows) | lê posição+direção e escreve num arquivo local |
| **companion** | app Wails (Go+WebView2) | PC de cada jogador | lê a posição, manda o mic, recebe os outros e **espacializa em 3D** (Web Audio/HRTF) |
| **server** | Go + pion/webrtc | VPS do dono | SFU: cada mic sobe uma vez e é repassado; relay de posição. Sem sala. |

Cada track de áudio sai com `StreamID = id do peer`, pro companion casar **áudio ↔ posição**.

---

## Para jogadores

1. Baixe o instalador da [última release](../../releases) (`PalProxVoice-Setup.exe`).
2. Rode — ele acha o Palworld (ou clique **Procurar em um disco** pra varrer), instala UE4SS + o mod + o companion e configura o auto-start.
3. Entre no jogo. O companion **detecta o servidor sozinho** e conecta a voz. Use fone. 🎧

> **"O Windows protegeu o computador"?** É o SmartScreen avisando que o app ainda
> não tem assinatura de código reconhecida — normal pra binário novo/open-source.
> Clique em **Mais informações → Executar assim mesmo**. (Assinatura de código no roadmap.)

Sem instalador: pegue `PalProxVoice-UE4SS.zip` (UE4SS + mod, extrai em
`Pal\Binaries\<Win64|WinGDK>\`) e `palproxvoice.exe` (companion) na release.

---

## Para donos de servidor

O servidor de voz é um container Go. Roda ao lado do seu Palworld (mesma VPS ou outra).

```bash
cp .env.example .env          # defina VOICE_PASSWORD e PUBLIC_IP=<ip-publico-da-vps>
docker compose up -d --build
```

- **Porta:** o companion auto-conecta em `IP-do-jogo:8765` por padrão. Exponha o
  voice nessa porta (mapeie host `8765` → container `8080`) **ou** ajuste o
  `autoPort` na config do companion pra casar com a sua.
- **Mic precisa de contexto seguro:** pro acesso via navegador, ponha o voice
  atrás de um domínio com **TLS** (ex.: Dokploy) e use `wss://`. O companion
  desktop conecta por IP direto.
- **Áudio (mídia):** abra **UDP 50000–50010** no firewall. O `PUBLIC_IP` faz o
  pion anunciar seu IP (sem TURN).
- **Sem sala:** todo mundo que entra com a senha é um pool único; quem você ouve
  é 100% proximidade.

Detalhes de TLS/firewall e um Palworld de teste opcional: ver
[docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) e `docker-compose.palworld-test.yml`.

---

## Para desenvolvedores

### Layout do repo

```
mod/PalProxVoice/    mod UE4SS (Lua) — lê posição+yaw, escreve C:\Users\Public\palproxvoice_pos.txt
companion/           app desktop Wails (Go+WebView2) — voz 3D + auto-connect   (BUILD.md)
server/              servidor de voz Go (SFU + relay de posição) + Dockerfile + web/ (cliente de teste)
mod-live/            [V1] mod C++ opcional — IP da sessão atual por socket (scaffold, build no Windows)
bridge/              [legado] ponte arquivo→HTTP pra navegador puro; o companion já faz isso
installer/           instalador Inno Setup (UE4SS + mod + companion + auto-start)
docs/                ARCHITECTURE.md · ROADMAP.md
docker-compose*.yml  voice · palworld de teste · local.sh
```

### Testar local (sem jogo, sem VPS)

```bash
cp .env.example .env          # VOICE_PASSWORD=test, PUBLIC_IP vazio
docker compose up --build     # só o voice
```

Abra `http://localhost:8088` em 2 abas (de fone!), mesma senha, **Entrar**. Mova
com `W A S D`, gire com `← →` — a outra aba muda de lado e volume. Isso exercita o
SFU + a espacialização sem precisar do Palworld. (Porta = `HTTP_PORT` do `.env`;
no WSL2 use 8088.) `./local.sh` sobe voice **+** um Palworld de teste juntos.

### Build do companion

Windows (WebView2) ou via GitHub Actions no push de tag `v*`. Ver
[companion/BUILD.md](companion/BUILD.md). Testes do Go: `go test ./...` em `companion/`.

### Contribuindo

Issues e PRs bem-vindos. O acoplamento com o jogo é fino e isolado de propósito
(o Palworld 1.0 sai em 2026-07-10 e updates grandes quebram mods de UE4SS) — veja
as decisões travadas no [ROADMAP](docs/ROADMAP.md) antes de mexer no `mod/`.

---

## Roadmap (resumo)

- **V1** — auto-connect: o companion acha o servidor pelo IP atual e conecta sozinho.
- **V1.5** — anti-spoof: protocolo agnóstico de fonte + reconciliação (`verify`/`strict`) por `userId` + IP-match.
- **V2** — server-side: mod no servidor (Proton) com posição+yaw autoritativos; cliente sem UE4SS.

Detalhe em [docs/ROADMAP.md](docs/ROADMAP.md).

## Licença

Ver [LICENSE](LICENSE).
