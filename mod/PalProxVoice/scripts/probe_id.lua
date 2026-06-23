-- PalProxVoice — PROBE de identidade (TEMPORARIO)
-- Objetivo: descobrir como ler, no cliente, o id do player local que casa com a
-- REST do servidor dedicado (playerId "61191836...0" = FGuid; userId "gdk_/steam_").
-- Tenta varios acessadores e ESCREVE o que funcionar em C:\Users\Public\ppv_id.txt.
-- Continua escrevendo a posicao normal (nao quebra a voz enquanto testa).
--
-- COMO USAR: substitua o main.lua instalado por este arquivo (renomeie p/ main.lua),
-- reinicie o jogo, entre num servidor. Depois leia o ppv_id.txt e RESTAURE o main.lua.
local UEHelpers = require("UEHelpers")
local OUT = (os.getenv("PUBLIC") or "C:\\Users\\Public") .. "\\palproxvoice_pos.txt"
local IDOUT = (os.getenv("PUBLIC") or "C:\\Users\\Public") .. "\\ppv_id.txt"
local RATE_MS = 50
local idDone = false

local function try(fn)
    local ok, res = pcall(fn)
    if ok then return res end
    return nil
end

-- formata qualquer coisa em string legivel (numero/hex p/ inteiros)
local function fmt(v)
    if v == nil then return nil end
    local t = type(v)
    if t == "number" then
        if v == math.floor(v) and v >= 0 then
            return string.format("%d (0x%X)", v, v)
        end
        return tostring(v)
    end
    if t == "boolean" or t == "string" then return tostring(v) end
    -- userdata (UObject/struct): tenta :ToString(), :type(), GetFullName()
    local s = try(function() return v:ToString() end)
    if s then return "ToString=" .. tostring(s) end
    local fn = try(function() return v:GetFullName() end)
    if fn then return "FullName=" .. tostring(fn) end
    return "userdata(" .. tostring(v) .. ")"
end

-- escreve identidade uma unica vez quando achar player valido
local function probeIdentity(pc, pawn)
    local lines = {}
    local function add(label, getter)
        local v = try(getter)
        if v ~= nil then
            local f = fmt(v)
            if f ~= nil then lines[#lines + 1] = label .. " = " .. f end
        end
    end

    local ps = try(function() return pc.PlayerState end)
        or try(function() return pc:GetPlayerState() end)
        or try(function() return pawn.PlayerState end)

    add("ps.present", function() return ps ~= nil end)
    if ps then
        add("ps.PlayerId", function() return ps.PlayerId end)
        add("ps.PlayerNamePrivate", function() return ps.PlayerNamePrivate end)
        add("ps:GetPlayerName()", function() return ps:GetPlayerName() end)
        -- Palworld: IndividualHandleId (FPalInstanceID) -> PlayerUId / InstanceId (FGuid)
        add("ps.IndividualHandleId", function() return ps.IndividualHandleId end)
        add("ps.IndividualHandleId.PlayerUId", function() return ps.IndividualHandleId.PlayerUId end)
        add("PlayerUId.A", function() return ps.IndividualHandleId.PlayerUId.A end)
        add("PlayerUId.B", function() return ps.IndividualHandleId.PlayerUId.B end)
        add("PlayerUId.C", function() return ps.IndividualHandleId.PlayerUId.C end)
        add("PlayerUId.D", function() return ps.IndividualHandleId.PlayerUId.D end)
        add("ps.IndividualHandleId.InstanceId", function() return ps.IndividualHandleId.InstanceId end)
        add("InstanceId.A", function() return ps.IndividualHandleId.InstanceId.A end)
        -- candidatos diretos
        add("ps.PlayerUId", function() return ps.PlayerUId end)
        add("ps.UserId", function() return ps.UserId end)
        add("ps:GetUniqueId()", function() return ps:GetUniqueId() end)
        add("ps.UniqueId", function() return ps.UniqueId end)
    end
    -- pelo PlayerController
    add("pc:GetUniqueNetIdAsString()", function() return pc:GetUniqueNetIdAsString() end)

    if #lines == 0 then return false end
    local f = io.open(IDOUT, "w")
    if f then
        f:write("[PalProxVoice probe de identidade]\n")
        f:write(table.concat(lines, "\n"))
        f:write("\n")
        f:close()
        print("[PalProxVoice] PROBE escrito em " .. IDOUT .. "\n")
        return true
    end
    return false
end

local function readpos()
    local pc = try(function() return UEHelpers:GetPlayerController() end)
    if not pc then return nil end
    if not try(function() return pc:IsValid() end) then return nil end
    local pawn = try(function() return pc.Pawn end)
    if not pawn then return nil end
    if not try(function() return pawn:IsValid() end) then return nil end

    if not idDone then idDone = probeIdentity(pc, pawn) end -- <- probe (uma vez)

    local loc = try(function() return pawn:K2_GetActorLocation() end)
    if not loc then return nil end
    local x, y, z = loc.X, loc.Y, loc.Z
    if not (x and y and z) then return nil end
    local rot = try(function() return pc:GetControlRotation() end)
        or try(function() return pawn:K2_GetActorRotation() end)
    local yaw = (rot and rot.Yaw) or 0
    return string.format("%.1f,%.1f,%.1f,%.1f", x, y, z, yaw)
end

LoopAsync(RATE_MS, function()
    ExecuteInGameThread(function()
        local ok, err = pcall(function()
            local line = readpos()
            if not line then return end
            local f = io.open(OUT, "w")
            if f then f:write(line); f:close() end
        end)
        if not ok then print("[PalProxVoice] erro: " .. tostring(err) .. "\n") end
    end)
    return false
end)

print("[PalProxVoice] PROBE de identidade carregado.\n")
