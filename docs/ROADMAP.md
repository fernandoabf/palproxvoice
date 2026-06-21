# Roadmap — PalProxVoice

## Estado atual — `v0.1.0-alpha`

Protótipo funcional ponta-a-ponta, validado:

- **M1** — mod UE4SS lê posição+direção do Palworld (build GDK) e escreve em arquivo. ✅ validado no jogo.
- **M2** — servidor de voz (Go/pion, SFU + relay de posição). ✅ compila e roda.
- **M3** — espacialização 3D (Web Audio/HRTF) no navegador, alimentada pela posição real via **bridge**. ✅ posição real chega no companion.

Hoje rodam **3 peças soltas**: jogo+mod · bridge · navegador. Este alpha é o
**checkpoint** antes de construir a companion, que junta tudo num app só.

## Próximo — o companion (vira o produto)

Absorve a bridge e o navegador num **app único** (Wails — Go + webview). Build
**no Windows** (precisa WebView2), feito pelo dono.

| Fase | Ideia | Status |
|------|-------|--------|
| 1 | **Consolidar**: backend lê o arquivo direto (bridge some) + janela própria. Reaproveita o áudio que já funciona. | feito ✓ (validado in-game) |
| 2 | **Config + auto-connect** (servidor, senha, alcance, volume; conecta sozinho). | feito ✓ (validado in-game) |
| 3 | **Overlay** (always-on-top, quem fala, push-to-talk, mute) — o feeling Simple Voice Chat. | a fazer |
| 4 | **Empacotar** (.exe único, auto-start opcional, instalar pra galera). | a fazer |

## Decisões travadas

- **Alpha = checkpoint**; em seguida a companion (Fase 1 + **config/auto-connect**). O teste com a galera acontece já com a companion (sem fase de teste-primeiro).
- **Sem sala** · **SFU** · **Web Audio/HRTF** · transporte mod→companion por **arquivo** (UE4SS não tem LuaSocket).
- UI de config mora no **companion (overlay)**, nunca dentro do Palworld (frágil; 1.0 sai 2026-07-10).
- Build do companion: **local no Windows** + **GitHub Actions no release** (tag `v*`) gerando o `.exe`, configurado pro free tier.

Detalhe de como cada peça funciona: [ARCHITECTURE.md](ARCHITECTURE.md).
