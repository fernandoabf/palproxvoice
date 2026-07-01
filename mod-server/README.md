# PalProxVoice — mod SERVER-SIDE (V2)

Mod **Lua** que roda DENTRO do servidor dedicado do Palworld (via UE4SS). É o **V2**
(estilo Simple Voice Chat): o servidor lê pos+yaw+FGuid **autoritativos** de TODOS os
players (a 5 Hz) e grava num arquivo que o **servidor de voz co-locado** consome →
delay mínimo, anti-spoof de graça (o cliente nunca afirma posição). **É Lua → não tem o
muro do Epic/C++ (UEPseudo).**

O `main.lua` produz `C:\Users\Public\palproxvoice_players.txt` (1 linha/player:
`fguid,x,y,z,yaw`) e loga a contagem no `UE4SS.log`. A ponte é por **arquivo** porque o
Lua do UE4SS não tem socket; o voz lê esse arquivo no mesmo host.

> ⚠️ Teste primeiro num **host descartável**, não na produção. O passo que ainda não foi
> provado no teu ambiente é o **UE4SS subir no dedicado** (no Linux = Proton).

## Pré-requisitos importantes
- **NÃO use o UE4SS v3.0.1** que o instalador do cliente traz no servidor — é a versão do
  bug de AOB (#645). Use o **`Okaetsu/RE-UE4SS`**, tag **`experimental-palworld`**
  (`UE4SS-Palworld.zip` p/ rodar, `..._zDev.zip` p/ desenvolver). Ele traz o
  `MemberVariableLayout.ini` pros edits de engine da Pocketpair.
- ⏰ **Palworld 1.0 sai 2026-07-10** e provavelmente quebra mods no day-one. Não teste na
  semana do lançamento; re-valide depois com a versão nova do Okaetsu.

## Onde colocar o mod
Em qualquer um dos casos, o mod vai na pasta de Mods do UE4SS do **servidor**:
```
Pal/Binaries/Win64/ue4ss/Mods/PalProxVoiceServer/
  enabled.txt
  scripts/main.lua      <- este mod
```
(copie a pasta `PalProxVoiceServer/` deste repo pra lá.)

## A) Servidor WINDOWS (caminho fácil)
1. Instale o dedicated server (SteamCMD, app `2394010`) → `Pal\Binaries\Win64`.
2. Extraia o **UE4SS Okaetsu (experimental-palworld)**: `dwmapi.dll` + pasta `ue4ss/` dentro de `Pal\Binaries\Win64`. (UE4SS 3.0+ auto-injeta via `dwmapi.dll`, sem injector externo.)
3. Copie `PalProxVoiceServer/` pra `Pal\Binaries\Win64\ue4ss\Mods\`.
4. Suba `PalServer-Win64-Shipping-Cmd.exe`.

## B) Servidor LINUX (via Proton — teu caso, é o frágil)
Mecânica provada pelo `Dekita/palhub-server` (copie a mecânica, **não** as versões dele):
1. `steamcmd +@sSteamCmdForcePlatformType windows +force_install_dir <dir> +login anonymous +app_update 2394010 validate +quit` (puxa o depot **Windows**).
2. **GE-Proton** recente (compatibilitytools.d). Semeie o prefixo (`cp -r <proton>/files/share/default_pfx` → `steamapps/compatdata/2394010`) e sete `STEAM_COMPAT_CLIENT_INSTALL_PATH`, `STEAM_COMPAT_DATA_PATH=<...>/compatdata/2394010`, `PROTON=<...>/proton`.
3. Injete `dwmapi.dll` + `ue4ss/` (Okaetsu) em `Pal/Binaries/Win64`. Copie o mod pra `ue4ss/Mods/`.
4. **GOTCHA crítico (faz ou quebra):** entre na pasta e lance por nome **RELATIVO** —
   caminho absoluto e `cd` da raiz **FALHAM** (o Proton resolve a `dwmapi.dll` pelo CWD):
   ```
   cd "<server>/Pal/Binaries/Win64"
   proton run ./PalServer-Win64-Shipping-Cmd.exe -port=8211
   ```

## ✅ Critério de SUCESSO
No log **`Pal/Binaries/Win64/ue4ss/UE4SS.log`** (o servidor é headless, sem console na tela):
1. Linhas do próprio UE4SS de **AOB scan OK** (`StaticConstructObject` resolvido) — se falhar aqui, é versão errada do UE4SS (use o Okaetsu).
2. `[PPV-SRV] mod server-side V2 carregado` e `hook de join registrado`.
3. Entre com **1 cliente** e confira:
   - o cliente **conecta normal** (sanity-check do bug antigo #452 de recusa de conexão);
   - aparece `[PPV-SRV] players autoritativos = 1` + uma linha `<fguid>,x,y,z,yaw`;
   - o arquivo **`C:\Users\Public\palproxvoice_players.txt`** existe com 1 linha por player.
     (sob Proton: dentro do prefixo Wine, `.../compatdata/2394010/pfx/drive_c/users/Public/`).

Isso prova a fundação do V2: **Lua roda no servidor + lê pos/identidade autoritativa de cada player + produz o feed.** Próximo passo: o servidor de voz consumir esse arquivo (em vez da posição que o cliente reporta) e, opcionalmente, o mod anunciar o endereço do voz via RPC (`SendScreenLogToClient` — já provado no cliente).

## Se der ruim
- AOB falhou → versão do UE4SS errada (use Okaetsu experimental-palworld) ou Palworld atualizou.
- Sob Proton o UE4SS carrega mas o mod não → checar o lançamento relativo (passo B.4); ver RE-UE4SS #1189.
- Não rode 2 UE4SS ao mesmo tempo (Workshop + manual) → crasha.
