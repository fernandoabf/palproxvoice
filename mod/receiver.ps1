# Receiver de teste do M1 — roda no PC do jogo (Windows), SEM instalar nada.
#   powershell -ExecutionPolicy Bypass -File receiver.ps1
# Le o arquivo que o mod escreve e mostra x,y,z,yaw em tempo real.
$f = Join-Path $env:TEMP "palproxvoice_pos.txt"
Write-Host "lendo $f - entra no jogo e anda (Ctrl+C pra sair)"
while ($true) {
  if (Test-Path $f) {
    try { $line = (Get-Content $f -Raw -ErrorAction Stop).Trim() } catch { $line = "" }
    if ($line) { Write-Host ("`r" + $line.PadRight(40)) -NoNewline }
  }
  Start-Sleep -Milliseconds 100
}
