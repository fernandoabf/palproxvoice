-- PalProxVoice — PROBE de identidade SEGURO / INCREMENTAL (TEMPORARIO)
-- ====================================================================
-- POR QUE ESTE ARQUIVO E' DIFERENTE DO PROBE ANTERIOR:
--   pcall NAO protege contra crash NATIVO do UE4SS. Quando um acessador
--   "ruim" (:ToString(), GetUniqueId(), GetFullName(), ou struct encadeada
--   num objeto que nao suporta) toca o C++ da engine, o processo INTEIRO
--   morre — o pcall nunca chega a capturar nada. O probe antigo tentava ~15
--   acessadores numa tacada so, entao 1 deles fatal derrubava tudo e voce
--   nem sabia qual foi.
--
-- ESTRATEGIA SEGURA (este arquivo):
--   1) Testa UM acessador "arriscado" por SESSAO de jogo (1 por restart).
--   2) Marca no disco "TENTANDO <nome>" ANTES de tocar no acessador.
--      -> se o jogo crashar, na proxima abertura o arquivo mostra qual era;
--         ele e' marcado FATAL e PULADO automaticamente.
--   3) Acessadores "safe" (leitura de propriedade simples) sao agrupados na
--      MESMA sessao; so os arriscados forcam restart.
--   4) Os acessadores que JA crasharam (ToString/GetUniqueId/GetFullName)
--      foram REMOVIDOS — nao adianta re-testar o que ja sabemos ser fatal.
--   5) A posicao normal continua sendo escrita -> a voz NAO quebra durante o teste.
--
-- COMO USAR:
--   - Substitua o main.lua instalado por este arquivo (renomeie p/ main.lua).
--     Caminho ativo (WinGDK/Game Pass):
--       ...\Pal\Binaries\WinGDK\Mods\PalProxVoice\scripts\main.lua
--   - Abra o jogo, entre num servidor, espere ~5s (estabilizacao).
--   - Leia os resultados em  C:\Users\Public\ppv_id.txt
--   - Se o console pedir "REINICIE", feche e reabra o jogo p/ o proximo acessador.
--   - Quando aparecer "PROBE COMPLETO", RESTAURE o main.lua original.
--   - Pra recomecar do zero: apague C:\Users\Public\ppv_probe_state.txt e ppv_id.txt
-- ====================================================================
local UEHelpers = require("UEHelpers")

local PUB    = (os.getenv("PUBLIC") or "C:\\Users\\Public")
local OUT    = PUB .. "\\palproxvoice_pos.txt"   -- posicao normal (voz continua viva)
local STATE  = PUB .. "\\ppv_probe_state.txt"    -- maquina: status de cada acessador
local RESULT = PUB .. "\\ppv_id.txt"             -- humano: resultados acumulados
local RATE_MS = 50
local STABILIZE_TICKS = 100  -- ~5s com player valido antes de sondar (evita load/transicao)

local function try(fn)
    local ok, res = pcall(fn)
    if ok then return res end
    return nil
end

-- ---- lista ORDENADA de acessadores -------------------------------------
-- safe=true  -> leitura simples; varios na mesma sessao.
-- safe=false -> ARRISCADO; UM por sessao (forca restart depois).
-- cada fn retorna uma STRING (resultado) ou nil.
local function buildProbes(ps)
    return {
        { name = "PlayerId", safe = true, fn = function()
            local v = ps.PlayerId
            if v == nil then return nil end
            return string.format("%s", tostring(v))
        end },

        { name = "PlayerNamePrivate", safe = true, fn = function()
            local v = ps.PlayerNamePrivate
            if v == nil then return nil end
            return tostring(v)
        end },

        -- primeiro toque na struct IndividualHandleId: isolado.
        { name = "IndividualHandleId.present", safe = false, fn = function()
            local h = ps.IndividualHandleId
            if h == nil then return nil end
            return "presente (userdata)"
        end },

        -- ALVO PRINCIPAL: FGuid do player (casa com playerId da REST "6119...").
        { name = "IndividualHandleId.PlayerUId", safe = false, fn = function()
            local h = ps.IndividualHandleId
            if h == nil then return nil end
            local g = h.PlayerUId
            if g == nil then return nil end
            local a = g.A
            if a == nil then return nil end
            local b, c, d = g.B or 0, g.C or 0, g.D or 0
            -- FGuid -> 32 hex (A,B,C,D = 4x uint32). Compara com playerId da REST.
            return string.format("%08X%08X%08X%08X  (A=%s B=%s C=%s D=%s)",
                a, b, c, d, tostring(a), tostring(b), tostring(c), tostring(d))
        end },

        { name = "IndividualHandleId.InstanceId", safe = false, fn = function()
            local h = ps.IndividualHandleId
            if h == nil then return nil end
            local g = h.InstanceId
            if g == nil then return nil end
            local a = g.A
            if a == nil then return nil end
            local b, c, d = g.B or 0, g.C or 0, g.D or 0
            return string.format("%08X%08X%08X%08X", a, b, c, d)
        end },
    }
end

