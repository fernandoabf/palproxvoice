-- PalProxVoice M1 — ponte de posicao (transporte por ARQUIVO; UE4SS nao traz LuaSocket)
-- Le posicao (X,Y,Z) + direcao (Yaw) do player e escreve "x,y,z,yaw" num arquivo
-- que o companion (ou o receiver de teste) le. Tambem imprime no console ~1x/s.
local UEHelpers = require("UEHelpers")
local OUT = (os.getenv("TEMP") or os.getenv("TMP") or ".") .. "\\palproxvoice_pos.txt"
local RATE_MS = 50          -- 20 Hz
local enabled = true
local tick = 0

-- No menu nao existe PlayerController e o GetPlayerController() *lanca erro*.
-- pcall pra tratar como "sem player ainda" (silencioso), nao como erro de verdade.
local function readpos()
    local ok, pc = pcall(UEHelpers.GetPlayerController, UEHelpers)
    if not ok or not pc or not pc:IsValid() then return nil end
    local pawn = pc.Pawn
    if not pawn or not pawn:IsValid() then return nil end
    local loc = pawn:K2_GetActorLocation()
    local rot = pc:GetControlRotation() -- se falhar: pawn:K2_GetActorRotation()
    return string.format("%.1f,%.1f,%.1f,%.1f", loc.X, loc.Y, loc.Z, rot.Yaw)
end

local function step()
    if not enabled then return end
    tick = tick + 1
    local line = readpos()
    if not line then
        if tick % 40 == 0 then print("[PalProxVoice] sem player ainda — entra num mundo\n") end
        return
    end
    local f = io.open(OUT, "w")
    if f then f:write(line); f:close() end
    if tick % 20 == 0 then print("[PalProxVoice] " .. line .. "\n") end -- ~1x/s no console
end

LoopAsync(RATE_MS, function()
    ExecuteInGameThread(function()
        local ok, err = pcall(step)
        if not ok then print("[PalProxVoice] erro real: " .. tostring(err) .. "\n") end
    end)
    return false -- continua o loop
end)

RegisterKeyBind(Key.F8, { ModifierKey.CONTROL }, function()
    enabled = not enabled
    print("[PalProxVoice] " .. (enabled and "ON" or "OFF") .. "\n")
end)

print("[PalProxVoice] carregado. escrevendo -> " .. OUT .. "\n")
