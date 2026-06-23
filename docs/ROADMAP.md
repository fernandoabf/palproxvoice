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
- **Instalador único** ✓ — `PalProxVoice-Installer.zip`: acha o Palworld (Steam auto ou pergunta), instala UE4SS+mod+companion+config e configura **auto-start**. Você preenche o `config.json` e distribui.
- **Nomes reais** _(depois)_ — o mod escreve o nome do player; peers aparecem com nome. Mais simples que REST.
- **Auto-detectar IP** ✓ _(Direct Connect)_ — companion lê `InputIPAddress` do `GameUserSettings.ini` → conecta em `ws://<IP>:8765`. Só Direct Connect (lista do Steam não atualiza esse campo; senha não fica salva, sem `Pal.log`); **fallback = seletor manual**.
- **REST API** — _ressuscitada_, mas como fonte de **anti-spoof** na V1.5 (reconciliação `verify` / autoritativo `strict`) e **presença/IP-match**, não como posição primária. Ver a seção de releases (V1 → V1.5 → V2) abaixo.

## Próximas releases — V1 → V1.5 → V2

Os três desejos (áudio **3D direcional** · **distribuição fácil** · **anti-spoof**)
brigam entre si; a estratégia entrega em três degraus.

### V1 — auto-connect pelo IP do servidor atual

**Objetivo:** o companion descobre **em qual servidor o jogador está** (pelo IP) e
conecta no voz daquele IP **sozinho** — sem o amigo digitar nada.

- Fontes do IP (já em [`companion/serverdetect.go`](../companion/serverdetect.go), ordem **live → save → ini**):
  1. `GameServerIPLive` — socket/arquivo do mod C++ (sessão **atual**).
  2. `GameServerIPFromSave` — `PalOptionSaveGame` (pega Direct Connect **e** join pela lista do Steam). ✅ testado.
  3. `GameServerIP` — `GameUserSettings.ini` (só Direct Connect).
- **Descoberta do voz:** convenção `IP-do-jogo : 8765` (`AutoPort`); override = lista manual no `config.json`.
- **Consolidação:** mod C++ ([`mod-live/`](../mod-live/)) servindo posição+IP por **socket local (SSE)**, aposentando Lua+bridge+txt.
- **Limite honesto:** sem anti-spoof ainda — posição é client-reported (confiança).

### V1.5 — anti-spoof (o que combinamos)

Servidor de voz **agnóstico de fonte** — posição pode vir do cliente OU do servidor,
mesma forma. É a fundação que a V2 só pluga.

```
join (companion → voz):    { userId }                  # identidade
posição (qualquer fonte):  { userId, x, y, z, yaw }
config do servidor de voz: { voicePort: 8765, authoritativeMode: "off"|"verify"|"strict" }
```

- **Identidade = `userId` do Palworld** (nativo da REST e do servidor; embute SteamID). Fallback = id de sessão. ⚠️ validar se o mod lê o próprio `userId`.
- **Anti-impersonação = IP-match**: a REST devolve `ip` por jogador; o voz casa com o IP da conexão. Hardening futuro: Steam ticket.
- **`Z` (altura) trafega mas é irrelevante** (2D basta).

Modos do `authoritativeMode`:

| Modo | Posição usada | Pega spoof? | Latência | Quando |
|---|---|---|---|---|
| `off` | cliente | ❌ | baixa | início, confiança cega |
| `verify` | **cliente** (rápido) + reconcilia com REST, ban se divergir | ✅ mentira grande, ~1-2 s p/ pegar | baixa | **recomendado** (Linux nativo, sem Proton) |
| `strict` | **REST** autoritativo | ✅ zero-leak | ~1 s | paranoia / sem mod no cliente |

Guard-rails do `verify` (senão bane jogador honesto):
- tolerância = `vel_máx_plausível × atraso_REST + margem` (montaria/planador não podem dar falso positivo)
- divergência **sustentada** (N polls ~3-5 s), não única — fast-travel se resolve sozinho
- nunca banir em erro/lag da REST
- ban escalonado por `userId`+`ip`

### V2 — servidor (100% server-side) — *o mod a criar*

