# Roadmap — PalProxVoice

## Estado atual (`develop`)

Produto funcionando ponta-a-ponta, validado com pessoas reais pela internet — bem além do MVP:

| Camada | O que | Status |
|---|---|:---:|
| **Base + V1** | voz 3D + auto-connect (IP + **probe da porta**, zero digitação) | 🟢 validado in-game |
| **V1.5** | anti-spoof por REST (`verify`/`strict`; mantém Z+yaw do cliente) | 🟢 codado (off por padrão) |
| **V2** | mod UE4SS no servidor (Proton) — pos+yaw+FGuid autoritativos @5Hz | 🟡 **server-side funciona** (server escuta + mod carrega + feed grava; voz e2e não testada) |
| **Canais de voz** | proximidade / guild / global + **hotkey global Alt+V** | 🟢 construído |
| **ETW** | IP do server em tempo real (UDP do kernel) | 🟢 construído (precisa admin) |
| **Game Pass (WinGDK)** | UE4SS + mod cliente rodam nele | 🟢 confirmado |
| **Auto-guild** | plumbing (companion) + probe de descoberta | 🟡 falta a leitura real (logs do `probe_guild.lua`) |
| **V3 (Linux nativo)** | leitor de memória externo (sem Proton/Wine) | 🔬 só scaffold + plano |

Distribuição: release com `palproxvoice.exe` + bundle `PalProxVoice-UE4SS.zip` (UE4SS + mod).

> As seções abaixo são o **planejamento histórico** (como chegamos aqui) + o que ainda falta. Para o estado real, use a tabela acima.

## M4 — polimento (a fazer)

Núcleo pronto; tudo abaixo é conforto:

- **Overlay** — always-on-top, "quem está falando", push-to-talk, mute.
- **Auto-start escondido** — companion sobe sozinho; jogador nunca abre o `.exe`.
- **Config embutida (sem o amigo configurar)** — companion lê um `config.json` ao lado do `.exe`; você preenche uma vez e distribui junto, amigo só abre e conecta.
- **Fechar a voz ao sair do servidor** — posição parada por N s → desconecta sozinho.
- **Escolher microfone + saída de áudio** — na config.
- **Instalador único** ✓ — `PalProxVoice-Setup.exe`: acha o Palworld (Steam auto ou pergunta), instala UE4SS+mod+companion+config e configura **auto-start**. Você preenche o `config.json` e distribui.
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
- **Consolidação:** mod C++ ([`mod-live/`](../mod-live/)) servindo posição+IP por **socket local (SSE)**, aposentando Lua+txt.
- **Limite honesto:** posição é client-reported (confiança) — resolvido na V1.5 abaixo.

### V1.5 — anti-spoof ✅ **implementado** (off por padrão)

Servidor de voz **agnóstico de fonte** — posição pode vir do cliente OU do servidor,
mesma forma. É a fundação que a V2 só pluga. Implementado em `server/antispoof.go`;
correlação por **FGuid** (mod escreve, companion manda no auth/`identify`) → IP →
IP+proximidade. Deploy: [ANTI-SPOOF-DEPLOY.md](ANTI-SPOOF-DEPLOY.md).

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

### V2 — servidor (100% server-side) 🟡 **server-side funciona** (experimental)

> **Feito (server-side).** `mod-server/PalProxVoiceServer` roda no PalServer Windows **sob Proton** no Linux (imagem `deploy/v2-experimental/linux`), lê pos+yaw+FGuid de todos e escreve o feed `palproxvoice_players.txt` (`fguid,x,y,z,yaw`); a voz consome via `PPV_PLAYERS_FILE` (`server/antispoof_v2.go`). Toda a fragilidade do Proton foi resolvida (ver CHANGELOG). **Falta o teste de voz ponta-a-ponta com 2 players.** O plano original abaixo continua como referência.

**Objetivo:** posição **e yaw** vêm do servidor, autoritativos. **Cliente não precisa de UE4SS** — só o companion (áudio) + identidade.

