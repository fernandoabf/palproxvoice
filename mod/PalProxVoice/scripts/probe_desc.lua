-- PalProxVoice — PROBE do ServerDescription SEGURO / crash-isolation (TEMPORARIO)
-- ====================================================================
-- Descobre se da pra ler o ServerDescription replicado via Lua SEM adivinhar.
-- LICAO aprendida: ler struct complexo via Lua pode CRASHAR nativo (pcall nao pega).
-- - NAO usa GetOptionWorldSettings (essa funcao JA crashou - retorna struct por valor).
-- - So walk IN-PLACE pelo GameState: GameState -> OptionReplicator -> OptionWorldSettings
--   -> ServerDescription -> :ToString().
-- - 1 hop ARRISCADO por sessao; grava "TRYING <hop>" ANTES de tocar. Se o jogo crashar,
--   na proxima abertura o arquivo mostra qual era -> marca FATAL e PULA.
-- - So roda IN-GAME (GameState existe). Continua escrevendo pos (a voz nao quebra).
--
-- USO: renomeie pra main.lua, entre num servidor (com ppv= na descricao), espere ~3s.
-- Leia C:\Users\Public\ppv_desc_probe.txt. Se pedir "REINICIE", reabre o jogo e entra
-- de novo. Quando der "COMPLETO" (ou achar o ServerDescription), me manda o arquivo.
-- Pra recomecar: apague ppv_desc_state.txt e ppv_desc_probe.txt.
-- ====================================================================
local UEHelpers = require("UEHelpers")

local PUB    = (os.getenv("PUBLIC") or "C:\\Users\\Public")
local POS    = PUB .. "\\palproxvoice_pos.txt"
local STATE  = PUB .. "\\ppv_desc_state.txt"   -- maquina: status de cada hop
local RESULT = PUB .. "\\ppv_desc_probe.txt"   -- humano: resultados
local RATE_MS = 50
local STABILIZE = 60 -- ~3s com GameState presente antes de sondar

local function try(fn)
    local ok, r = pcall(fn)
    if ok then return r end
    return nil
end

-- re-walka do zero ate a profundidade pedida (cada hop e' auto-contido)
local function getGS()  return try(function() return FindFirstOf("PalGameStateInGame") end) end
local function getRep() local gs = getGS(); return gs and try(function() return gs.OptionReplicator end) end
local function getOWS() local r = getRep(); return r and try(function() return r.OptionWorldSettings end) end

-- ---- hops (ordem; safe agrupa, risky isola 1/sessao) ----
local ORDER = { "GameState", "OptionReplicator", "OptionWorldSettings", "SrvDesc.field", "SrvDesc.toString", "SrvName.toString" }
local function runHop(name)
    if name == "GameState" then
        local gs = getGS(); if not gs then return nil end
        return "presente"
    elseif name == "OptionReplicator" then
        local r = getRep(); if not r then return nil end
        return "presente (" .. tostring(r) .. ")"
    elseif name == "OptionWorldSettings" then -- RISKY: materializa o struct grande
        local o = getOWS(); if not o then return nil end
        return "presente (" .. tostring(o) .. ")"
    elseif name == "SrvDesc.field" then -- RISKY: pega o FString (sem :ToString())
        local o = getOWS(); if not o then return nil end
        local f = try(function() return o.ServerDescription end)
        if f == nil then return nil end
        return "tipo=" .. tostring(f) -- "FString: ptr" se for FString de verdade
    elseif name == "SrvDesc.toString" then -- RISKY: o valor de verdade
        local o = getOWS(); if not o then return nil end
        local f = try(function() return o.ServerDescription end)
        if not f then return nil end
        local v = try(function() return f:ToString() end)
        return v and ("\"" .. tostring(v) .. "\"") or nil
    elseif name == "SrvName.toString" then -- RISKY (bonus)
        local o = getOWS(); if not o then return nil end
        local f = try(function() return o.ServerName end)
        if not f then return nil end
        local v = try(function() return f:ToString() end)
        return v and ("\"" .. tostring(v) .. "\"") or nil
    end
    return nil
end
local function isSafe(name) return name == "GameState" or name == "OptionReplicator" end

-- ---- estado persistido ----
local status = {}
local function saveState() local f=io.open(STATE,"w"); if f then for _,n in ipairs(ORDER) do f:write(n.."|"..(status[n] or "PENDING").."\n") end f:close() end end
local function appendResult(t) local f=io.open(RESULT,"a"); if f then f:write(t.."\n"); f:close() end print("[PPV-DESC] "..t.."\n") end
local function loadState()
    for _,n in ipairs(ORDER) do status[n]="PENDING" end
    local f=io.open(STATE,"r"); if not f then return end
    for line in f:lines() do local n,s=line:match("^(.-)|(.+)$"); if n and status[n]~=nil then status[n]=s end end
    f:close()
end
local function resolveCrash()
    local c=false
    for _,n in ipairs(ORDER) do if status[n]=="TRYING" then status[n]="FATAL"; c=true
        appendResult("["..n.."] = FATAL (CRASHOU o jogo aqui) -> pulando") end end
    if c then saveState() end
end
local function nextTarget() for _,n in ipairs(ORDER) do if status[n]=="PENDING" then return n end end return nil end

local complete=false
local passDone=false
local gsTicks=0
local tick=0

local function runPass()
    while true do
        local t=nextTarget()
        if not t then complete=true; appendResult("=== COMPLETO. Restaure o main.lua. ==="); saveState(); return end
        status[t]="TRYING"; saveState()
        appendResult("sondando: "..t.." ...")
        local res=runHop(t)
        if res==nil then status[t]="NIL"; appendResult("["..t.."] = nil (nao alcancou)")
        else status[t]="DONE"; appendResult("["..t.."] = "..tostring(res)) end
        saveState()
        if not isSafe(t) then appendResult("'"..t.."' OK. REINICIE o jogo p/ o proximo hop."); return end
    end
end

local function writePos()
    local pc=try(function() return UEHelpers:GetPlayerController() end)
    if not pc or not try(function() return pc:IsValid() end) then return end
    local pawn=try(function() return pc.Pawn end)
    if not pawn or not try(function() return pawn:IsValid() end) then return end
    local loc=try(function() return pawn:K2_GetActorLocation() end)
    if not loc or not (loc.X and loc.Y and loc.Z) then return end
    local rot=try(function() return pc:GetControlRotation() end) or try(function() return pawn:K2_GetActorRotation() end)
    local yaw=(rot and rot.Yaw) or 0
    local f=io.open(POS,"w"); if f then f:write(string.format("%.1f,%.1f,%.1f,%.1f",loc.X,loc.Y,loc.Z,yaw)); f:close() end
end

loadState()
resolveCrash()

LoopAsync(RATE_MS, function()
    ExecuteInGameThread(function()
        local ok,err=pcall(function()
            writePos()
            if complete or passDone then return end
            -- so sonda quando ja em jogo (GameState presente) e estavel
            if not getGS() then gsTicks=0; return end
            gsTicks=gsTicks+1
            if gsTicks<STABILIZE then return end
            passDone=true
            runPass()
        end)
        if not ok then print("[PPV-DESC] erro: "..tostring(err).."\n") end
    end)
    return false
end)

print("[PPV-DESC] PROBE ServerDescription (crash-isolation) carregado. Log -> "..RESULT.."\n")