local ORDER = {
    "PlayerId",
    "PlayerNamePrivate",
    "IndividualHandleId.present",
    "IndividualHandleId.PlayerUId",
    "IndividualHandleId.InstanceId",
}

-- ---- estado persistido --------------------------------------------------
local status = {} -- name -> PENDING | TRYING | DONE | NIL | FATAL

local function saveState()
    local f = io.open(STATE, "w")
    if not f then return end
    for _, n in ipairs(ORDER) do
        f:write(n .. "|" .. (status[n] or "PENDING") .. "\n")
    end
    f:close()
end

local function appendResult(text)
    local f = io.open(RESULT, "a")
    if f then f:write(text .. "\n"); f:close() end
end

local function loadState()
    for _, n in ipairs(ORDER) do status[n] = "PENDING" end
    local f = io.open(STATE, "r")
    if not f then return end
    for line in f:lines() do
        local n, s = line:match("^(.-)|(.+)$")
        if n and status[n] ~= nil then status[n] = s end
    end
    f:close()
end

-- qualquer TRYING encontrado na abertura = a sessao anterior CRASHOU ali.
local function resolveCrash()
    local crashed = false
    for _, n in ipairs(ORDER) do
        if status[n] == "TRYING" then
            status[n] = "FATAL"
            crashed = true
            appendResult("[" .. n .. "] = FATAL (crashou o jogo nesta tentativa) -> pulando")
            print("[PalProxVoice] '" .. n .. "' crashou na sessao anterior -> marcado FATAL\n")
        end
    end
    if crashed then saveState() end
end

local function nextTarget()
    for _, n in ipairs(ORDER) do
        if status[n] == "PENDING" then return n end
    end
    return nil
end

-- ---- runtime ------------------------------------------------------------
local probeComplete = false
local passDone = false   -- ja rodou a pass NESTA sessao
local stable = 0

local function runProbePass(ps)
    local list = buildProbes(ps)
    local byName = {}
    for _, p in ipairs(list) do byName[p.name] = p end

    while true do
        local target = nextTarget()
        if not target then
            probeComplete = true
            appendResult("=== PROBE COMPLETO. Restaure o main.lua original. ===")
            print("[PalProxVoice] PROBE COMPLETO -> restaure o main.lua original.\n")
            saveState()
            return
        end

        local p = byName[target]
        -- marca TRYING e PERSISTE *antes* de tocar no acessador (crash aponta o culpado)
        status[target] = "TRYING"
        saveState()
        print("[PalProxVoice] sondando: " .. target .. " ...\n")

        local res = try(p.fn)
        if res == nil then
            status[target] = "NIL"
            appendResult("[" .. target .. "] = nil (nao existe / sem valor)")
        else
            status[target] = "DONE"
            appendResult("[" .. target .. "] = " .. tostring(res))
            print("[PalProxVoice] " .. target .. " = " .. tostring(res) .. "\n")
        end
        saveState()

        if not p.safe then
            print("[PalProxVoice] '" .. target .. "' OK. REINICIE o jogo p/ sondar o proximo.\n")
            return -- arriscado resolvido: fecha a sessao aqui
        end
        -- safe: segue pro proximo no mesmo restart
    end
end

local function getPlayer()
    local pc = try(function() return UEHelpers:GetPlayerController() end)
    if not pc then return nil, nil end
    if not try(function() return pc:IsValid() end) then return nil, nil end
    local pawn = try(function() return pc.Pawn end)
    if not pawn then return nil, nil end
    if not try(function() return pawn:IsValid() end) then return nil, nil end
    return pc, pawn
end

local function writePos(pc, pawn)
    local loc = try(function() return pawn:K2_GetActorLocation() end)
    if not loc or not (loc.X and loc.Y and loc.Z) then return end
    local rot = try(function() return pc:GetControlRotation() end)
        or try(function() return pawn:K2_GetActorRotation() end)
    local yaw = (rot and rot.Yaw) or 0
    local f = io.open(OUT, "w")
    if f then f:write(string.format("%.1f,%.1f,%.1f,%.1f", loc.X, loc.Y, loc.Z, yaw)); f:close() end
end

loadState()
resolveCrash()

LoopAsync(RATE_MS, function()
    ExecuteInGameThread(function()
        local ok, err = pcall(function()
            local pc, pawn = getPlayer()
            if not pc or not pawn then
                stable = 0 -- instavel: zera o contador de estabilizacao
                return
            end

            writePos(pc, pawn) -- voz continua funcionando durante o probe

            if probeComplete or passDone then return end
            stable = stable + 1
            if stable < STABILIZE_TICKS then return end

            local ps = try(function() return pc.PlayerState end)
                or try(function() return pc:GetPlayerState() end)
                or try(function() return pawn.PlayerState end)
            if not ps then return end

            passDone = true
            runProbePass(ps)
        end)
        if not ok then print("[PalProxVoice] erro: " .. tostring(err) .. "\n") end
    end)
    return false
end)

print("[PalProxVoice] PROBE SEGURO carregado. Resultados -> " .. RESULT .. "\n")
