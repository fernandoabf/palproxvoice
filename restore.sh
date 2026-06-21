#!/bin/sh
# Baixa o ultimo backup .tar.gz do bucket R2 "ourworld" e restaura
# o mundo no volume de TESTE, antes do servidor subir.
set -e

# LOCAL sem chaves R2: pula o restore e sobe mundo novo (nao bloqueia o stack).
if [ -z "$RCLONE_CONFIG_R2_ACCESS_KEY_ID" ] || [ -z "$RCLONE_CONFIG_R2_SECRET_ACCESS_KEY" ]; then
  echo "[restore] sem chaves R2 -> pulando restore, Palworld sobe com mundo NOVO."
  exit 0
fi

echo "[restore] procurando o ultimo backup no R2..."
# lista os .tar.gz com modtime, ordena, pega o mais novo
LATEST=$(rclone lsf r2:ourworld --files-only --include "*.tar.gz" --format "tp" | sort | tail -n1 | cut -d';' -f2-)
if [ -z "$LATEST" ]; then
  echo "[restore] ERRO: nenhum backup .tar.gz encontrado no bucket ourworld."
  exit 1
fi
echo "[restore] mais novo = $LATEST"

echo "[restore] baixando..."
rclone copyto "r2:ourworld/$LATEST" /tmp/latest.tar.gz

echo "[restore] extraindo..."
rm -rf /tmp/restore && mkdir -p /tmp/restore
tar -xzf /tmp/latest.tar.gz -C /tmp/restore

# acha o Level.sav do mundo (ignora copias dentro de subpasta /backup)
LEVEL=$(find /tmp/restore -name Level.sav | grep -vi '/backup' | head -n1)
[ -z "$LEVEL" ] && LEVEL=$(find /tmp/restore -name Level.sav | head -n1)
if [ -z "$LEVEL" ]; then
  echo "[restore] ERRO: Level.sav nao encontrado dentro do backup. Estrutura extraida:"
  find /tmp/restore -maxdepth 4 -type d
  exit 1
fi
WORLD=$(dirname "$LEVEL")
echo "[restore] mundo encontrado em: $WORLD"

# coloca o mundo no lugar certo do volume de teste
mkdir -p /palworld/Pal/Saved/SaveGames
rm -rf /palworld/Pal/Saved/SaveGames/0
mkdir -p /palworld/Pal/Saved/SaveGames/0
cp -a "$WORLD" /palworld/Pal/Saved/SaveGames/0/

# o servidor roda como uid 1000 (PUID/PGID), entao tem que ser dono dos arquivos
chown -R 1000:1000 /palworld/Pal/Saved

echo "[restore] OK! mundo restaurado:"
ls -la /palworld/Pal/Saved/SaveGames/0/
