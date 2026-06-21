-- PalProxVoice M1 — ponte de posicao (arquivo fixo; UE4SS nao traz LuaSocket)
-- Le posicao (X,Y,Z) + direcao (Yaw) do player e escreve "x,y,z,yaw" num arquivo.
-- BLINDADO: cada chamada de engine vai em pcall, pra objeto em estado ruim
-- (durante load/transicao) nao derrubar o jogo.
local UEHelpers = require("UEHelpers")
local OUT = (os.getenv("PUBLIC") or "C:\\Users\\Public") .. "\\palproxvoice_pos.txt"
local RATE_MS = 50          -- 20 Hz
local enabled = true
local tick = 0

-- chama fn protegido; retorna nil em qualquer falha
local function try(fn)
    local ok, res = pcall(fn)
    if ok then return res end
    return nil
end

local function readpos()
    local pc = try(function() return UEHelpers:GetPlayerController() end)
    if not pc then return nil end
    if not try(function() return pc:IsValid() end) then return nil end

    local pawn = try(function() return pc.Pawn end)
    if not pawn then return nil end
    if not try(function() return pawn:IsValid() end) then return nil end

    local loc = try(function() return pawn:K2_GetActorLocation() end)
    if not loc then return nil end
    local x, y, z = loc.X, loc.Y, loc.Z
    if not (x and y and z) then return nil end

    -- yaw: ControlRotation (camera); se falhar, ActorRotation (corpo)
    local rot = try(function() return pc:GetControlRotation() end)
        or try(function() return pawn:K2_GetActorRotation() end)
    local yaw = (rot and rot.Yaw) or 0

    return string.format("%.1f,%.1f,%.1f,%.1f", x, y, z, yaw)
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
        if not ok then print("[PalProxVoice] erro: " .. tostring(err) .. "\n") end
    end)
    return false -- continua o loop
end)

-- keybind defensivo (alguns builds podem nao ter Key/ModifierKey)
pcall(function()
    RegisterKeyBind(Key.F8, { ModifierKey.CONTROL }, function()
        enabled = not enabled
        print("[PalProxVoice] " .. (enabled and "ON" or "OFF") .. "\n")
    end)
end)

print("[PalProxVoice] carregado. escrevendo -> " .. OUT .. "\n")
