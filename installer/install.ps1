# PalProxVoice — instalador
#  1. acha o Palworld (Steam auto, ou pergunta)
#  2. instala UE4SS v3.0.1 + mod (do bundle) no jogo
#  3. instala o companion + config.json
#  4. configura auto-start (sobe escondido com o Windows)
$ErrorActionPreference = 'Stop'
$here = $PSScriptRoot
Write-Host "==== PalProxVoice - instalador ====" -ForegroundColor Cyan

# ---- 1) achar o Palworld ----
function Find-Palworld {
  try {
    $steam = (Get-ItemProperty 'HKCU:\Software\Valve\Steam' -Name SteamPath -ErrorAction Stop).SteamPath
  } catch { $steam = $null }
  $roots = @()
  if ($steam) {
    $roots += (Join-Path $steam 'steamapps\common\Palworld')
    $vdf = Join-Path $steam 'steamapps\libraryfolders.vdf'
    if (Test-Path $vdf) {
      foreach ($line in Get-Content $vdf) {
        if ($line -match '"path"\s*"(.+?)"') {
          $roots += (Join-Path ($matches[1] -replace '\\\\', '\') 'steamapps\common\Palworld')
        }
      }
    }
  }
  foreach ($r in $roots) { if (Test-Path (Join-Path $r 'Pal\Binaries')) { return $r } }
  return $null
}

# resolve a RAIZ do Palworld a partir de qualquer pasta perto: sobe ate um
# ancestral com Pal\Binaries; senao procura pra baixo (ex: a 'common' do Steam).
function Resolve-PalRoot([string]$dir) {
  if (-not $dir) { return $null }
  $cur = $dir.TrimEnd('\')
  for ($i = 0; $i -lt 5 -and $cur; $i++) {
    if (Test-Path (Join-Path $cur 'Pal\Binaries')) { return $cur }
    $cur = Split-Path $cur -Parent
  }
  if (Test-Path $dir) {
    $hit = Get-ChildItem -Path $dir -Directory -Recurse -Depth 3 -ErrorAction SilentlyContinue |
      Where-Object { Test-Path (Join-Path $_.FullName 'Pal\Binaries') } | Select-Object -First 1
    if ($hit) { return $hit.FullName }
  }
  return $null
}

$game = Find-Palworld
if (-not $game) {
  Write-Host "Nao achei o Palworld automaticamente." -ForegroundColor Yellow
  $typed = (Read-Host "Cola a pasta do Palworld (procuro o jogo dentro dela)").Trim('"')
  $game = Resolve-PalRoot $typed
}
if (-not $game) { Write-Host "Nao achei o Palworld nessa pasta." -ForegroundColor Red; exit 1 }

$bin = $null
foreach ($s in 'Win64', 'WinGDK') {
  $p = Join-Path $game "Pal\Binaries\$s"
  if (Test-Path (Join-Path $p "Palworld-$s-Shipping.exe")) { $bin = $p; break }
}
if (-not $bin) { Write-Host "Nao achei Palworld-*-Shipping.exe em Pal\Binaries. Abortando." -ForegroundColor Red; exit 1 }
Write-Host "Palworld encontrado: $bin"

# ---- 2) UE4SS + mod (do bundle, layout flat) ----
Write-Host "Instalando UE4SS + mod..."
Expand-Archive (Join-Path $here 'PalProxVoice-UE4SS.zip') -DestinationPath $bin -Force
Write-Host "  OK -> $bin"

# ---- 3) companion + config ----
$app = Join-Path $env:LOCALAPPDATA 'PalProxVoice'
New-Item -ItemType Directory -Force -Path $app | Out-Null
Copy-Item (Join-Path $here 'palproxvoice.exe') (Join-Path $app 'palproxvoice.exe') -Force
if (Test-Path (Join-Path $here 'config.json')) {
  Copy-Item (Join-Path $here 'config.json') (Join-Path $app 'config.json') -Force
}
Write-Host "Companion instalado: $app"

# ---- 4) auto-start (sobe OCULTO com o Windows, sem aba na barra de tarefas) ----
$startup = [Environment]::GetFolderPath('Startup')
$wsh = New-Object -ComObject WScript.Shell
$lnk = $wsh.CreateShortcut((Join-Path $startup 'PalProxVoice.lnk'))
$lnk.TargetPath = (Join-Path $app 'palproxvoice.exe')
$lnk.Arguments = '-min'  # -min = iniciar escondido (vira overlay quando voce entra no jogo)
$lnk.Save()

# sobe ele agora VISIVEL (sem -min) pra voce ver a janela e configurar
Start-Process (Join-Path $app 'palproxvoice.exe')

Write-Host ""
Write-Host "PRONTO! O companion sobe sozinho com o Windows." -ForegroundColor Green
Write-Host "Abre o Palworld e entra num servidor - a voz conecta automatica."
