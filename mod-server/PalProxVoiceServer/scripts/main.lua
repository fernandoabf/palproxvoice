-- PalProxVoice — mod SERVER-SIDE (V2): leitura AUTORITATIVA de posicao
-- ====================================================================
-- Roda DENTRO do servidor dedicado (via UE4SS). Le pos+yaw+FGuid de TODOS os players
-- (a fonte da verdade) e grava num arquivo que o servidor de voz CO-LOCADO consome.
-- Como o cliente nunca afirma posicao, isso e' anti-spoof de graca + delay minimo.
--
-- (UE4SS Lua nao tem socket -> ponte por ARQUIVO; o voz le esse arquivo no mesmo host.)
--
-- SAIDA:
--   C:\Users\Public\palproxvoice_players.txt  -> 1 linha/player: "fguid,x,y,z,yaw"
--   ue4ss/UE4SS.log                            -> logs de verificacao (headless)
--
-- INSTALACAO + Proton/Docker: ver mod-server/README.md. Use o UE4SS Okaetsu
-- experimental-palworld. So testar em host descartavel; NAO no servidor de producao.
-- ====================================================================
local UEHelpers = require("UEHelpers")

local PUB     = (os.getenv("PUBLIC") or "C:\\Users\\Public")
local OUT     = PUB .. "\\palproxvoice_players.txt"  -- feed pro voz (sobrescrito a cada tick)
local RATE_MS = 50   -- 20 Hz (delay minimo; ajustavel)

local function try(fn)
    local ok, r = pcall(fn)
    if ok then return r end
    return nil
end

local function log(line) print("[PPV-SRV] " .. line .. "\n") end

-- FGuid do player (== playerId da REST) via PlayerState (leitura provada no cliente)
local function fguidOf(pc)
    local ps = try(function() return pc.PlayerState end)
    if not ps then return nil end
    local g = try(function() return ps.IndividualHandleId.PlayerUId end)
    if not g then return nil end
    local a = try(function() return g.A end)
    if a == nil then return nil end
    local b = try(function() return g.B end) or 0
    local c = try(function() return g.C end) or 0
    local d = try(function() return g.D end) or 0
    if a == 0 and b == 0 and c == 0 and d == 0 then return nil end
    return string.format("%08X%08X%08X%08X", a, b, c, d)
end

-- snapshot de todos os players -> linhas "fguid,x,y,z,yaw"
local function snapshot()
    local pcs = try(function() return FindAllOf("PlayerController") end) or {}
    local lines = {}
    for _, pc in ipairs(pcs) do
        if try(function() return pc:IsValid() end) then
            local pawn = try(function() return pc.Pawn end)
            local loc = pawn and try(function() return pawn:K2_GetActorLocation() end)
            if loc and loc.X then -- so quem tem corpo no mundo (filtra CDO/sem pawn)
                local rot = try(function() return pc:GetControlRotation() end)
                    or try(function() return pawn:K2_GetActorRotation() end)
                local yaw = (rot and rot.Yaw) or 0
                local fg = fguidOf(pc) or "?"
                lines[#lines + 1] = string.format("%s,%.1f,%.1f,%.1f,%.1f", fg, loc.X, loc.Y, loc.Z, yaw)
            end
        end
    end
    return lines
end

local lastCount = -1

local function step()
    local lines = snapshot()
    local n = #lines
    -- grava o feed (sobrescreve) — o voz co-locado le isso
    local f = io.open(OUT, "w")
    if f then f:write(table.concat(lines, "\n")); if n > 0 then f:write("\n") end; f:close() end
    -- log so quando a contagem muda (nao floda o UE4SS.log)
    if n ~= lastCount then
        lastCount = n
        log("players autoritativos = " .. n .. " -> " .. OUT)
        for i, l in ipairs(lines) do log("  " .. l); if i >= 8 then break end end
    end
end

-- hook de join (server-only) — confirma que Lua roda no servidor com contexto de player
try(function()
    RegisterHook("/Script/Engine.PlayerController:ServerAcknowledgePossession", function()
        log("join detectado (ServerAcknowledgePossession)")
    end)
    log("hook de join registrado")
end)

LoopAsync(RATE_MS, function()
    ExecuteInGameThread(function()
        local ok, err = pcall(step)
        if not ok then print("[PPV-SRV] erro: " .. tostring(err) .. "\n") end
    end)
    return false
end)

log("mod server-side V2 carregado. feed -> " .. OUT)
