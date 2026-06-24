# PalProxVoice V2 — instalar UE4SS + mod no servidor Palworld NATIVO no Windows
# =====================================================================================
# No Windows o UE4SS roda NATIVO (sem Proton) -> não precisa de docker. Este script:
#   1) baixa o UE4SS (Okaetsu experimental-palworld)
#   2) extrai dwmapi.dll + ue4ss/ no Pal\Binaries\Win64 do servidor
#   3) instala o mod PalProxVoiceServer (deste repo)
# Depois você sobe o PalServer-Win64-Shipping-Cmd.exe com as portas DESLOCADAS (abaixo).
#
# Uso (PowerShell):
#   .\install.ps1 -ServerDir "C:\palworld-server-v2"
# (ServerDir = onde está/vai o dedicated server. Use um install SEPARADO do teu prod.)
param(
  [Parameter(Mandatory=$true)][string]$ServerDir,
  [string]$ModRepo = (Resolve-Path "$PSScriptRoot\..\..\..\mod-server\PalProxVoiceServer").Path
)
$ErrorActionPreference = "Stop"

$win64 = Join-Path $ServerDir "Pal\Binaries\Win64"
if (-not (Test-Path $win64)) { throw "Pal\Binaries\Win64 não existe em $ServerDir — instale o dedicated server primeiro (SteamCMD app 2394010)." }

Write-Host "[ppv-v2] baixando UE4SS (Okaetsu experimental-palworld)..."
$zip = Join-Path $env:TEMP "ue4ss-palworld.zip"
Invoke-WebRequest -Uri "https://github.com/Okaetsu/RE-UE4SS/releases/download/experimental-palworld/UE4SS-Palworld.zip" -OutFile $zip
Write-Host "[ppv-v2] extraindo em $win64 ..."
Expand-Archive -Path $zip -DestinationPath $win64 -Force
Remove-Item $zip -Force

Write-Host "[ppv-v2] instalando o mod PalProxVoiceServer..."
$moddst = Join-Path $win64 "ue4ss\Mods\PalProxVoiceServer"
New-Item -ItemType Directory -Force -Path (Join-Path $moddst "scripts") | Out-Null
Copy-Item -Recurse -Force (Join-Path $ModRepo "scripts\*") (Join-Path $moddst "scripts")
if (-not (Test-Path (Join-Path $moddst "enabled.txt"))) { New-Item -ItemType File -Path (Join-Path $moddst "enabled.txt") | Out-Null }

Write-Host ""
Write-Host "[ppv-v2] PRONTO. Confira:"
Write-Host "  - $win64\dwmapi.dll  e  $win64\ue4ss\  existem"
Write-Host "  - $moddst\scripts\main.lua  existe"
Write-Host ""
Write-Host "Suba o servidor (portas DESLOCADAS pra coexistir com o prod):"
Write-Host "  cd `"$win64`""
Write-Host "  .\PalServer-Win64-Shipping-Cmd.exe -port=8311 -QueryPort=27115 -RCONPort=25675 -RESTAPIPort=8312 -useperfthreads -NoAsyncLoadingThread -UseMultithreadForDS"
Write-Host ""
Write-Host "Sucesso = ue4ss\UE4SS.log com AOB ok + '[PPV-SRV] players autoritativos = N'"
Write-Host "e o arquivo C:\Users\Public\palproxvoice_players.txt aparecendo."
Write-Host "O servidor de voz (Go) lê esse arquivo via PPV_PLAYERS_FILE."