**Objetivo:** posição **e yaw** vêm do servidor, autoritativos. **Cliente não precisa de UE4SS** — só o companion (áudio) + identidade.

- **Mod server-side em UE4SS**, rodando o **executável Windows do servidor sob Proton/Wine** no host Linux (ver [palhub-server](https://github.com/Dekita/palhub-server)). **Não existe UE4SS nativo no Linux.**
- O mod **itera todos os `PlayerController`/`Pawn`** e lê `posição + yaw` de cada um → empurra pro servidor de voz (push, alta frequência), chaveado por `userId`.
- Servidor de voz em `authoritativeMode: "strict"` consumindo essa fonte → spoof de posição **morre** (cliente nunca afirma posição).
- **Ganhos:** anti-spoof real + **zero instalação no cliente** (some a maior fricção de distribuição).
- **Custos/riscos:** operador roda Windows-sob-Proton (mais RAM, menos estável que nativo — testar fora do OurWorld primeiro); UE4SS server-side pode ser chato ([AOB falhando relatado](https://github.com/UE4SS-RE/RE-UE4SS/issues/645)); quebra em update grande (1.0 sai 2026-07-10).
- **Correlação:** `userId` in-game ↔ peer de voz (mesma chave da V1 + IP-match).

### Tarefas de validação (antes de codar fundo)

- [ ] mod lê o **próprio `userId`** (probe in-game, como fizemos pra posição)
- [ ] V2: confirmar UE4SS server-side sob Proton no Palworld (AOB / `StaticConstructObject`)
- [ ] V2: ler `yaw` (control rotation) de cada player no servidor

## Backlog — canais de voz (aprendizado do concorrente PalVoice)

O PalVoice tem **proximidade + global + guilda**; nós só temos proximidade (pool único).
O pedido priorizado é o **canal de guilda GLOBAL**.

### Canal de guilda (global) — _planejado_

**Objetivo:** membros da **mesma guilda** se ouvem em **qualquer distância** (voz
global entre a guilda), **por cima** da proximidade. Ex.: a galera espalhada pelo
mapa continua se falando; quem não é da guilda só entra pela proximidade.

- **Mix com a proximidade:** a guilda toca num barramento **global** (sem atenuar
  por distância — volume fixo, sem HRTF ou com leve pan opcional); não-guilda
  continua no caminho 3D/proximidade. Toggle no companion pra mutar o canal global.
- **De onde vem a guilda (a decidir):**
  1. **REST** — checar se `/v1/api/...` expõe guild id por jogador (o `/players`
     que medimos **não** trazia guilda; ver se há endpoint/campo). Se sim, o
     servidor de voz agrupa por guild id → zero config.
  2. **mod** — ler o guild id in-game (probe, como userId/posição) e mandar no join.
  3. **manual / código** — "canal de guilda" por um código compartilhado, desacoplado
     da guilda do jogo (mais simples; bom fallback enquanto 1/2 não estão prontos).
- **Servidor:** o SFU já faz broadcast; é taguear o áudio por **canal** (proximidade
  vs guilda-global) e rotear. Encaixa na mesma base agnóstica da V1.5.
- **Anti-spoof:** entrar numa guilda-global de quem você não é membro é um vetor —
  validar a associação pela fonte autoritativa (REST/mod), não pelo que o cliente diz.

### Outros canais (depois, menor prioridade)

- **Global (push-to-talk)** — falar pra todo mundo do servidor com uma tecla.
- **Party/grupo** — canal por código pra um subconjunto, sem ser a guilda inteira.

## Decisões travadas

- **Alpha = checkpoint**; em seguida a companion (Fase 1 + **config/auto-connect**). O teste com a galera acontece já com a companion (sem fase de teste-primeiro).
- **Sem sala** · **SFU** · **Web Audio/HRTF** · transporte mod→companion por **arquivo** (UE4SS não tem LuaSocket).
- UI de config mora no **companion (overlay)**, nunca dentro do Palworld (frágil; 1.0 sai 2026-07-10).
- Build do companion: **local no Windows** + **GitHub Actions no release** (tag `v*`) gerando o `.exe`, configurado pro free tier.

Detalhe de como cada peça funciona: [ARCHITECTURE.md](ARCHITECTURE.md).
