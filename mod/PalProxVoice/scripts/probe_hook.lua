-- PalProxVoice — PROBE de HOOK cliente (recebe RPC do servidor) — TEMPORARIO
-- ====================================================================
-- Testa a METADE CLIENTE da ideia "servidor anuncia o voz via RPC": dar RegisterHook
-- nos UFunctions candidatos e LER o argumento (FString). NAO precisa de mod no servidor.
--   - Confirma quais UFunctions EXISTEM no build (RegisterHook funciona = existe).
--   - BroadcastChatMessage dispara ao MANDAR UMA MENSAGEM NO CHAT -> prova hook + leitura.
--   - SendScreenLogToClient / Debug_..._ToClient so disparam se o SERVIDOR chamar (NetClient);
--     aqui o valor e' so saber se EXISTEM (registro ok).
--
-- Cada callback loga "FIRED <nome>" ANTES de ler o argumento (se a leitura crashar,
-- a gente ainda sabe que o hook funciona). Continua escrevendo a posicao (voz viva).
--
-- USO: renomeie pra main.lua, entre num servidor, MANDE UMA MENSAGEM NO CHAT, veja
-- C:\Users\Public\ppv_hook_probe.txt. Manda o conteudo. Depois restaure o main.lua.
-- ====================================================================
local UEHelpers = require("UEHelpers")

local PUB = (os.getenv("PUBLIC") or "C:\\Users\\Public")
local OUT = PUB .. "\\ppv_hook_probe.txt"
local POS = PUB .. "\\palproxvoice_pos.txt"
local RATE_MS = 50

local function try(fn)
    local ok, r = pcall(fn)
    if ok then return r end
    return nil
end

local function logf(line)
    local f = io.open(OUT, "a")
    if f then f:write(line .. "\n"); f:close() end
    print("[PPV-HOOK] " .. line .. "\n")
end

-- le um argumento FString de hook: :get() -> string OU objeto FString -> :ToString()
local function readFStr(p)
    local g = try(function() return p:get() end)
    if type(g) == "string" then return g end
    if g ~= nil then
        local ts = try(function() return g:ToString() end)
        if ts then return ts end
        return "userdata(" .. tostring(g) .. ")"
    end
    return try(function() return p:ToString() end)
end

-- UFunctions candidatos (do dump do SDK)
local HOOKS = {
    { name = "SendScreenLogToClient", path = "/Script/Pal.PalPlayerController:SendScreenLogToClient",
      cb = function(self, Message) logf("FIRED SendScreenLogToClient"); logf("  Message = " .. tostring(readFStr(Message))) end },
    { name = "Debug_ReceiveCheatCommand_ToClient", path = "/Script/Pal.PalPlayerController:Debug_ReceiveCheatCommand_ToClient",
      cb = function(self, Message) logf("FIRED Debug_ReceiveCheatCommand_ToClient"); logf("  Message = " .. tostring(readFStr(Message))) end },
    { name = "BroadcastChatMessage", path = "/Script/Pal.PalGameStateInGame:BroadcastChatMessage",
      cb = function(self, ChatMessage)
          logf("FIRED BroadcastChatMessage (mande chat = isso aqui)")
          local cm = try(function() return ChatMessage:get() end)
          if not cm then logf("  ChatMessage:get() = nil"); return end
          logf("  .Message = " .. tostring(try(function() return cm.Message:ToString() end)))
          logf("  .Sender  = " .. tostring(try(function() return cm.Sender:ToString() end)))
      end },
}

local registered = false
local function registerAll()
    for _, h in ipairs(HOOKS) do
        local ok = try(function() RegisterHook(h.path, h.cb); return true end)
        logf("RegisterHook " .. h.name .. " -> " .. (ok and "OK (existe)" or "FALHOU (nao existe nesse build?)"))
    end
end

local function writePos()
    local pc = try(function() return UEHelpers:GetPlayerController() end)
    if not pc or not try(function() return pc:IsValid() end) then return end
    local pawn = try(function() return pc.Pawn end)
    if not pawn or not try(function() return pawn:IsValid() end) then return end
    local loc = try(function() return pawn:K2_GetActorLocation() end)
    if not loc or not (loc.X and loc.Y and loc.Z) then return end
    local rot = try(function() return pc:GetControlRotation() end) or try(function() return pawn:K2_GetActorRotation() end)
    local yaw = (rot and rot.Yaw) or 0
    local f = io.open(POS, "w")
    if f then f:write(string.format("%.1f,%.1f,%.1f,%.1f", loc.X, loc.Y, loc.Z, yaw)); f:close() end
end

do local f = io.open(OUT, "a"); if f then f:write("==== probe de hook cliente ====\n"); f:close() end end

LoopAsync(RATE_MS, function()
    ExecuteInGameThread(function()
        local ok, err = pcall(function()
            writePos()
            if not registered then
                local pc = try(function() return UEHelpers:GetPlayerController() end)
                if pc and try(function() return pc:IsValid() end) then
                    registered = true
                    logf("entrou no mundo -> registrando hooks...")
                    registerAll()
                    logf("hooks registrados. AGORA MANDE UMA MENSAGEM NO CHAT.")
                end
            end
        end)
        if not ok then print("[PPV-HOOK] erro: " .. tostring(err) .. "\n") end
    end)
    return false
end)

print("[PPV-HOOK] probe de hook carregado. Log -> " .. OUT .. "\n")
