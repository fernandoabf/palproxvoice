-- PalProxVoice — PROBE do IP do servidor v3 (INFINITO + delay, sem flood)
-- ====================================================================
--   - 1 RODADA a cada ~3s, pra SEMPRE (sem cap).
--   - DEDUP por valor: so loga quando o resultado MUDA (nao repete linha igual).
--     (o flood do v1 era dedup no ponteiro do userdata, que muda a cada leitura;
--      aqui a chave e o VALOR extraido por :get(), que e estavel.)
--   - extrai com :get() (RemoteUnrealParam do UE4SS). SEM :ToString() (crash nativo).
--   - continua escrevendo a posicao (a voz nao quebra).
--
-- USO: renomeie pra main.lua, REINICIE o jogo (no menu), entre num servidor
-- (Direct Connect). Veja C:\Users\Public\ppv_server_probe.txt e me manda.
-- ====================================================================
local UEHelpers = require("UEHelpers")

local PUB    = (os.getenv("PUBLIC") or "C:\\Users\\Public")
local OUT    = PUB .. "\\palproxvoice_pos.txt"
local SRVOUT = PUB .. "\\palproxvoice_server.txt"
local PROBE  = PUB .. "\\ppv_server_probe.txt"
local RATE_MS = 50

local function try(fn)
    local ok, res = pcall(fn)
    if ok then return res end
    return nil
end

-- descreve com seguranca: tipo (sem ponteiro -> dedup) + :get() + :ToString().
-- :ToString() inexistente no wrapper vira "attempt to call nil" -> pcall pega (sem crash).
local function show(v)
    if v == nil then return "nil" end
    local lt = type(v)
    if lt ~= "userdata" then return lt .. "(" .. tostring(v) .. ")" end
    local tw = (tostring(v):match("^(%S+)") or "userdata"):gsub(":$", "") -- "FString"/"UObject"/...
    local got = try(function() return v:get() end)
    local ts  = try(function() return v:ToString() end)
    local s = "ud<" .. tw .. ">"
    if got ~= nil then s = s .. " get=" .. tostring(got) end
    if ts ~= nil then s = s .. " ToString=" .. tostring(ts) end
    if got == nil and ts == nil then s = s .. " (sem get/ToString)" end
    return s
end

-- log com DEDUP por chave: so escreve quando a linha daquela chave muda.
local lastLog = {}
local function logf(key, line)
    if lastLog[key] == line then return end
    lastLog[key] = line
    local f = io.open(PROBE, "a")
    if f then f:write(line .. "\n"); f:close() end
    print("[PalProxVoice] " .. line .. "\n")
end

local function writePos()
    local pc = try(function() return UEHelpers:GetPlayerController() end)
    if not pc or not try(function() return pc:IsValid() end) then return end
    local pawn = try(function() return pc.Pawn end)
    if not pawn or not try(function() return pawn:IsValid() end) then return end
    local loc = try(function() return pawn:K2_GetActorLocation() end)
    if not loc or not (loc.X and loc.Y and loc.Z) then return end
    local rot = try(function() return pc:GetControlRotation() end)
        or try(function() return pawn:K2_GetActorRotation() end)
    local yaw = (rot and rot.Yaw) or 0
    local f = io.open(OUT, "w")
    if f then f:write(string.format("%.1f,%.1f,%.1f,%.1f", loc.X, loc.Y, loc.Z, yaw)); f:close() end
end

local function ipv4(s)
    return type(s) == "string" and s:match("^%d+%.%d+%.%d+%.%d+$") ~= nil
end

local tick = 0
local srvWritten = false

do local f = io.open(PROBE, "a"); if f then f:write("==== probe v3 (infinito + delay) ====\n"); f:close() end end

LoopAsync(RATE_MS, function()
    ExecuteInGameThread(function()
        local ok, err = pcall(function()
            writePos()
            tick = tick + 1
            if tick % 60 ~= 0 then return end -- 1 rodada a cada ~3s, pra sempre

            -- candidato 1: PendingNetGame.URL (so existe no handshake de conexao)
            local png = try(function() return FindFirstOf("PendingNetGame") end)
            if not png then
                logf("png", "1. PendingNetGame = nil")
            else
                local url = try(function() return png.URL end)
                logf("png", "1. PendingNetGame.URL = " .. show(url))
                if url then
                    local host = try(function() return url.Host end)
                    local port = try(function() return url.Port end)
                    logf("png.host", "     .Host = " .. show(host))
                    logf("png.port", "     .Port = " .. show(port))
                    local h = try(function() return host:get() end)
                    local p = try(function() return port:get() end)
                    if ipv4(h) and not srvWritten then
                        local s = p and (tostring(h) .. ":" .. tostring(p)) or tostring(h)
                        local f = io.open(SRVOUT, "w"); if f then f:write(s); f:close() end
                        srvWritten = true
                        logf("hit", "  >>> ACHOU IP: escrevi '" .. s .. "' em palproxvoice_server.txt")
                    end
                end
            end

            -- candidato 3: NetConnection.URL (rule-out; so quando ja conectado)
            local pc = try(function() return UEHelpers:GetPlayerController() end)
            local nc = pc and try(function() return pc.NetConnection end)
            if not nc then
                logf("nc", "3. NetConnection = nil (menu/host/local)")
            else
                local url2 = try(function() return nc.URL end)
                logf("nc", "3. NetConnection.URL = " .. show(url2))
                if url2 then
                    logf("nc.host", "     .Host = " .. show(try(function() return url2.Host end)))
                    logf("nc.port", "     .Port = " .. show(try(function() return url2.Port end)))
                end
            end
        end)
        if not ok then print("[PalProxVoice] erro: " .. tostring(err) .. "\n") end
    end)
    return false
end)

print("[PalProxVoice] PROBE v3 (infinito + delay) carregado. Log -> " .. PROBE .. "\n")
