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

echo "[ppv-v2] 4/5 preparando Proton + feed compartilhado..."
export STEAM_COMPAT_CLIENT_INSTALL_PATH="${STEAM_COMPAT_CLIENT_INSTALL_PATH:-/root/.steam}"
export STEAM_COMPAT_DATA_PATH="${STEAM_COMPAT_DATA_PATH:-$PAL_DIR/compatdata/$APPID}"
PROTON="${PROTON:-/proton/proton}"
mkdir -p "$STEAM_COMPAT_DATA_PATH" "$STEAM_COMPAT_CLIENT_INSTALL_PATH" "$FEED_DIR"
# semeia o prefixo do Wine (1ª vez)
if [ ! -d "$STEAM_COMPAT_DATA_PATH/pfx" ]; then
  cp -r "$(dirname "$PROTON")/files/share/default_pfx" "$STEAM_COMPAT_DATA_PATH/pfx" 2>/dev/null || true
fi
# o mod escreve em C:\Users\Public (Wine) -> aponta esse dir pro volume /feed,
# assim o container do voz lê /feed/palproxvoice_players.txt
PUBDIR="$STEAM_COMPAT_DATA_PATH/pfx/drive_c/users/Public"
mkdir -p "$(dirname "$PUBDIR")"
rm -rf "$PUBDIR" && ln -sfn "$FEED_DIR" "$PUBDIR"

echo "[ppv-v2] 5/5 lançando o servidor sob Proton (cd Win64 + nome relativo)..."
cd "$WIN64"   # CRÍTICO: lançar por nome RELATIVO ou o Proton não resolve a dwmapi.dll
exec "$PROTON" run ./PalServer-Win64-Shipping-Cmd.exe \
  -port="$PORT" -QueryPort="$QUERY_PORT" -RCONPort="$RCON_PORT" -RESTAPIPort="$REST_PORT" \
  -ServerName="OurWorld V2 TEST" -AdminPassword="$ADMIN_PASSWORD" \
  -useperfthreads -NoAsyncLoadingThread -UseMultithreadForDS
