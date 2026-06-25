-- probe_guild.lua — DESCOBERTA: como ler a GUILD do player local (Fase auto-guild).
-- Roda junto com o mod, loga o que o PalGroupManager / PalGroupGuild expoem no UE4SS.log.
-- NAO escreve feed nem altera nada. Use ESTANDO numa guild. Manda o log que eu escrevo a
-- leitura real (mesma estrategia que achou o FGuid). Tudo em pcall (objeto ruim nao crasha).
local function try(fn) local ok, r = pcall(fn); if ok then return r end end
local function s(v) if v == nil then return "nil" end
  local ok, str = pcall(function() if type(v) == "userdata" and v.ToString then return v:ToString() end return tostring(v) end)
  return ok and str or "?"
end
local function log(line) print("[ppv-guild] " .. line .. "\n") end

local function fguidStr(g)
  local a = try(function() return g.A end); if a == nil then return nil end
  return string.format("%08X%08X%08X%08X", a, try(function() return g.B end) or 0,
    try(function() return g.C end) or 0, try(function() return g.D end) or 0)
end

local done = false
local function dump()
  if done then return end
  log("===== probe guild =====")

  -- 0) meu PlayerUId (referencia pra casar quem sou eu na guild)
  local UEHelpers = try(function() return require("UEHelpers") end)
  local pc = UEHelpers and try(function() return UEHelpers:GetPlayerController() end)
  local ps = pc and (try(function() return pc.PlayerState end) or try(function() return pc:GetPlayerState() end))
  local myUid = ps and try(function() return ps.IndividualHandleId.PlayerUId end)
  log("meu PlayerUId = " .. (myUid and (fguidStr(myUid) or "struct") or "nil"))

  -- 1) PalGroupManager existe? (gerencia as guilds)
  local gm = try(function() return FindFirstOf("PalGroupManager") end)
  log("PalGroupManager = " .. s(gm))

  -- 2) guilds existentes: classes provaveis
  for _, cls in ipairs({ "PalGroupGuild", "PalGroupBase", "PalGroupOrganization" }) do
    local arr = try(function() return FindAllOf(cls) end)
    if arr then
      log(cls .. " -> count=" .. #arr)
      for i, g in ipairs(arr) do
        if i > 3 then break end
        local name = try(function() return g.GroupName end) or try(function() return g.group_name end) or try(function() return g.guild_name end)
        local gid  = try(function() return g.group_id end) or try(function() return g.GroupId end) or try(function() return g.id end)
        local pls  = try(function() return g.players end) or try(function() return g.RawPlayerArray end) or try(function() return g.IndividualHandleIds end)
        local plsN = pls and try(function() return #pls end)
        log(string.format("  [%d] name=%s  group_id=%s  players=%s (#%s)",
          i, s(name), gid and (fguidStr(gid) or s(gid)) or "nil", s(pls), tostring(plsN)))
        -- 1o membro da guild: que campos tem? (pra saber como casar o meu uid)
        local p0 = pls and try(function() return pls[1] end)
        if p0 then
          local puid = try(function() return p0.player_uid end) or try(function() return p0.PlayerUId end) or try(function() return p0.individual_id end)
          log("       player[1].uid = " .. (puid and (fguidStr(puid) or s(puid)) or s(p0)))
        end
      end
      if #arr > 0 then done = true end -- achou guild -> para (ja temos a estrutura)
    else
      log(cls .. " -> (FindAllOf nil)")
    end
  end
  log("===== fim (manda esse trecho) =====")
end

-- tenta de tempos em tempos ate achar uma guild (voce precisa estar numa)
LoopAsync(4000, function()
  ExecuteInGameThread(function()
    local ok, err = pcall(dump)
    if not ok then log("erro: " .. tostring(err)) end
  end)
  return done   -- para quando achar
end)

log("probe_guild carregado — entre/esteja numa guild; dump a cada 4s ate achar.")
