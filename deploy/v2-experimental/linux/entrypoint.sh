#!/usr/bin/env bash
# PalProxVoice V2 — entrypoint do servidor Palworld MODADO (Linux + Proton + UE4SS)
# =====================================================================================
# ⚠️ EXPERIMENTAL / NÃO TESTADO. Baseado nas mecânicas do Dekita/palhub-server.
# Roda o .exe WINDOWS do PalServer sob Proton (UE4SS é Windows-only) e:
#   1) instala/atualiza o Palworld dedicated (depot Windows via steamcmd)
#   2) AUTO-BAIXA o UE4SS (Okaetsu experimental-palworld)
#   3) instala o mod PalProxVoiceServer (montado por volume)
#   4) liga o feed compartilhado (C:\Users\Public -> /feed) pro voz ler
#   5) lança sob Proton (gotcha: cd Win64 + nome RELATIVO)
# Use num host DESCARTÁVEL primeiro. Portas DESLOCADAS pra coexistir com o prod.
set -euo pipefail

PAL_DIR="${PAL_DIR:-/palworld}"
WIN64="$PAL_DIR/Pal/Binaries/Win64"
APPID=2394010
UE4SS_URL="https://github.com/Okaetsu/RE-UE4SS/releases/download/experimental-palworld/UE4SS-Palworld.zip"
MOD_SRC="${MOD_SRC:-/mods/PalProxVoiceServer}"   # mod deste repo (montado read-only)
FEED_DIR="${FEED_DIR:-/feed}"                    # volume compartilhado com o voz

# portas DESLOCADAS (default +100 do padrão) — ajuste no compose se quiser
PORT="${PORT:-8311}"; QUERY_PORT="${QUERY_PORT:-27115}"
REST_PORT="${REST_PORT:-8312}"; RCON_PORT="${RCON_PORT:-25675}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-trocar-senha-forte}"

echo "[ppv-v2] 1/5 instalando/atualizando Palworld (depot Windows)..."
/steamcmd/steamcmd.sh +@sSteamCmdForcePlatformType windows \
  +force_install_dir "$PAL_DIR" +login anonymous +app_update "$APPID" validate +quit

echo "[ppv-v2] 2/5 baixando UE4SS (Okaetsu experimental-palworld)..."
if [ ! -f "$WIN64/dwmapi.dll" ] || [ "${UE4SS_FORCE:-0}" = "1" ]; then
  curl -fsSL -o /tmp/ue4ss.zip "$UE4SS_URL"
  unzip -o /tmp/ue4ss.zip -d "$WIN64" >/dev/null
  rm -f /tmp/ue4ss.zip
  echo "[ppv-v2]   UE4SS instalado em $WIN64 (dwmapi.dll + ue4ss/)"
else
  echo "[ppv-v2]   UE4SS já presente (UE4SS_FORCE=1 pra rebaixar)"
fi

echo "[ppv-v2] 3/5 instalando mod PalProxVoiceServer..."
MODDST="$WIN64/ue4ss/Mods/PalProxVoiceServer"
mkdir -p "$MODDST"
cp -rf "$MOD_SRC/scripts" "$MODDST/"
: > "$MODDST/enabled.txt"
# UE4SS-settings.ini p/ HEADLESS: sem isso o GUI console do UE4SS tenta criar janela
# num prefixo sem X e TRAVA o boot (RE-UE4SS #497) -> Pal.log nem e' criado. Espelha o
# unico server UE4SS-sob-Proton comprovado (NewittAll). + salvaguarda #452.
SETTINGS="$WIN64/ue4ss/UE4SS-settings.ini"
if [ -f "$SETTINGS" ]; then
  sed -i 's/^[[:space:]]*GuiConsoleEnabled[[:space:]]*=.*/GuiConsoleEnabled = 0/I'  "$SETTINGS"
  sed -i 's/^[[:space:]]*GuiConsoleVisible[[:space:]]*=.*/GuiConsoleVisible = 0/I'  "$SETTINGS"
  sed -i 's/^[[:space:]]*GraphicsAPI[[:space:]]*=.*/GraphicsAPI = dx11/I'           "$SETTINGS"
  sed -i 's/^[[:space:]]*bUseUObjectArrayCache[[:space:]]*=.*/bUseUObjectArrayCache = false/I' "$SETTINGS"
fi

