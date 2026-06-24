# Arquitetura — PalProxVoice

Voz por **proximidade** self-hosted pro Palworld (tipo o Simple Voice Chat do
Minecraft). Os jogadores se ouvem conforme a **distância no mundo do jogo**.
Tudo roda na infra do dono (VPS + PCs dos jogadores); nada de serviço de terceiro.

## Por que esse desenho

O servidor dedicado de Palworld roda em **Linux**, mas o **UE4SS** (única forma de
rodar lógica dentro do jogo) é **só Windows** → não dá pra ser um mod server-side.
Então: **mod fino no cliente** lê posição/identidade, e um **backend de voz
self-hosted** faz o resto. A parte acoplada ao jogo fica mínima e isolada (updates
grandes do Palworld quebram mods de UE4SS).

## Três componentes

| # | Componente | Onde | O que faz | Status |
|---|-----------|------|-----------|--------|
| M1 | **Mod** (`mod/PalProxVoice/`) | cliente Windows | UE4SS Lua lê `X,Y,Z`+`Yaw` ~20 Hz e o **FGuid** do player; escreve `C:\Users\Public\palproxvoice_pos.txt` e `..._id.txt` (UE4SS não traz LuaSocket → arquivo) | feito, validado no jogo |
| M2 | **Servidor de voz** (`server/`) | VPS (Docker) | Go + pion/webrtc. SFU: cada mic sobe uma vez, repassa pra todos. Relay de posição por ws. Anti-spoof opcional. Sem sala. Só-API (sem cliente embutido). | feito e compila |
| M3 | **Companion** (`companion/`) | PC de cada jogador | app desktop **Wails** (Go + WebView2). Captura de mic nativa (WASAPI), espacializa em 3D (Web Audio `PannerNode`/HRTF na webview), auto-conecta, abre/fecha junto com o Palworld, i18n PT/EN | feito |

No M2, o `StreamID` de cada track de áudio = id do peer que enviou, pro cliente
correlacionar **áudio ↔ posição**.

## Fluxo de dados

```
Palworld (cliente) ──UE4SS Lua──> C:\Users\Public\palproxvoice_{pos,id}.txt
                                          │ (companion lê)
                                   Companion (Wails)
                            ws: auth(+FGuid), pos 20Hz  │  WebRTC: mic (Opus)
                                          ▼
                       Servidor de voz (SFU + relay de posição + anti-spoof)
                            broadcast: pos + áudio dos outros peers
                                          ▼
                       Companion espacializa cada peer (PannerNode/HRTF)
```

## Anti-spoof (V1.5) — opcional, **off** por padrão

A posição é **client-reported** (spoofável). Em `verify`/`strict`, o servidor de voz
reconcilia com a fonte autoritativa: a **REST API** do servidor dedicado do Palworld
(`GET /v1/api/players` → `userId, playerId (FGuid), ip, location_x/y`).

- **Correlação** peer↔jogador: por **FGuid/userId** (o mod escreve, o companion manda
  no auth e completa via `identify` quando o FGuid replica, ~6 s após entrar no mundo)
  → senão por **IP** → senão por **IP+proximidade** (cobre a mesma casa / 2 PCs).
- `location_x/y` da REST vem na **mesma escala (cm)** do mod → reconciliar = distância 2D.
- **Nunca** pune com REST stale/laggada. Política A (padrão): ignora a mentira sustentada;
  B (`PPV_BAN=1`): derruba + bloqueia no nível da voz (não bane do jogo).

Deploy: voz e Palworld na mesma rede Docker (`palprox`); a REST é API de admin e **nunca**
fica pública. Ver [ANTI-SPOOF-DEPLOY.md](ANTI-SPOOF-DEPLOY.md).

## Descoberta do servidor (qual IP conectar)

O companion descobre o IP do servidor de jogo em 3 fontes (`companion/serverdetect.go`,
da mais precisa pra menos):

1. **`GameServerIPLive`** — arquivo escrito por um mod com o IP da sessão **atual**
   (tempo real). *Em desenvolvimento — ver [`../mod-live/`](../mod-live/).*
2. **`GameServerIPFromSave`** — `PalOptionSaveGame`: pega o servidor **mais frequente**
   do histórico (cobre Direct Connect **e** join pela lista do Steam). ✅ o que funciona hoje.
3. **`GameServerIP`** — `GameUserSettings.ini` (só Direct Connect).

## Decisões travadas

- **Sem sala** — pool único + senha compartilhada + proximidade decide quem você ouve.
- **SFU** no VPS (não mesh P2P) — mais simples e confiável p/ poucos players, sem TURN.
- **Áudio 3D via Web Audio `PannerNode`** (HRTF de graça, zero DSP à mão).
- **2D basta** (altura irrelevante no Palworld); o `z` trafega mas quase não pesa.
- **Escala**: posições em unidades Unreal (**cm**). 1 m ≈ 100 u. Alcance ~50 m ≈ 5000 u, configurável.
- **Transporte mod→companion por arquivo** (UE4SS não traz LuaSocket); o companion lê direto.

## Plano B documentado

A REST API do servidor expõe `location_x`/`location_y` por jogador (server-side, roda no
Linux, sem UE4SS) — porém 2D e com ~1 s de polling. O dono escolheu o mod UE4SS (sensação
in-game + direção); a rota REST virou justamente a **fonte autoritativa do anti-spoof**.
