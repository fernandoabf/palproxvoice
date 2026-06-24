# PalProxVoice V2 — deploy do servidor MODADO (EXPERIMENTAL)

Sobe o servidor Palworld com **UE4SS + o mod `PalProxVoiceServer`** (lê pos+yaw+FGuid
autoritativos a 20Hz → feed `palproxvoice_players.txt`) e o **servidor de voz** consumindo
esse feed (`PPV_PLAYERS_FILE`). É o V2 estilo Simple Voice Chat: posição vem do servidor,
delay mínimo, anti-spoof de graça.

> ⚠️ **EXPERIMENTAL / NÃO TESTADO.** A parte frágil é o **UE4SS no servidor** (no Linux =
> Proton). Rode num **host descartável** primeiro, não no teu Palworld de produção.
> ⏰ Palworld 1.0 (2026-07-10) provavelmente quebra mods no day-one — re-valide depois.

## Portas DESLOCADAS (pra coexistir com teu prod)
| | prod (padrão) | V2 TEST (aqui) |
|---|---|---|
| jogo (UDP) | 8211 | **8311** |
| query (UDP) | 27015 | **27115** |
| REST | 8212 | **8312** (não publicada) |
| RCON | 25575 | **25675** |
| voz (ws) | 8765 | **8766** |
| voz (mídia UDP) | 50000-50010 | **50100-50110** |

## Dois caminhos

### 🐧 Linux (Docker + Proton) — `linux/`
O UE4SS é Windows-only, então no Linux a gente roda o **.exe Windows do PalServer sob
Proton** dentro do container. O `entrypoint.sh` **auto-baixa o UE4SS** (Okaetsu
experimental-palworld), instala o mod e lança (com o gotcha do `cd Win64` + nome relativo).
```bash
docker compose -f deploy/v2-experimental/linux/docker-compose.yml up -d --build
```
Sobe **palworld-v2** (modado, portas deslocadas) + **voice-v2** (lê o feed) + volume
compartilhado `ppv_feed`. Baseado nas mecânicas do `Dekita/palhub-server` — se a base de
Proton der trabalho, considere usar a imagem dele e só montar o mod.

### 🪟 Windows (NATIVO, sem docker) — `windows/`
No Windows o UE4SS roda **nativo** (sem Proton) → **não usa docker**. Use o script:
```powershell
deploy\v2-experimental\windows\install.ps1 -ServerDir "C:\palworld-server-v2"
```
Ele baixa o UE4SS, instala o mod, e te diz como subir o `PalServer-Win64-Shipping-Cmd.exe`
com as portas deslocadas. O voz (Go) você roda apontando `PPV_PLAYERS_FILE` pro
`C:\Users\Public\palproxvoice_players.txt`.

> **Por que não tem "compose do Windows"?** Docker no Windows roda **container Linux** →
> seria o mesmo Proton do compose de Linux. A vantagem do Windows é rodar **nativo** (sem
> Proton), e aí não é docker — é o `install.ps1`. Se você REALMENTE quer um container
> Windows nativo, dá (Windows containers), mas é exótico/pesado — me fala que eu monto.

## Como o feed chega no voz
O mod escreve `C:\Users\Public\palproxvoice_players.txt`. No Linux/Proton isso é o
`drive_c/users/Public` do prefixo Wine — o `entrypoint.sh` aponta esse dir pro volume
`/feed`, e o container do voz monta o mesmo volume e lê via `PPV_PLAYERS_FILE`.

## ✅ Sucesso
- `ue4ss/UE4SS.log`: AOB scan ok + `[PPV-SRV] mod server-side V2 carregado` + (com player) `players autoritativos = N`.
- arquivo `palproxvoice_players.txt` aparece com 1 linha/player.
- conecta o companion no voz (porta 8766) → ouve com a posição **do servidor**.

## Se der ruim
- AOB falhou → UE4SS errado (use o Okaetsu) ou Palworld atualizou.
- Proton: o UE4SS carrega mas o mod não → confira o lançamento relativo (no `entrypoint.sh`); ver RE-UE4SS #1189.
- Cliente não conecta após instalar UE4SS → bug antigo #452 (deve estar resolvido; confirme).