echo "[ppv-v2] 4/5 preparando Proton + feed compartilhado..."
export HOME="${HOME:-/root}"
# /etc/machine-id: wine/dbus reclamam sem ele ("Failed to open /etc/machine-id")
[ -s /etc/machine-id ] || tr -d '-' < /proc/sys/kernel/random/uuid > /etc/machine-id 2>/dev/null || true
export STEAM_COMPAT_CLIENT_INSTALL_PATH="${STEAM_COMPAT_CLIENT_INSTALL_PATH:-$HOME/.steam/steam}"
export STEAM_COMPAT_DATA_PATH="${STEAM_COMPAT_DATA_PATH:-$PAL_DIR/compatdata/$APPID}"
export STEAM_COMPAT_APP_ID="$APPID" SteamAppId="$APPID" SteamGameId="$APPID"
# SO o proxy do UE4SS. NAO desligar d3d: o PalServer precisa do RHI (com d3d off ele sai cedo).
export WINEDLLOVERRIDES="${WINEDLLOVERRIDES:-dwmapi=n,b}"
# PLANO B: esync/fsync OFF -> sync server-side antigo do wine (sem eventfd/futex). O ulimit
# alto NAO resolveu o hang, entao testamos o mecanismo de sync inteiro. Bonus: a "tempestade
# de SEH" pode ser o esync falhando a criar eventfd -> com esync off deve sumir.
export PROTON_NO_ESYNC="${PROTON_NO_ESYNC:-1}"
export PROTON_NO_FSYNC="${PROTON_NO_FSYNC:-1}"
# logs QUIETOS (default): o wine grava um steam-$APPID.log gigante com o "storm" benigno de
# nomeacao de thread. P/ debugar: PROTON_LOG=1 WINEDEBUG=  no compose.
export WINEDEBUG="${WINEDEBUG:--all}"
export PROTON_LOG="${PROTON_LOG:-0}" PROTON_LOG_DIR="${PROTON_LOG_DIR:-$PAL_DIR}"
PROTON="${PROTON:-/proton/proton}"
mkdir -p "$STEAM_COMPAT_CLIENT_INSTALL_PATH" "$STEAM_COMPAT_DATA_PATH" "$FEED_DIR"

# steamclient.so: o stub steam.exe do Proton procura num caminho HARDCODED
# ($HOME/.steam/sdk{32,64}/steamclient.so — Proton #9068; IGNORA o COMPAT_CLIENT_INSTALL_PATH).
# Sem isso o PalServer crasha no boot (era o nosso loop de restart). Linka do steamcmd.
echo "[ppv-v2]   linkando steamclient.so (Proton stub: ~/.steam/sdk{32,64})..."
mkdir -p "$HOME/.steam/sdk32" "$HOME/.steam/sdk64"
sc32="$(find /steamcmd /root/.steam /root/Steam -path '*linux32*/steamclient.so' 2>/dev/null | head -n1 || true)"
sc64="$(find /steamcmd /root/.steam /root/Steam -path '*linux64*/steamclient.so' 2>/dev/null | head -n1 || true)"
if [ -n "${sc32:-}" ]; then ln -sf "$sc32" "$HOME/.steam/sdk32/steamclient.so"; echo "[ppv-v2]     sdk32 -> $sc32"; fi
if [ -n "${sc64:-}" ]; then ln -sf "$sc64" "$HOME/.steam/sdk64/steamclient.so"; echo "[ppv-v2]     sdk64 -> $sc64"; else echo "[ppv-v2]     WARN: steamclient.so linux64 nao achado — PalServer pode crashar no boot"; fi

# IMPORTANTE: NAO copiar default_pfx na mao. Isso deixa o prefixo meio-inicializado
# (a pasta pfx existe mas sem os arquivos de controle 'version'/'tracked_files') e o
# Proton crasha no upgrade: FileNotFoundError .../compatdata/2394010/tracked_files.
# Deixa o PROTON criar o prefixo do zero com um boot inicial; so depois faz o symlink.
if [ ! -f "$STEAM_COMPAT_DATA_PATH/tracked_files" ]; then
  # limpa prefixo meio-inicializado de tentativa anterior (o pfx fica no volume e o
  # Proton crasharia de novo no upgrade). So o prefixo Wine, NAO o jogo nem os saves.
  rm -rf "$STEAM_COMPAT_DATA_PATH"
  mkdir -p "$STEAM_COMPAT_DATA_PATH"
  echo "[ppv-v2]   inicializando prefixo Proton (1a vez; cria pfx + tracked_files)..."
  "$PROTON" run wineboot --init || true   # setup_prefix roda ANTES do cmd e ja cria tudo
fi

# feed: o mod escreve em C:\Users\Public (Wine). Com o prefixo ja criado, troca esse
# dir por um symlink pro volume /feed -> o container do voz le o mesmo arquivo.
PUBDIR="$STEAM_COMPAT_DATA_PATH/pfx/drive_c/users/Public"
mkdir -p "$(dirname "$PUBDIR")"
rm -rf "$PUBDIR" && ln -sfn "$FEED_DIR" "$PUBDIR"

echo "[ppv-v2] 5/5 lançando o servidor sob Proton (Xvfb :99 + cd Win64 + nome relativo)..."
# Xvfb manual: NA PRATICA, com display o PalServer chegou ate o FTcpListener (init de rede).
# SEM display + d3d off ele SAI cedo (RHI falha). Entao damos um X virtual ao Wine.
Xvfb :99 -screen 0 1024x768x24 -nolisten tcp >/dev/null 2>&1 &
export DISPLAY=:99
n=0; while [ ! -S /tmp/.X11-unix/X99 ] && [ "$n" -lt 20 ]; do n=$((n+1)); sleep 0.5; done
cd "$WIN64"   # CRÍTICO: lançar por nome RELATIVO ou o Proton não resolve a dwmapi.dll
exec "$PROTON" run ./PalServer-Win64-Shipping-Cmd.exe \
  -port="$PORT" -QueryPort="$QUERY_PORT" -RCONPort="$RCON_PORT" -RESTAPIPort="$REST_PORT" \
  -ServerName="OurWorld V2 TEST" -AdminPassword="$ADMIN_PASSWORD" \
  -useperfthreads -NoAsyncLoadingThread -UseMultithreadForDS
