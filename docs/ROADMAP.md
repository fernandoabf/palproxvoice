# Roadmap — PalProxVoice

## Estado atual — `v0.2.0` (validado in-game)

Produto funcionando ponta-a-ponta, **2 pessoas reais pela internet**:

- **M1** — mod UE4SS lê posição+direção e escreve em `C:\Users\Public\palproxvoice_pos.txt`. ✅ validado no jogo.
- **M2** — servidor de voz (Go/pion, SFU + relay de posição) na VPS, atrás de TLS + UDP. ✅ no ar.
- **M3** — companion (Wails, app único) lê a posição e espacializa em 3D (Web Audio/HRTF); auto-conecta. ✅ proximidade validada in-game.

Distribuição: release com `palproxvoice.exe` + bundle `PalProxVoice-UE4SS.zip` (UE4SS v3.0.1 exato + mod).

## M4 — polimento (a fazer)

Núcleo pronto; tudo abaixo é conforto:

- **Overlay** — always-on-top, "quem está falando", push-to-talk, mute.
- **Auto-start escondido** — companion sobe sozinho; jogador nunca abre o `.exe`.
- **Config embutida (sem o amigo configurar)** — companion lê um `config.json` ao lado do `.exe`; você preenche uma vez e distribui junto, amigo só abre e conecta.
- **Fechar a voz ao sair do servidor** — posição parada por N s → desconecta sozinho.
- **Escolher microfone + saída de áudio** — na config.
- **Instalador único** — UE4SS v3.0.1 + mod + companion num clique (hoje: 2 downloads + extrair).
- **Nomes reais** _(depois)_ — o mod escreve o nome do player; peers aparecem com nome. Mais simples que REST.
- ~~Plano B (REST, sem UE4SS)~~ — **parado por enquanto**.

## Decisões travadas

- **Alpha = checkpoint**; em seguida a companion (Fase 1 + **config/auto-connect**). O teste com a galera acontece já com a companion (sem fase de teste-primeiro).
- **Sem sala** · **SFU** · **Web Audio/HRTF** · transporte mod→companion por **arquivo** (UE4SS não tem LuaSocket).
- UI de config mora no **companion (overlay)**, nunca dentro do Palworld (frágil; 1.0 sai 2026-07-10).
- Build do companion: **local no Windows** + **GitHub Actions no release** (tag `v*`) gerando o `.exe`, configurado pro free tier.

Detalhe de como cada peça funciona: [ARCHITECTURE.md](ARCHITECTURE.md).
