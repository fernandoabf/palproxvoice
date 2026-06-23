# Servidor de voz (PalProxVoice)

SFU de áudio + relay de posição, em Go (pion/webrtc). Cada cliente manda o mic
**uma vez** e o servidor repassa pra todos os outros; a posição também é
repassada, pro companion espacializar. **Sem sala**: pool único + senha.

## Rodar

```bash
# na raiz do repo:
cp .env.example .env       # VOICE_PASSWORD, PUBLIC_IP
docker compose up -d --build
# ou direto:
cd server && VOICE_PASSWORD=test go run .
```

## Configuração (variáveis de ambiente)

| Var | Padrão | O que faz |
|-----|--------|-----------|
| `VOICE_PASSWORD` | _(vazio)_ | senha de entrada. Vazio = sem senha (passwordless). |
| `PUBLIC_IP` | _(vazio)_ | IP público anunciado pro áudio (NAT 1:1, sem TURN). **Obrigatório na VPS.** |
| `WS_PORT` | `8080` | porta da sinalização HTTP/WebSocket. |
| `SERVER_NAME` | `PalProxVoice` | nome mostrado no companion (evento `serverinfo`). |
| `VOICE_RANGE` | `50` | alcance recomendado da voz, em metros. |

Mídia (áudio) usa **UDP 50000–50010** — abra no firewall. A sinalização é a
`WS_PORT`; o companion auto-conecta em `IP:8765` por padrão, então mapeie o host
`8765` → container `8080` (ou ajuste o `autoPort` do companion).

## Protocolo (WebSocket em `/ws`)

JSON `{ event, data, id? }`. Handshake e relay:

| Direção | event | data | quando |
|---------|-------|------|--------|
| cliente → server | `auth` | senha | **1ª mensagem** (obrigatória) |
| server → cliente | `error` | motivo | auth falhou → fecha |
| server → cliente | `hello` | id do peer | autenticado |
| server → cliente | `serverinfo` | `{name, range}` | logo após o hello |
| ambos | `offer` / `answer` / `candidate` | SDP/ICE | negociação WebRTC |
| cliente → server | `pos` | `"x,y,z,yaw"` | posição do jogador |
| server → cliente | `pos` | `"x,y,z,yaw"` (+`id`) | posição de **outro** peer |

Cada track de áudio sai com `StreamID = id do peer`, pro cliente casar áudio ↔ posição.

> **V1.5** vai estender isto pra protocolo agnóstico de fonte (`{userId,...}` +
> `authoritativeMode`). Ver [../docs/ROADMAP.md](../docs/ROADMAP.md).

A pasta `web/` é um **cliente de teste** servido em `/` — abre 2 abas e anda com
WASD pra exercitar SFU + espacialização sem o jogo.
