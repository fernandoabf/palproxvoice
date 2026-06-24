-- PalProxVoice M1 — ponte de posicao (arquivo fixo; UE4SS nao traz LuaSocket)
-- Le posicao (X,Y,Z) + direcao (Yaw) do player e escreve "x,y,z,yaw" num arquivo.
-- BLINDADO: cada chamada de engine vai em pcall, pra objeto em estado ruim
-- (durante load/transicao) nao derrubar o jogo.
local UEHelpers = require("UEHelpers")
local OUT = (os.getenv("PUBLIC") or "C:\\Users\\Public") .. "\\palproxvoice_pos.txt"
local IDOUT = (os.getenv("PUBLIC") or "C:\\Users\\Public") .. "\\palproxvoice_id.txt"
local RATE_MS = 50          -- 20 Hz
local enabled = true
local tick = 0
local posCount = 0          -- frames validos seguidos no mundo (estabilizacao)
local idDone = false        -- FGuid do player escrito uma unica vez por sessao

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

-- escreve o FGuid do player (== playerId da REST do Palworld) UMA vez por sessao.
-- O companion le esse arquivo e manda no auth (campo "user"); o servidor de voz
-- correlaciona com a REST pro anti-spoof (cobre o caso mesmo-IP/2 PCs na casa).
-- ps.IndividualHandleId.PlayerUId foi confirmado seguro no probe; mesmo assim cada
-- acesso vai em try() e so roda depois de ~1s estavel no mundo (evita load/transicao).
local function writeId(pc, pawn)
    if idDone then return end
    local ps = try(function() return pc.PlayerState end)
        or try(function() return pc:GetPlayerState() end)
        or try(function() return pawn.PlayerState end)
    if not ps then return end
    local g = try(function() return ps.IndividualHandleId.PlayerUId end)
    if not g then return end
    local a = try(function() return g.A end)
    if a == nil then return end
    local b = try(function() return g.B end) or 0
    local c = try(function() return g.C end) or 0
    local d = try(function() return g.D end) or 0
    -- FGuid ainda nao replicou (tudo zero) -> nao trava um id ruim; tenta de novo
    if a == 0 and b == 0 and c == 0 and d == 0 then return end
    local id = string.format("%08X%08X%08X%08X", a, b, c, d)
    local f = io.open(IDOUT, "w")
    if f then
        f:write(id); f:close(); idDone = true
        print("[PalProxVoice] id=" .. id .. " -> " .. IDOUT .. "\n")
    end
end

local function step()
    if not enabled then return end
    tick = tick + 1
    -- so busca controller/pawn no nivel do step enquanto falta capturar o id;
    -- depois disso a posicao usa o pc/pawn proprios do readpos (sem fetch dobrado).
    local pc, pawn
    if not idDone then
        pc = try(function() return UEHelpers:GetPlayerController() end)
        pawn = pc and try(function() return pc.Pawn end)
    end
    local line = readpos()
    if not line then
        posCount = 0
        if tick % 40 == 0 then print("[PalProxVoice] sem player ainda — entra num mundo\n") end
        return
    end
    local f = io.open(OUT, "w")
    if f then f:write(line); f:close() end
    if tick % 20 == 0 then print("[PalProxVoice] " .. line .. "\n") end -- ~1x/s no console

    -- estabiliza ~1s no mundo antes de tocar na struct de identidade
    posCount = posCount + 1
    if not idDone and posCount >= 20 and pc and pawn then
        writeId(pc, pawn)
        if not idDone and posCount % 20 == 0 then
            print("[PalProxVoice] aguardando id do player (FGuid ainda nao pronto)...\n")
        end
    end
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
