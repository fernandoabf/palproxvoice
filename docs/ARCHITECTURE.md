# Arquitetura — PalProxVoice

Voz por **proximidade** self-hosted pro Palworld (tipo o Simple Voice Chat do
Minecraft). Os jogadores se ouvem conforme a **distância no mundo do jogo**.
Tudo roda na infra do dono (VPS); nada de serviço de terceiro.

## Por que esse desenho

O servidor dedicado de Palworld roda em **Linux**, mas o **UE4SS** (única forma
de rodar lógica dentro do jogo) é **só Windows** → não dá pra ser um mod
server-side. Então: **mod fino no cliente** lê a posição, e um **backend de voz
self-hosted** faz o resto. A parte acoplada ao jogo fica mínima e isolada (o
Palworld 1.0 sai em 2026-07-10 e updates grandes quebram mods de UE4SS).

## Três componentes

| # | Componente | Onde | O que faz | Status |
|---|-----------|------|-----------|--------|
| M1 | **Mod de posição** (`mod/`) | cliente Windows | UE4SS Lua lê `X,Y,Z` + `Yaw` ~20 Hz, escreve CSV `"x,y,z,yaw"` em `%TEMP%\palproxvoice_pos.txt` (UE4SS não traz LuaSocket → arquivo, não UDP) | feito (falta validar no jogo) |
| M2 | **Servidor de voz** (`server/`) | VPS | Go + pion/webrtc. SFU: cada mic sobe uma vez, repassa pra todos. Relay de posição por ws. Sem sala. | feito e compila |
| M3 | **Companion** (`server/web/`) | PC de cada jogador | recebe áudio + posições, espacializa em 3D (Web Audio `PannerNode`/HRTF) | feito no navegador p/ teste; falta empacotar nativo |

No M2, o `StreamID` de cada track de áudio = id do peer que enviou, pro cliente
correlacionar **áudio ↔ posição**.

## Decisões travadas

- **Sem sala** — pool único + senha compartilhada + proximidade decide quem você
  ouve. Sem canal, sem "todo mundo se ouve".
- **SFU** no VPS (não mesh P2P) — mais simples e confiável p/ ≤8 players, sem TURN.
- **Áudio 3D via Web Audio `PannerNode`** (HRTF de graça, zero DSP escrito à mão).
- **2D basta** (altura é irrelevante no Palworld); o `z` trafega mas quase não pesa.
- **Escala**: posições em unidades Unreal (**cm**). 1 m ≈ 100 u. Alcance de voz
  inicial ~50 m ≈ 5000 u — manter **configurável**.

## A escada de testes (do mais fácil pro mais real)

1. **Navegador, local** — `./local.sh` sobe o voice; 2 abas, anda com WASD,
   ouve a proximidade. Testa M2 (relay) + M3 (espacialização) sem jogo nenhum.
2. **Palworld local** — mesmo `./local.sh` sobe um servidor de Palworld de teste
   no localhost (mundo novo, ou o último backup do R2 se houver chaves).
3. **VPS** — voice atrás de TLS + Palworld de teste com o mundo real restaurado.
4. **No jogo (M1)** — o mod real alimentando posição de dentro do Palworld.
   É o **último** e mais real: prova a ponta incerta (ler pos+dir do jogo).

## Pendências (próximo)

- **Empacotar num app único** (opcional, sugestão **Wails**): hoje o companion é
  navegador + `bridge/` (exe Go que lê `%TEMP%\palproxvoice_pos.txt` e serve em
  `http://127.0.0.1:47475/pos`; a página puxa de lá). Funciona ponta-a-ponta;
  Wails só juntaria os dois numa janela só + push-to-talk pra galera usar fácil.
- **M4 — polimento**: push-to-talk, tecla in-game, auto-conectar, VAD, mute,
  config de alcance/volume, instalador.

## Plano B documentado

A REST API do servidor expõe `location_x`/`location_y` por jogador (server-side,
roda no Linux, sem UE4SS) — porém 2D e com ~1 s de polling. O dono escolheu o
mod UE4SS (sensação in-game + direção). A rota REST fica como alternativa.