- **Mod server-side em UE4SS**, rodando o **executável Windows do servidor sob Proton/Wine** no host Linux (ver [palhub-server](https://github.com/Dekita/palhub-server)). **Não existe UE4SS nativo no Linux.**
- O mod **itera todos os `PlayerController`/`Pawn`** e lê `posição + yaw` de cada um → empurra pro servidor de voz (push, alta frequência), chaveado por `userId`.
- Servidor de voz em `authoritativeMode: "strict"` consumindo essa fonte → spoof de posição **morre** (cliente nunca afirma posição).
- **Ganhos:** anti-spoof real + **zero instalação no cliente** (some a maior fricção de distribuição).
- **Custos/riscos:** operador roda Windows-sob-Proton (mais RAM, menos estável que nativo — testar fora do OurWorld primeiro); UE4SS server-side pode ser chato ([AOB falhando relatado](https://github.com/UE4SS-RE/RE-UE4SS/issues/645)); quebra em update grande (1.0 sai 2026-07-10).
- **Correlação:** `userId` in-game ↔ peer de voz (mesma chave da V1 + IP-match).

### Tarefas de validação (antes de codar fundo)

- [x] mod lê a **própria identidade** ✅ — FGuid via `ps.IndividualHandleId.PlayerUId` (probe in-game), bate com o `playerId` da REST
- [x] V2: UE4SS server-side sob Proton no Palworld ✅ — AOB casa (build Okaetsu `experimental-palworld`); server escuta + mod carrega + feed grava
- [x] V2: ler pos+`yaw`+FGuid de cada player no servidor ✅ — feed `palproxvoice_players.txt` (`fguid,x,y,z,yaw`) @5Hz
- [ ] V2: teste de voz com 2 players usando a posição autoritativa do feed

## Backlog — canais de voz (aprendizado do concorrente PalVoice)

O PalVoice tem **proximidade + global + guilda**; nós só temos proximidade (pool único).

### Canais de voz ✅ **construído** — proximidade / guild / global

> **Feito** (a versão de 3 canais). O peer manda `meta{guild,channel}`; o servidor repassa (`peermeta`); o companion mistura por canal — **proximidade = panner 3D**, **guild/global = som plano** (guild só entre a mesma guild). Troca de canal **in-game** por **hotkey global Alt+V** (`RegisterHotKey`). Guild = auto (mod, em finalização) + código manual. O plano original de **4 canais** (separar guilda-local × guilda-global) fica abaixo como evolução futura.

<details><summary>Plano original (4 canais)</summary>

Dois eixos: **quem ouve** (todos / só guilda) × **alcance** (proximidade / global):

| Canal | Quem ouve | Alcance | Áudio |
|---|---|---|---|
| **Proximidade** (aberta) | qualquer um perto | local | 3D/HRTF ← _é o que temos hoje_ |
| **Global** (aberta) | todo o servidor | qualquer distância | fixo (sem atenuar) |
| **Guilda — local** | só a guilda, quando perto | local | 3D/HRTF |
| **Guilda — global** | só a guilda | qualquer distância | fixo (sem atenuar) |

- **Jogador escolhe** qual canal usar (toggle no companion). A decidir: ouvir só um
  por vez, ou empilhar (ex.: proximidade **+** guilda-global por cima — guilda num
  barramento de volume fixo, não-guilda no caminho 3D).
- **Servidor obriga/restringe** quais canais valem (política do admin via env). Ex.:
  só proximidade; proximidade + guilda-global; desligar o global-aberto. O companion
  só mostra/permite os canais liberados pelo servidor. _(prioridade do pedido)_
- **De onde vem a guilda (a decidir):**
  1. **REST** — ver se a REST expõe guild id por jogador (o `/players` que medimos
     **não** trazia; checar endpoint/campo). Se sim, o servidor agrupa por guild id → zero config.
  2. **mod** — ler o guild id in-game (probe, como userId/posição) e mandar no join.
  3. **manual / código** — canal por código compartilhado, desacoplado da guilda do
     jogo (mais simples; bom fallback enquanto 1/2 não estão prontos).
- **Servidor (SFU):** já faz broadcast; é taguear o áudio por **canal** e rotear por
  quem-pode-ouvir. Encaixa na base agnóstica da V1.5.
- **Anti-spoof:** entrar numa guilda/global de quem não é vetor de bisbilhotice —
  validar a associação (guilda, e o "está mesmo no servidor") pela fonte autoritativa
  (REST/mod), nunca pelo que o cliente afirma.

</details>

### Push-to-talk (PTT) — _planejado_ (a infra de hotkey global já existe: Alt+V troca canal)

**Objetivo:** falar segurando uma tecla, como alternativa ao voice-activity/gate.

- **Tecla configurável** no companion (salva na config).
- **Modos de transmissão:** _voz aberta (gate)_ ↔ _PTT_ ↔ _mutado_ — escolha do jogador.
- **PTT por canal:** poder ter modos diferentes por canal — ex.: **proximidade aberta**
  (gate) e **global/guilda em PTT** (só fala quando segura), evitando spam no canal amplo.
- **Implementação:** corta/abre o envio no caminho de captura nativa (WASAPI → WS local
  → WebRTC) sem reabrir o device; reaproveita o `micGain`/gate que já existe. Tecla
  capturada pela janela do companion (overlay always-on-top) — validar captura global
  da tecla mesmo com o jogo em foco (pode exigir hook de teclado de baixo nível).
- **HUD:** indicador "transmitindo" no overlay enquanto a tecla está pressionada.

### Outros (depois, menor prioridade)

- **Party/grupo** — canal por código pra um subconjunto, sem ser a guilda inteira.

## ETW — IP do servidor em tempo real ✅ **construído**

O companion lê o UDP do processo do Palworld no kernel (`Microsoft-Windows-Kernel-Network`) via `golang-etw`, filtra pelo PID e pega o IP de destino público **com mais tráfego (bytes)** → escreve no `palproxvoice_server.txt`, que o `DetectGameServerIP()` (fonte "live") já consome com prioridade. Fecha a fonte de IP **realtime** que faltava (a `GameServerIPLive`). **Precisa de admin** (sessão ETW de kernel); sem admin, degrada pras outras fontes (save/ini). Fica ligado no startup.

## V3 — Linux nativo (sem Proton/UE4SS) — 🔬 pesquisa

A mesma posição autoritativa do V2, mas lendo direto do **PalServer nativo de Linux** (sem `.exe` Windows, sem Proton, sem Wine) — via um **leitor de memória externo** (`process_vm_readv` + AOB scan do `GUObjectArray`). Pesquisa concluída: é o único caminho que entrega pos+yaw+FGuid a 5-20Hz no nativo (REST não tem yaw/Z; Blueprint é sandbox; sniff é inviável). **Só o scaffold + plano existem** (harness `thijsvanloef`+Frida + `probe.js` de recon); o leitor de verdade é o trabalho difícil (RE iterando contra o binário ao vivo). Plano completo em `v3-linux-native/`, na branch `experimental/v3-linux-native`.

## Decisões travadas

- **Alpha = checkpoint**; em seguida a companion (Fase 1 + **config/auto-connect**). O teste com a galera acontece já com a companion (sem fase de teste-primeiro).
- **Sem sala** · **SFU** · **Web Audio/HRTF** · transporte mod→companion por **arquivo** (UE4SS não tem LuaSocket).
- UI de config mora no **companion (overlay)**, nunca dentro do Palworld (frágil; 1.0 sai 2026-07-10).
- Build do companion: **local no Windows** + **GitHub Actions no release** (tag `v*`) gerando o `.exe`, configurado pro free tier.

Detalhe de como cada peça funciona: [ARCHITECTURE.md](ARCHITECTURE.md).
