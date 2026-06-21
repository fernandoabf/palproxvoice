# PalProxVoice

Voz por proximidade pro Palworld. Sem sala: pool único + senha + proximidade.
Desenho completo em [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

```
server/              M2  servidor de voz Go (SFU + relay de posicao) + Dockerfile
  main.go
  web/index.html     M3  companion no navegador — espacializacao 3D (HRTF)
mod/                 M1  mod UE4SS (Lua) — le posicao+yaw e escreve num arquivo
  PalProxVoice/      ......copia essa pasta pro UE4SS
  receiver.py/.ps1   ......teste do M1 (le o arquivo de posicao)
bridge/              ponte: le o arquivo do mod e serve a posicao real pro navegador
docker-compose.yml             voice
docker-compose.palworld-test.yml  + restore.sh   palworld de teste
local.sh             sobe tudo local
```

## Rodar TUDO local (staging) — um comando

Sobe voice **e** o Palworld de teste no localhost:
```
cp .env.example .env        # VOICE_PASSWORD=test, PUBLIC_IP vazio, R2 vazio
./local.sh                  # = up -d --build (voice + palworld)
```
- **Voice/proximidade:** abre `http://localhost:8088` em **2 abas** (de fone!),
  mesma senha, **Entrar**. Move com `W A S D`, gira com `← →`. Aproxima/afasta/vira
  numa aba e ouve a outra mudar de lado e volume → proximidade 3D funcionando.
  (Porta = `HTTP_PORT` do `.env`. No **WSL2** use **8088**: a 8080 do host
  costuma estar tomada pelo Windows e o docker nem binda.)
- **Acessar de outra máquina (LAN):** `http://IP_DA_MAQUINA:8088` e põe esse
  mesmo IP em `PUBLIC_IP` no `.env` (o áudio precisa dele pra achar o caminho de
  volta). Libera 8088/TCP + UDP 50000–50010 no firewall do Windows.
- **Palworld:** Direct Connect em `localhost:8222`, senha `teste123`.
  Sem chaves R2 no `.env` → sobe **mundo novo**. Com chaves → restaura o último backup.
- Derrubar: `./local.sh down`  ·  logs: `./local.sh logs -f`

> ⚠️ O servidor Palworld baixa ~8–16 GB e pede RAM (~8 GB+). Só o voice é leve:
> pra testar só a proximidade, `docker compose up --build` (sem o palworld).

## M1 — o mod no jogo (Windows)

O passo local acima já testa M2+M3 sem o jogo. O M1 conecta o **jogo real**:
1. Instala o **UE4SS v3.0.1** ([releases](https://github.com/UE4SS-RE/RE-UE4SS/releases),
   zip padrão) — extrai `dwmapi.dll` + `UE4SS.dll` + `UE4SS-settings.ini` + `Mods/`
   dentro de `Pal/Binaries/Win64/` (Steam) ou `Pal/Binaries/WinGDK/` (Game Pass/GDK).
2. Copia `mod/PalProxVoice/` pra `...Binaries/<Win64|WinGDK>/Mods/PalProxVoice/`.
   (O `enabled.txt` já liga o mod. **Layout flat** do v3.0.1 — `Mods/` ao lado do dll,
   *não* dentro de `ue4ss/`.)
3. Entra em qualquer mundo e anda. **Valida de dois jeitos:**
   - **Console do UE4SS:** sai `[PalProxVoice] x,y,z,yaw` ~1x/s.
   - **Receiver** (lê o arquivo que o mod escreve em `%TEMP%\palproxvoice_pos.txt`):
     `powershell -ExecutionPolicy Bypass -File receiver.ps1`

> Transporte é por **arquivo** (`%TEMP%\palproxvoice_pos.txt`), não UDP — o UE4SS
> não traz LuaSocket. O mod é 100% **client-side**: o servidor (qualquer mundo) só
> serve pra você andar, não participa do M1.

## Posição real do jogo no companion (bridge)

O navegador não lê arquivo, então a **bridge** (`bridge/palproxvoice-bridge.exe`)
lê `%TEMP%\palproxvoice_pos.txt` e serve em `http://127.0.0.1:47475/pos`. A página
puxa de lá e usa a posição do **jogo** no lugar do WASD.

1. Com o jogo aberto (mod rodando), roda o exe: duplo-clique em
   `palproxvoice-bridge.exe` (ou no terminal). Deixa aberto.
2. Abre a página do companion (`http://localhost:8088` local, ou a URL da VPS).
3. O status mostra **`[posicao: JOGO]`** e os números seguem teu personagem.
4. **Entrar** liga o áudio. Com um amigo conectado → voz por proximidade real.

> Recompila o exe (de qualquer máquina com Docker):
> `docker run --rm -v "$PWD/bridge":/src -w /src -e GOOS=windows -e GOARCH=amd64 golang:1.22 go build -o palproxvoice-bridge.exe .`

> Se o log do jogo disser que **LuaSocket nao foi achado**, me avisa — passo o
> fallback por arquivo.

## VPS (Dokploy)

- **Voice:** `.env` com `VOICE_PASSWORD` forte + `PUBLIC_IP=69.62.88.69`, depois
  `docker compose up -d --build`. Põe a 8080 atrás de domínio com **TLS** (o browser
  só libera o mic em HTTPS) e abre **UDP 50000–50010** no firewall.
- **Palworld de teste:** `.env` com as chaves R2 **novas**, depois
  `docker compose -f docker-compose.palworld-test.yml up -d`. Abre UDP 8222.
  Cada `up` re-restaura o mundo com o backup mais novo (de propósito).
