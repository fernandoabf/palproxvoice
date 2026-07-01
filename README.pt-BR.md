<div align="center">

# 🎙️ PalProxVoice

**Voz por proximidade 3D pro Palworld — self-hosted, sem serviço de terceiro.**

_Você ouve quem está perto de você no jogo — mais alto quanto mais perto, e com direção real (áudio 3D / HRTF). Tipo o Simple Voice Chat do Minecraft, mas pro Palworld. Sem sala: tudo roda na sua infra._

![status](https://img.shields.io/badge/status-alpha%20ativo-2ea44f)
![plataforma](https://img.shields.io/badge/plataforma-Windows%20%C2%B7%20Linux-1f6feb)
![game](https://img.shields.io/badge/Palworld-Steam%20%C2%B7%20Game%20Pass-orange)
![self-hosted](https://img.shields.io/badge/self--hosted-sim-success)
![audio](https://img.shields.io/badge/audio-3D%20HRTF%20%2B%20Opus-blueviolet)
![licenca](https://img.shields.io/badge/licen%C3%A7a-ver%20LICENSE-lightgrey)

🇬🇧 **[Read in English](README.md)** · [⬇️ Baixar](../../releases) · [📝 Novidades](CHANGELOG.md) · [🏗️ Arquitetura](docs/ARCHITECTURE.md) · [🗺️ Roadmap](docs/ROADMAP.md)

</div>

> 🎥 **Demo:** _(em breve — ~20s da voz por proximidade 3D; ouça de **fone** 🎧)_

---

## 🚀 Comece aqui — escolha seu caminho

| Você é… | Vá para |
|---|---|
| 🎮 **Só quero falar com meus amigos** | **[Para jogadores](#-para-jogadores)** — baixa, instala, joga (~2 min) |
| 🖥️ **Eu tenho um servidor de Palworld** | **[Para donos de servidor](#-para-donos-de-servidor)** |
| 🧑‍💻 **Quero desenvolver / contribuir** | **[Para desenvolvedores](#-para-desenvolvedores)** |

---

## 🎮 Para jogadores

**Você precisa:** Palworld (**Steam _ou_ Game Pass**) + fone de ouvido + **um amigo ou dono de servidor rodando o servidor de voz do PalProxVoice** (veja [Para donos de servidor](#-para-donos-de-servidor) — se ninguém subiu ainda, manda esse link pra ele).

1. Baixe o **`PalProxVoice-Setup.exe`** na [última release](../../releases).
2. **Execute.** O Windows pede **permissão de administrador** (aquele aviso azul) — clique em **Sim** (é preciso pra instalar o mod dentro do jogo). No **Steam** ele acha o Palworld sozinho; se não achar (ex.: **Game Pass**), clique em **Procurar nos discos** e escolha a pasta do Palworld.
3. **Abra o Palworld e jogue.** A voz **conecta sozinha** — sem digitar IP nem porta. Se o servidor tiver senha de voz, peça pro dono. Use fone. 🎧

Pronto. Quem estiver perto de você no mundo, você ouve — mais alto quanto mais perto, e com direção (esquerda/direita/frente/trás).

> **Apareceu "O Windows protegeu o computador"?** É só o aviso do Windows pra programas novos sem assinatura paga. Clique em **Mais informações → Executar assim mesmo**. É seguro (o código é aberto).

<details><summary><b>Sem instalador? (modo manual)</b></summary>

Na mesma release, pegue o `PalProxVoice-UE4SS.zip` (extrai em `Pal\Binaries\<Win64 ou WinGDK>\`) e o `palproxvoice.exe` (o companion). Roda o companion — ele auto-conecta do mesmo jeito.
</details>

### 🗣️ Canais de voz
Tipo o chat do jogo. Aperte **Alt+V** enquanto joga pra trocar em qual canal **você fala** (o canal atual aparece no app do PalProxVoice):

- **📍 Proximidade** _(padrão)_ — ouve quem está perto, em 3D.
- **🛡️ Guild** — fala com a sua guild, de qualquer distância.
- **🌐 Global** — fala com o servidor todo.

Você sempre **ouve** guild (mesma guild) e global; proximidade só quando perto.

### ✨ O que você ganha
- **Voz posicional 3D** — direção + distância, acompanhando a câmera do jogo.
- **Zero configuração** — conecta sozinho no server em que você está. Nada pra digitar.
- **Não estraga o resto do teu áudio** — sua música e o som do jogo continuam normais enquanto você fala (aquele problema clássico de voz no navegador foi resolvido).
- **Supressão de ruído por IA** (opcional) — corta barulho de fundo tipo ventilador ou teclado. Mais um painel de ajuste de mic pros exigentes.
- **Escolhe o microfone e a saída por nome.**
- **Feito pra internet ruim** — qualidade que se ajusta sozinha + reconexão automática em quedas.
- **Roda junto com o jogo** — abre com o Palworld, some quando você fecha. Você não abre nada.

---

## 🧩 Versões

O PalProxVoice cresce em camadas. A **base** funciona em qualquer servidor; cada versão adiciona **anti-trapaça mais forte / menos delay** sem quebrar as de baixo. Como jogador você não escolhe — quem escolhe é o dono do servidor.

| Tag | Status | O que adiciona | Anti-spoof | Servidor precisa de |
|:---:|:---:|---|:---:|---|
| **Base + V1** | 🟢 **funciona** | voz por proximidade + **auto-connect** (acha o IP do server e a porta da voz sozinho) | — _confia no cliente_ | só o container do voz |
| **V1.5** | 🟢 **codado** _(opcional)_ | **reconciliação por REST** — trapaceiro não se teleporta pros seus ouvidos. Mantém `Z`+`yaw` do cliente. | ✅ horizontal | + REST do Palworld alcançável por dentro |
| **V2** | 🟡 **server-side funciona** _(exp.)_ | **posição autoritativa server-side** (pos+yaw+FGuid @5Hz) via **mod UE4SS no servidor** (`.exe` Windows sob **Proton** no Linux). Zero confiança no cliente. _Voz e2e (2 players) ainda não testada._ | ✅✅ 3D completo | + UE4SS no servidor dedicado |
| **V3** | 🔬 **pesquisa** _(só scaffold)_ | a posição do V2, mas no servidor **nativo de Linux** — leitor de memória externo, **sem UE4SS / Proton / Wine**. | ✅✅ 3D completo | _(ainda não construído)_ |

> **Qual eu rodo?** A maioria: **Base + V1** (só funciona). Proximidade à prova de trapaça num server normal: liga o **V1.5**. Posição 3D autoritativa com delay mínimo: **V2**. **V3** é experimento inicial — veja [a branch](../../tree/experimental/v3-linux-native).

---

## 🔧 Como funciona

Três peças. O jogo nunca fala com o servidor de voz direto — quem faz a ponte é o companion.

| Peça | Onde | O que faz |
|---|---|---|
| **mod** (UE4SS/Lua) | PC de cada jogador (Steam **ou** Game Pass) | lê posição + direção + identidade, escreve arquivos locais |
| **companion** (app Wails) | PC de cada jogador | lê a posição, manda o mic, recebe os outros, **espacializa em 3D** e **auto-conecta** |
| **servidor de voz** (Go/pion) | VPS do dono | SFU: cada mic sobe uma vez e é repassado pra todos; relay de posição + canal. Sem sala. |

Cada track de áudio sai com `StreamID = id do peer`, pro companion casar **áudio ↔ posição**. No **V2/V3**, a posição vira **autoritativa do servidor** (um mod/leitor no servidor escreve um feed que a voz consome), então o cliente não consegue mentir onde está.

---

## 🖥️ Para donos de servidor

O servidor de voz é um container Go. Roda ao lado do seu Palworld (mesma VPS ou outra). Todo mundo que entra com a senha é um pool único; quem você ouve é 100% proximidade/canal — sem sala.

### 1️⃣ Base / V1 — todo mundo começa aqui
```bash
cp .env.example .env          # defina VOICE_PASSWORD e PUBLIC_IP=<ip-da-vps>  (ou PUBLIC_IP=auto)
docker compose up -d --build
```
- **Firewall:** abra **UDP 50000–50010** (o áudio). O `PUBLIC_IP` faz o server anunciar teu IP (sem TURN); `auto` descobre sozinho.
- **Porta da voz:** o `HTTP_PORT` do `.env` é a porta **do host** (padrão **8765**). O companion **testa as portas 8765–8768** e conecta na 1ª que responde como PalProxVoice — então exponha a voz numa delas.
- **Mic no navegador** precisa de HTTPS (`wss://`, ex.: atrás do Dokploy). O **companion desktop conecta por IP direto — sem TLS/proxy** (é o caminho normal).

### 2️⃣ V1.5 — anti-spoof (REST), opcional
No `.env`: `PPV_AUTH_MODE=verify` (ou `strict`) + `PPV_REST_URL=http://<serviço-do-palworld>:8212` + `PPV_REST_PASS=<AdminPassword do Palworld>`. A REST é API de **admin** — **nunca** exponha pública; a voz alcança ela **por dentro da rede docker**. Então a voz e o Palworld têm que dividir uma rede: este compose cria a rede **`palprox`** — suba a **voz primeiro** (ela cria a rede), depois ponha o Palworld nela como `external` (veja o [`docker-compose.palworld.example.yml`](docker-compose.palworld.example.yml)). A voz reconcilia a posição reportada com a REST e ignora mentiras sustentadas; mantém a altura + direção do cliente (a REST não tem). → [docs/ANTI-SPOOF-DEPLOY.md](docs/ANTI-SPOOF-DEPLOY.md)

### 3️⃣ V2 — autoritativo server-side (experimental)
Um **mod UE4SS no servidor dedicado** escreve um feed `fguid,x,y,z,yaw` @5Hz; a voz consome (zero confiança no cliente). No Linux o `.exe` Windows roda sob **Proton**. Um comando (Dokploy ou local):
```bash
docker compose -f docker-compose.v2.yml up -d --build
```
- **Portas deslocadas** (pra coexistir com um Palworld de produção): jogo **8311/udp**, query **27115/udp**, voz **8766**, mídia **UDP 50100–50110** — abra essas.
- **Status:** o server-side está validado in-game (o servidor escuta, o mod carrega, o feed grava pos+FGuid reais); a **voz ponta-a-ponta com 2 players ainda não foi testada**. Rode num **host descartável** primeiro.

> A parte frágil é o **UE4SS-sob-Proton** — a gente resolveu a corrente inteira (symlink do steamclient, GUI console headless, esync/fsync, NetDriver timeout, taxa do mod). Notas completas + o caminho Windows-nativo: **[deploy/v2-experimental/README.md](deploy/v2-experimental/README.md)**.

### 🗣️ Canais de voz (qualquer versão)
Proximidade / guild / global funcionam de fábrica. A **guild** vem do mod do jogador (auto) ou de um **código de guild** compartilhado no companion (fallback). Sem config extra no servidor.

---

## 🧑‍💻 Para desenvolvedores

### Layout do repo
```
mod/PalProxVoice/        mod UE4SS (CLIENTE) — lê pos+yaw+FGuid -> C:\Users\Public\palproxvoice_*.txt
mod-server/              [V2] mod UE4SS (SERVIDOR) — feed autoritativo pos+yaw+FGuid (fguid,x,y,z,yaw) @5Hz
companion/               app Wails (Go+WebView2) — voz 3D, auto-connect (detecta IP + probe de porta + ETW),
                         canais de voz, mic nativo WASAPI
server/                  servidor de voz Go — SFU + relay de posição/canal + anti-spoof (V1.5 REST, V2 feed)
deploy/v2-experimental/  [V2] servidor Palworld MODADO (Proton+UE4SS): linux/ (Docker) + windows/ (install.ps1)
v3-linux-native/         [V3] leitor de memória nativo de Linux (scaffold) — na branch experimental/v3-linux-native
installer/               instalador Inno Setup (UE4SS + mod + companion + auto-start)
docs/                    ARCHITECTURE · ROADMAP · ANTI-SPOOF-DEPLOY
docker-compose.yml                    servidor de voz (Base / V1 / V1.5)
docker-compose.v2.yml                 [V2] Palworld modado (Proton) + voz — pro Dokploy
docker-compose.palworld.example.yml   exemplo: um Palworld com REST na rede `palprox` (pro V1.5)
```

### Testar local (sem jogo, sem VPS)
```bash
cp .env.example .env          # VOICE_PASSWORD=test, PUBLIC_IP vazio
docker compose up --build     # só o servidor de voz (API; sem cliente web embutido)
```
O servidor é **só-API** — conecte com o **companion** apontando pra `localhost:<HTTP_PORT>`. Precisa de dois clientes (duas máquinas) pra ouvir o SFU + espacialização.

### Build do companion
Windows (WebView2), ou via GitHub Actions no push de tag `v*`. Ver [companion/BUILD.md](companion/BUILD.md). Testes do Go: `go test ./...` em `companion/`.

### Contribuindo
Issues e PRs bem-vindos. O acoplamento com o jogo é fino e isolado de propósito (o Palworld 1.0 sai em 2026-07-10 e updates grandes quebram mods de UE4SS) — veja as decisões travadas no [ROADMAP](docs/ROADMAP.md) antes de mexer no `mod/`.

---

## 🗺️ Roadmap (resumo)

| Marco | O quê | Status |
|---|---|:---:|
| **V1** | auto-connect: acha o server pelo IP + **descobre a porta da voz** | 🟢 feito |
| **V1.5** | anti-spoof por reconciliação REST (`verify`/`strict`) | 🟢 codado |
| **V2** | posição autoritativa server-side via UE4SS (Proton) | 🟡 server-side funciona |
| **Canais** | proximidade / guild / global + troca global (Alt+V) | 🟢 construído |
| **IP realtime** | ETW lê o IP do server atual ao vivo (UDP do kernel) | 🟢 construído |
| **V3** | leitor de memória nativo de Linux (sem Proton/Wine) | 🔬 pesquisa |
| **próximos** | assinatura de código · fechar o auto-guild · anúncio do endereço da voz in-game | 🔜 planejado |

Detalhe completo em [docs/ROADMAP.md](docs/ROADMAP.md).

## 📄 Licença

Ver [LICENSE](LICENSE).
