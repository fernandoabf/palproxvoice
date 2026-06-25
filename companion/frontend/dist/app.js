// PalProxVoice Companion — frontend (Wails v2)
// Padrao: auto-conecta no servidor selecionado ao entrar no jogo, config escondida.
// Se falhar -> mostra fallback (Configurar/Ignorar). Multi-servidor fica na config.

const me = { x: 0, y: 0, z: 0, yaw: 0 };
const CM = 100;
const peers = {};
let posFromGame = false;
let inGame = false;
let myChannel = 'proximity';                          // canal que EU falo (Valorant): proximity | guild | global
let myGuild = localStorage.getItem('ppv_guild') || ''; // minha guild (codigo manual; o mod sobrescreve no auto)

let voiceRangeMeters = 50;
let masterVolume = 1.0;      // volume de saida
let inputVol = 1.0;          // volume do microfone (entrada)
let micMuted = false, deafened = false, micGain = null, micStream = null, micWs = null;
let micProc = null, meterRAF = 0, rnn = null;
// processamento do mic (persistido em localStorage; monitor sempre off no inicio)
const proc = (() => { let p = { highpass:true, comp:true, gate:true, sens:50, rnnoise:false };
  try { p = Object.assign(p, JSON.parse(localStorage.getItem('ppv_proc') || '{}')); } catch (_) {}
  p.monitor = false; return p; })();
function saveProc(){ try { localStorage.setItem('ppv_proc',
  JSON.stringify({ highpass:proc.highpass, comp:proc.comp, gate:proc.gate, sens:proc.sens, rnnoise:proc.rnnoise })); } catch (_) {} }

// RNNoise (IA, Xiph BSD): supressao de ruido neural. Processa frames de 480 @48k —
// exatamente o que o Go envia. Roda na MAIN THREAD ao receber cada frame do WS,
// antes do worklet. Mata teclado/ventilador ATE enquanto voce fala (o gate so corta
// no silencio). WASM embutido (rnnoise-sync.js), 100% local.
async function initRNNoise() {
  if (rnn) return rnn;
  try {
    const factory = (await import('./rnnoise-sync.js')).default;
    const M = await factory();
    const FRAME = 480, ctx = M._rnnoise_create(0), ptr = M._malloc(FRAME * 4);
    rnn = { M, ctx, ptr, FRAME, process(frame) {
      const heap = M.HEAPF32, base = ptr >> 2;
      for (let i = 0; i < FRAME; i++) heap[base + i] = frame[i] * 32768; // RNNoise quer faixa int16
      M._rnnoise_process_frame(ctx, ptr, ptr);
      const out = new Float32Array(FRAME);
      for (let i = 0; i < FRAME; i++) out[i] = heap[base + i] / 32768;
      return out;
    } };
    log(t('lg_rnnoise_ok'), 'ok');
  } catch (e) { log(t('lg_rnnoise_fail', {e}), 'err'); rnn = null; }
  return rnn;
}
// sensibilidade 0..100 -> limiar linear do gate (sens alto = mais sensivel = limiar menor)
function sensToThreshold(s){ return Math.pow(10, (-25 - (s/100)*35) / 20); }

// config completa (lista de servidores + global)
let cfg = { servers: [], selected: 0, volume: 1.0, micDeviceId: '', outputDeviceId: '', autoConnect: true };

const $ = id => document.getElementById(id);

// ----- i18n (PT/EN; padrao = idioma do sistema; salvo em localStorage) -----
const I18N = {
  pt: {
    win_min:'Diminuir (modo overlay)', win_close:'Fechar',
    tagline:'Voz por proximidade no Palworld',
    fb_msg:'⚠️ Não consegui conectar ao servidor', fb_config:'Configurar', fb_ignore:'Ignorar',
    st_title:'Status do sistema', m_server:'Servidor', m_players:'Jogadores', m_channel:'Canal', ch_prox:'Proximidade',
    btn_start:'🎙️ Iniciar voz de proximidade', btn_leave:'Desconectar',
    st_off:'Offline', sub_off:'Palworld fechado ou voz desligada',
    st_connecting:'Conectando…', sub_connecting:'estabelecendo conexão',
    st_searching:'Procurando jogo', sub_searching:'conectado — entre num servidor',
    st_on:'Conectado', sub_on:'voz de proximidade ativa',
    mute_on:'🎤 Mic', mute_off:'🔇 Mic mutado', deaf_on:'🔊 Som', deaf_off:'🔇 Sem som',
    lang:'🌐 Idioma / Language', audio:'Áudio', mic:'Microfone', vol_in:'Volume do microfone (entrada)',
    out:'Saída de áudio', vol_out:'Volume de saída', refresh:'Atualizar dispositivos',
    proc_title:'Processamento do microfone',
    p_highpass:'Passa-alta', p_highpass_h:'(corta zumbido/hum)',
    p_comp:'Compressor', p_comp_h:'(nivela a voz)',
    p_gate:'Noise gate', p_gate_h:'(corta ruído de fundo)',
    p_rnnoise:'Supressão de ruído IA', p_rnnoise_h:'(RNNoise — mata teclado/ventilador até falando)',
    p_sens:'Sensibilidade do gate',
    p_meter_hint:'Falando, a barra passa do marcador (branco) → abre. Em silêncio, fica abaixo → corta.',
    p_monitor:'🎧 Ouvir meu microfone', p_monitor_h:'(teste — use fone)',
    srv_title:'Servidor', srv_autodetect:'Detectar servidor do jogo (automático)',
    srv_port:'Porta da voz', srv_passauto:'Senha (auto)', ph_passauto:'vazio = sem senha',
    srv_manual:'Servidor manual (fallback)', srv_new:'+ Novo', srv_remove:'Remover',
    srv_name:'Nome', srv_addr:'Endereço', srv_pass:'Senha', srv_range:'Alcance da voz (metros)',
    srv_autoconnect:'Conectar automaticamente ao entrar no jogo', srv_save:'Salvar',
    log_title:'Logs do sistema', hud_expand:'Abrir janela', hud_players:'players',
    // logs do sistema (painel)
    lg_rnnoise_ok:'RNNoise (IA) carregado', lg_rnnoise_fail:'RNNoise falhou: {e}',
    lg_game_detected:'servidor do jogo detectado: {ip}', lg_no_ip:'não detectei o IP — usando servidor salvo',
    lg_left_server:'saiu do servidor — voz desconectada', lg_no_wails:'aviso: fora do Wails (sem posição do jogo)',
    lg_bitrate:'[rede] bitrate -> {kbps}kbps (perda {loss}%, rtt {rtt}ms)',
    lg_configure_server:'configure um servidor', lg_no_url:'servidor sem URL',
    lg_worklet_fail:'worklet do mic falhou: {e}', lg_hearing_peer:'ouvindo peer {id}',
    lg_mic_ws_err:'WS do mic nativo: erro', lg_mic_ok:'mic nativo ok (WASAPI — sem degradar o áudio)',
    lg_no_mic:'sem microfone ({err}) — você ouve, mas não fala',
    lg_error:'ERRO: {data}', lg_connected:'conectado (id {id})', lg_ws_err:'erro de websocket',
    lg_reconnecting:'conexão caiu — reconectando em {s}s…', lg_not_found:'falhou — não achei o servidor',
    lg_save_err:'erro ao salvar: {err}', lg_ready:'pronto — esperando entrar no jogo',
    lg_add_server:'adicione um servidor na config', lg_cfg_err:'erro ao carregar config: {err}',
  },
  en: {
    win_min:'Minimize (overlay mode)', win_close:'Close',
    tagline:'Voice proximity for Palworld',
    fb_msg:"⚠️ Couldn't connect to server", fb_config:'Configure', fb_ignore:'Ignore',
    st_title:'System status', m_server:'Server', m_players:'Players', m_channel:'Channel', ch_prox:'Proximity',
    btn_start:'🎙️ Start proximity voice', btn_leave:'Disconnect',
    st_off:'Offline', sub_off:'Palworld closed or voice off',
    st_connecting:'Connecting…', sub_connecting:'establishing connection',
    st_searching:'Looking for game', sub_searching:'connected — join a server',
    st_on:'Connected', sub_on:'proximity voice active',
    mute_on:'🎤 Mic', mute_off:'🔇 Mic muted', deaf_on:'🔊 Sound', deaf_off:'🔇 Deafened',
    lang:'🌐 Idioma / Language', audio:'Audio', mic:'Microphone', vol_in:'Microphone volume (input)',
    out:'Audio output', vol_out:'Output volume', refresh:'Refresh devices',
    proc_title:'Microphone processing',
    p_highpass:'High-pass', p_highpass_h:'(cuts rumble/hum)',
    p_comp:'Compressor', p_comp_h:'(levels the voice)',
    p_gate:'Noise gate', p_gate_h:'(cuts background noise)',
    p_rnnoise:'AI noise suppression', p_rnnoise_h:'(RNNoise — kills keyboard/fan even while talking)',
    p_sens:'Gate sensitivity',
    p_meter_hint:'While talking, the bar passes the marker (white) → opens. In silence, stays below → cuts.',
    p_monitor:'🎧 Hear my microphone', p_monitor_h:'(test — use headphones)',
    srv_title:'Server', srv_autodetect:'Detect game server (automatic)',
    srv_port:'Voice port', srv_passauto:'Password (auto)', ph_passauto:'empty = no password',
    srv_manual:'Manual server (fallback)', srv_new:'+ New', srv_remove:'Remove',
    srv_name:'Name', srv_addr:'Address', srv_pass:'Password', srv_range:'Voice range (meters)',
    srv_autoconnect:'Connect automatically when entering the game', srv_save:'Save',
    log_title:'System log', hud_expand:'Open window', hud_players:'players',
    // system log (panel)
    lg_rnnoise_ok:'RNNoise (AI) loaded', lg_rnnoise_fail:'RNNoise failed: {e}',
    lg_game_detected:'game server detected: {ip}', lg_no_ip:"couldn't detect the IP — using saved server",
    lg_left_server:'left the server — voice disconnected', lg_no_wails:'warning: outside Wails (no game position)',
    lg_bitrate:'[net] bitrate -> {kbps}kbps (loss {loss}%, rtt {rtt}ms)',
    lg_configure_server:'configure a server', lg_no_url:'server without URL',
    lg_worklet_fail:'mic worklet failed: {e}', lg_hearing_peer:'hearing peer {id}',
    lg_mic_ws_err:'native mic WS: error', lg_mic_ok:'native mic ok (WASAPI — no audio degradation)',
    lg_no_mic:"no microphone ({err}) — you hear, but can't talk",
    lg_error:'ERROR: {data}', lg_connected:'connected (id {id})', lg_ws_err:'websocket error',
    lg_reconnecting:'connection dropped — reconnecting in {s}s…', lg_not_found:"failed — couldn't find the server",
    lg_save_err:'failed to save: {err}', lg_ready:'ready — waiting to enter the game',
    lg_add_server:'add a server in the config', lg_cfg_err:'failed to load config: {err}',
  },
};
let lang = localStorage.getItem('ppv_lang') || ((navigator.language || '').toLowerCase().startsWith('pt') ? 'pt' : 'en');
function t(k, p){ let s = (I18N[lang] && I18N[lang][k]) || I18N.en[k] || k; if (p) for (const n in p) s = s.split('{'+n+'}').join(p[n]); return s; }
function applyI18n(){
  document.documentElement.lang = lang;
  document.querySelectorAll('[data-i18n]').forEach(el => { el.textContent = t(el.getAttribute('data-i18n')); });
  document.querySelectorAll('[data-i18n-ph]').forEach(el => { el.setAttribute('placeholder', t(el.getAttribute('data-i18n-ph'))); });
  document.querySelectorAll('[data-i18n-title]').forEach(el => { el.title = t(el.getAttribute('data-i18n-title')); });
  try { refreshStatus(); } catch (_) {}
  try { updateAudioBtns(); } catch (_) {}
}
function setLang(l){ lang = l; try { localStorage.setItem('ppv_lang', l); } catch (_) {} applyI18n(); }
const RT = window.runtime || {}; // runtime do Wails (controle de janela)
// log colorido com timestamp. level: 'ok' | 'warn' | 'err' | 'info'
function log(m, level) {
  const col = { ok:'var(--good)', warn:'var(--warn)', err:'var(--bad)', info:'var(--info)' };
  const e = $('log'); const d = new Date();
  const ts = ('0'+d.getHours()).slice(-2) + ':' + ('0'+d.getMinutes()).slice(-2);
  const row = document.createElement('div'); row.className = 'l';
  const tEl = document.createElement('span'); tEl.className = 't'; tEl.textContent = ts;
  const mEl = document.createElement('span'); mEl.textContent = m;
  if (col[level]) mEl.style.color = col[level];
  row.append(tEl, mEl); e.appendChild(row); e.scrollTop = e.scrollHeight;
}

const fwd = yaw => { const r = yaw * Math.PI / 180; return [Math.cos(r), 0, Math.sin(r)]; };
(() => { const [fx,,fz] = fwd(0); console.assert(Math.abs(fx-1)<1e-9 && Math.abs(fz)<1e-9, "fwd(0) +X"); })();

let actx, listener, masterGain;
function audioPos(p) { return [p.x/CM, p.z/CM, p.y/CM]; }
// SUAVIZACAO: posicao/orientacao chegam a ~10-20Hz; aplicar direto (.value=) faz o
// pan "pular" (HRTF e sensivel). setTargetAtTime faz a engine interpolar em taxa de
// audio (sem ziper, sem clique). TC=~60ms -> soa imediato mas glide suave entre updates.
const SMOOTH = 0.06;
function ramp(param, v) { if (!param) return; try { param.setTargetAtTime(v, actx.currentTime, SMOOTH); } catch (_) { param.value = v; } }
// Abordagem A PROVA DE EIXO: o LISTENER fica fixo na origem olhando -Z (frente),
// up +Y — o frame canonico do Web Audio (right = +X). Cada peer e' posicionado pela
// DIRECAO RELATIVA (bearing) calculada no plano do jogo a partir da MINHA posicao+yaw.
// Assim nao dependemos do mapeamento de eixos UE<->WebAudio (que causava o espelhamento):
// computamos o angulo e jogamos no frame conhecido. Sobra 1 sinal por eixo p/ acertar.
const AUDIO_FLIP_LR = 1;   // se esquerda/direita vier trocada -> troca p/ -1
const AUDIO_FLIP_FB = 1;   // se frente/tras vier trocada -> troca p/ -1
const NEAR_FULL = 1.5;     // ate aqui = 100% direto/centralizado (presente, "colado")
const NEAR_BLEND = 4;      // daqui pra frente = 100% HRTF (direcional)
function placeListener() {
  if (!listener) return;
  if (listener.positionX) {
    listener.positionX.value = 0; listener.positionY.value = 0; listener.positionZ.value = 0;
    listener.forwardX.value = 0; listener.forwardY.value = 0; listener.forwardZ.value = -1;
    listener.upX.value = 0; listener.upY.value = 1; listener.upZ.value = 0;
  } else { listener.setPosition(0, 0, 0); listener.setOrientation(0, 0, -1, 0, 1, 0); }
  for (const id in peers) placePanner(peers[id]); // re-coloca todos relativo a mim
}
function placePanner(p) {
  if (!p.panner) return;
  const dx = p.x - me.x, dy = p.y - me.y;
  const hd = Math.hypot(dx, dy) / CM;                         // distancia horizontal (m)
  const beta = Math.atan2(dy, dx) - me.yaw * Math.PI / 180;   // bearing relativo ao MEU yaw (0 = frente)
  const x = AUDIO_FLIP_LR * Math.sin(beta) * hd;              // +X = direita
  const z = AUDIO_FLIP_FB * -Math.cos(beta) * hd;             // -Z = frente
  const y = (p.z - me.z) / CM;                                // altura (cue fraco)
  if (p.panner.positionX) { ramp(p.panner.positionX, x); ramp(p.panner.positionY, y); ramp(p.panner.positionZ, z); }
  else p.panner.setPosition(x, y, z);
  // crossfade: perto = direto/centralizado (presente), longe = HRTF (direcional)
  if (p.directGain && p.pannerGain) {
    const near = Math.max(0, Math.min(1, (NEAR_BLEND - hd) / (NEAR_BLEND - NEAR_FULL)));
    // cutoff de proximidade: fade ate o SILENCIO ao chegar no alcance de voz (o modelo
    // inverse nunca zera sozinho -> senao da pra ouvir gente longe demais).
    const r = voiceRangeMeters, fs = r * 0.8;
    const range = hd <= fs ? 1 : Math.max(0, 1 - (hd - fs) / Math.max(1, r - fs));
    ramp(p.directGain.gain, near * range);
    ramp(p.pannerGain.gain, (1 - near) * range);
  }
  // frente/tras: atras (+Z no frame canonico) = abafado; frente = aberto.
  if (p.lp) ramp(p.lp.frequency, z > 0 ? 2800 : 18000);
}

// ----- canais de voz: proximidade (3D) / guild (plano, mesma guild) / global (plano) -----
// Estilo Valorant: o player ESCOLHE o canal que fala; a guild vem do mod (auto) ou de um codigo.
// O servidor so REPASSA quem fala em qual canal/guild; o MIX (o que voce ouve) e decidido aqui.
const CHANNELS = ['proximity', 'guild', 'global'];
// aplica o mix de um peer conforme o CANAL que ELE fala + a guild dele vs a MINHA.
function applyPeerChannel(p) {
  if (!p) return;
  if ((p.channel || 'proximity') === 'proximity') {
    if (p.flatGain) ramp(p.flatGain.gain, 0);
    placePanner(p);                                 // 3D por distancia (so proximidade)
    return;
  }
  if (p.pannerGain) ramp(p.pannerGain.gain, 0);     // guild/global = PLANO, sem panner
  if (p.directGain) ramp(p.directGain.gain, 0);
  const audible = p.channel === 'global' || (p.channel === 'guild' && myGuild && p.guild === myGuild);
  if (p.flatGain) ramp(p.flatGain.gain, audible ? 1 : 0);
}
function sendMeta() {
  if (ws && ws.readyState === 1)
    ws.send(JSON.stringify({ event: 'meta', data: JSON.stringify({ guild: myGuild || '', channel: myChannel }) }));
}
function setChannel(ch) {
  if (!CHANNELS.includes(ch) || ch === myChannel) return;
  myChannel = ch; sendMeta(); updateChannelUI(); log('canal de voz: ' + ch, 'ok');
}
function cycleChannel() { setChannel(CHANNELS[(CHANNELS.indexOf(myChannel) + 1) % CHANNELS.length]); }
function setMyGuild(g) {
  g = (g || '').trim();
  if (g === myGuild) return;
  myGuild = g; try { localStorage.setItem('ppv_guild', g); } catch (_) {}
  sendMeta();
  for (const id in peers) applyPeerChannel(peers[id]); // re-avalia audibilidade da guild
}
function updateChannelUI() {
  const pill = $('chanPill'); if (pill) pill.textContent = myChannel;
  document.querySelectorAll('[data-chan]').forEach(b => {
    const on = b.dataset.chan === myChannel;
    b.style.outline = on ? '2px solid #2ea44f' : ''; b.style.opacity = on ? '1' : '0.6';
  });
  const gi = $('guildCode'); if (gi && gi.value !== myGuild) gi.value = myGuild;
}
function bindChannelUI() {
  document.querySelectorAll('[data-chan]').forEach(b => b.onclick = () => setChannel(b.dataset.chan));
  const gi = $('guildCode'); if (gi) { gi.value = myGuild; gi.onchange = () => setMyGuild(gi.value); }
  updateChannelUI();
}
// tecla V cicla o canal (Valorant). So com a janela do companion FOCADA; in-game (overlay
// sem foco) precisa de hotkey global (Go) — proximo passo.
document.addEventListener('keydown', e => {
  const tag = e.target && e.target.tagName;
  if (tag && /^(INPUT|TEXTAREA|SELECT)$/.test(tag)) return;
  if (e.key === 'v' || e.key === 'V') cycleChannel();
});

// ----- status (badge + card + dot + metricas) -----
let connState = 'off'; // 'off' | 'connecting' | 'on'
function refreshStatus() {
  let txt, cls, sub;
  if (connState === 'off')             { txt=t('st_off');        cls='off';       sub=t('sub_off'); }
  else if (connState === 'connecting') { txt=t('st_connecting'); cls='searching'; sub=t('sub_connecting'); }
  else if (!posFromGame)               { txt=t('st_searching'); cls='searching'; sub=t('sub_searching'); }
  else                                 { txt=t('st_on');        cls='on';        sub=t('sub_on'); }
  $('conn').textContent = txt;    $('conn').className = 'badge ' + cls;
  $('connTxt').textContent = txt; $('statusDot').className = 'sdot ' + cls;
  $('status').textContent = sub;
  $('go').hidden = (connState !== 'off');
  $('leave').hidden = (connState === 'off');
  $('connMeta').hidden = (connState !== 'on');
  $('audioCtl').hidden = (connState !== 'on');
  if ($('hudDot')) { $('hudDot').className = 'sdot ' + cls; $('hudTxt').textContent = txt; }
}
function updatePlayers() {
  const n = Object.values(peers).filter(p => p.panner).length;
  $('mPlayers').textContent = n;
  if ($('hudPlayers')) $('hudPlayers').textContent = n;
}
refreshStatus();

// ----- overlay compacto (janela frameless) -----
// pequeno = HUD "Conectado · N players" no topo; cheio = dashboard/config.
let compact = false;
function setOverlayMode(on)  { try { window.go.main.App.SetOverlayMode(on); } catch (_) {} }
function applyOverlayStyle() { try { window.go.main.App.ApplyOverlayStyle(); } catch (_) {} }
async function positionTopRight(w) {
  try {
    if (!RT.ScreenGetAll) return;
    const ss = await RT.ScreenGetAll();
    const s = ss.find(x => x.isCurrent || x.IsCurrent) || ss.find(x => x.isPrimary || x.IsPrimary) || ss[0];
    if (!s) return;
    const sw = s.width || s.Width || (s.size && s.size.width) || (s.Size && s.Size.Width) || 0;
    if (sw > 0 && RT.WindowSetPosition) RT.WindowSetPosition(sw - w - 24, 28);
  } catch (_) {}
}
async function setCompact(on) {
  compact = on;
  setOverlayMode(on); // liga/desliga o watchdog (anti "mostrar area de trabalho")
  const wb = document.querySelector('.winbar'), sc = document.querySelector('.scroll');
  if (wb) wb.hidden = on;
  if (sc) sc.hidden = on;
  if ($('hud')) $('hud').hidden = !on;
  if (!RT.WindowSetSize) return;
  if (on) {
    RT.WindowSetSize(230, 46);
    RT.WindowSetAlwaysOnTop && RT.WindowSetAlwaysOnTop(true);
    await positionTopRight(230);
  } else {
    RT.WindowSetAlwaysOnTop && RT.WindowSetAlwaysOnTop(false);
    RT.WindowSetSize(960, 700);
    RT.WindowCenter && RT.WindowCenter();
  }
  applyOverlayStyle(); // garante "sem aba na taskbar" depois de mostrar/redimensionar
}
if ($('tbCompact')) $('tbCompact').onclick = () => setCompact(true);  // Diminuir -> HUD
if ($('tbClose'))   $('tbClose').onclick   = () => RT.Quit && RT.Quit(); // Fechar
if ($('hudExpand')) $('hudExpand').onclick = () => setCompact(false);

// ----- eventos do backend: "pos" e "posLost" -----
function onPos(data) {
  if (!data) return;
  const t = String(data).trim(); if (!t) return;
  const [x, y, z, yaw] = t.split(',').map(Number);
  if ([x, y, z, yaw].some(Number.isNaN)) return;
  me.x = x; me.y = y; me.z = z; me.yaw = yaw;
  posFromGame = true; placeListener();
  if (!inGame) {
    inGame = true; refreshStatus();
    RT.WindowShow && RT.WindowShow(); // aparece com o jogo, como overlay
    setCompact(true);
    if (cfg.autoConnect) onGameEnter();
  }
}

// ao entrar no jogo: auto-detecta o IP do servidor atual; senao, servidor salvo.
// DetectGameServerIP tenta live (mod) -> save (PalOptionSaveGame, pega Direct
// Connect E lista do Steam) -> ini (so Direct Connect). Antes era so o ini.
async function onGameEnter() {
  if (ws) return; // ja conectado -> nao abre uma 2a conexao
  if (cfg.autoDetect) {
    let ip = '';
    try { ip = await window.go.main.App.DetectGameServerIP(); } catch (_) {}
    if (ip) {
      log(t('lg_game_detected', {ip}));
      // acha a PORTA do voz sozinho: testa portas comuns e usa a 1a que responde como
      // PalProxVoice (handshake auth->hello). autoPort (se setado) vai primeiro.
      const ports = [...new Set([cfg.autoPort, 8765, 8766, 8767, 8768].filter(Boolean))];
      const port = await probeVoicePort(ip, ports, cfg.autoPassword || '');
      if (port) {
        log('voz encontrado em ' + ip + ':' + port);
        start({ name: 'auto ' + ip, url: 'ws://' + ip + ':' + port, password: cfg.autoPassword || '', voiceRangeMeters: 50 });
        return;
      }
      log('nenhum voz respondeu (portas ' + ports.join(',') + ')');
    }
    log(t('lg_no_ip'));
  }
  connectSelected();
}

// testa as portas em sequencia; resolve com a 1a que faz o handshake do PalProxVoice
// (recebe "hello"). Conexao descartavel e leve (so ws+auth, sem WebRTC). null = nenhuma.
function probeVoicePort(ip, ports, password) {
  return new Promise(resolve => {
    let i = 0;
    const tryNext = () => {
      if (i >= ports.length) { resolve(null); return; }
      const port = ports[i++];
      let done = false, sock = null;
      const finish = ok => { if (done) return; done = true; clearTimeout(to); try { sock && sock.close(); } catch (_) {} ok ? resolve(port) : tryNext(); };
      const to = setTimeout(() => finish(false), 1500);
      try { sock = new WebSocket('ws://' + ip + ':' + port + '/ws'); } catch (_) { return finish(false); }
      sock.onopen = () => { try { sock.send(JSON.stringify({ event: 'auth', data: password, user: '' })); } catch (_) {} };
      sock.onmessage = ev => { try { const m = JSON.parse(ev.data); if (m.event === 'hello') return finish(true); if (m.event === 'error') return finish(false); } catch (_) {} };
      sock.onerror = () => finish(false);
      sock.onclose = () => finish(false);
    };
    tryNext();
  });
}
if (window.runtime && window.runtime.EventsOn) {
  window.runtime.EventsOn('pos', onPos);
  // (re)entrou num mundo -> HUD pequeno de volta, mesmo se tinha minimizado
  window.runtime.EventsOn('gameEnter', () => { RT.WindowShow && RT.WindowShow(); setCompact(true); });
  window.runtime.EventsOn('posLost', () => {
    inGame = false; posFromGame = false; refreshStatus();
    wantConnected = false; wasEverConnected = false; clearTimeout(reconnectTimer); // saiu do jogo -> nao reconecta
    if (ws) { log(t('lg_left_server')); stop(); }
    setOverlayMode(false);
    RT.WindowHide && RT.WindowHide(); // saiu do jogo -> some (sem aba na taskbar)
  });
  // reabrir o .exe traz a config de volta (single-instance no Go)
  window.runtime.EventsOn('showConfig', () => { RT.WindowShow && RT.WindowShow(); setCompact(false); });
} else { log(t('lg_no_wails')); }

// ----- WebSocket / WebRTC -----
let ws = null, pc = null, posTimer = null, identifyTimer = null, gotHello = false;
// reconexao automatica em QUEDA DE REDE (internet ruim/instavel)
let wantConnected = false, wasEverConnected = false, lastServer = null, reconnectTimer = null, reconnectDelay = 2000;

// qualidade de audio: liga FEC (corrige perda de pacote), forca mono e bitrate alvo
function tuneOpus(sdp) {
  const m = sdp.match(/a=rtpmap:(\d+)\s+opus/i); if (!m) return sdp;
  const pt = m[1];
  const re = new RegExp('(a=fmtp:' + pt + ' )([^\\r\\n]*)');
  if (re.test(sdp)) {
    return sdp.replace(re, (_, h, params) => {
      const kv = params.split(';').filter(Boolean);
      const set = (k, v) => { const i = kv.findIndex(x => x.trim().startsWith(k + '=')); if (i >= 0) kv[i] = k + '=' + v; else kv.push(k + '=' + v); };
      set('useinbandfec', '1'); set('usedtx', '1'); set('stereo', '0'); set('sprop-stereo', '0'); set('maxaveragebitrate', '48000');
      return h + kv.join(';');
    });
  }
  return sdp.replace(new RegExp('(a=rtpmap:' + pt + '\\s+opus[^\\r\\n]*\\r?\\n)'),
    '$1a=fmtp:' + pt + ' minptime=10;useinbandfec=1;usedtx=1;stereo=0;sprop-stereo=0;maxaveragebitrate=48000\r\n');
}
// define o teto de bitrate do mic enviado (ao vivo, via setParameters)
async function applyBitrate(br) {
  if (!pc) return;
  const s = pc.getSenders().find(x => x.track && x.track.kind === 'audio');
  if (!s) return;
  const p = s.getParameters(); if (!p.encodings || !p.encodings.length) p.encodings = [{}];
  p.encodings[0].maxBitrate = br;
  try { await s.setParameters(p); } catch (_) {}
}
async function setAudioBitrate() { await applyBitrate(curBitrate); }

// ----- adaptacao automatica de bitrate (internet ruim) -----
// le perda de pacote + RTT reais (getStats) e baixa o bitrate sozinho quando a rede
// piora, subindo de volta devagar quando limpa. Histerese evita oscilar.
const BR_LEVELS = [16000, 24000, 32000, 48000]; // degraus de bitrate
let curBitrate = 48000, netTimer = null, cleanStreak = 0;
function startNetAdapt() {
  stopNetAdapt();
  netTimer = setInterval(async () => {
    if (!pc) return;
    let loss = 0, rtt = 0, have = false;
    try {
      (await pc.getStats()).forEach(r => {
        if (r.type === 'remote-inbound-rtp' && r.kind === 'audio') {
          if (typeof r.fractionLost === 'number') { loss = Math.max(loss, r.fractionLost); have = true; }
          if (typeof r.roundTripTime === 'number') rtt = Math.max(rtt, r.roundTripTime);
        }
      });
    } catch (_) {}
    if (!have) return;
    let i = BR_LEVELS.indexOf(curBitrate); if (i < 0) i = BR_LEVELS.length - 1;
    if (loss > 0.05 || rtt > 0.4) { i = Math.max(0, i - 1); cleanStreak = 0; }            // rede ruim -> desce ja
    else if (loss < 0.01) { if (++cleanStreak >= 3) { i = Math.min(BR_LEVELS.length - 1, i + 1); cleanStreak = 0; } } // limpa -> sobe devagar
    else cleanStreak = 0;
    const target = BR_LEVELS[i];
    if (target !== curBitrate) {
      curBitrate = target; applyBitrate(target);
      log(t('lg_bitrate', { kbps: target / 1000, loss: (loss * 100).toFixed(0), rtt: (rtt * 1000) | 0 }), 'info');
    }
  }, 3000);
}
function stopNetAdapt() { if (netTimer) clearInterval(netTimer); netTimer = null; }

function setConn(s) { connState = s; refreshStatus(); }
function showFallback(name) { $('fbName').textContent = name || ''; $('fallback').hidden = false; }
function hideFallback() { $('fallback').hidden = true; }
function showSettings() { try { $('cfgServer').focus(); $('cfgServer').scrollIntoView({ block: 'center' }); } catch (_) {} }

function wsURLFrom(serverURL) {
  let u = (serverURL || '').trim(); if (!u) return '';
  if (u.startsWith('https://'))      u = 'wss://' + u.slice(8);
  else if (u.startsWith('http://'))  u = 'ws://'  + u.slice(7);
  else if (!u.startsWith('ws://') && !u.startsWith('wss://')) u = 'ws://' + u;
  u = u.replace(/\/+$/, ''); if (!u.endsWith('/ws')) u += '/ws';
  return u;
}

function selectedServer() { return cfg.servers[cfg.selected] || null; }

function connectSelected() {
  const s = selectedServer();
  if (!s || !s.url) { showSettings(); log(t('lg_configure_server')); return; }
  start(s);
}

async function start(s) {
  if (ws) return;
  const url = wsURLFrom(s.url);
  if (!url) { log(t('lg_no_url')); return; } // senha vazia = passwordless (ok)
  wantConnected = true; lastServer = s; // intencao de estar conectado (p/ reconectar em queda)
  hideFallback();
  gotHello = false;
  voiceRangeMeters = s.voiceRangeMeters || 50;
  setConn('connecting');

  if (!actx) {
    // 48kHz pra casar com a captura nativa (mic-feed). Carrega o AudioWorklet do mic.
    actx = new (window.AudioContext || window.webkitAudioContext)({ sampleRate: 48000 });
    listener = actx.listener;
    masterGain = actx.createGain(); masterGain.gain.value = deafened ? 0 : masterVolume; masterGain.connect(actx.destination);
    try { await actx.audioWorklet.addModule('mic-feed.js'); } catch (e) { log(t('lg_worklet_fail', {e}), 'err'); }
  }
  if (actx.state === 'suspended') { try { await actx.resume(); } catch (_) {} }
  // saida no device escolhido na config (vazio = device padrao do sistema)
  if (actx.setSinkId) { try { await actx.setSinkId(cfg.outputDeviceId || ''); } catch (_) {} }
  placeListener();

  ws = new WebSocket(url);
  pc = new RTCPeerConnection({ iceServers: [{ urls: 'stun:stun.l.google.com:19302' }] });

  pc.ontrack = e => {
    const id = e.streams[0].id;
    if (peers[id] && peers[id].panner) return;
    const src = actx.createMediaStreamSource(e.streams[0]);
    const panner = new PannerNode(actx, { panningModel: 'HRTF', distanceModel: 'inverse', refDistance: 5, maxDistance: voiceRangeMeters, rolloffFactor: 0.7 });
    // dois caminhos: HRTF (direcional, longe) + DIRETO/centralizado (presente, colado).
    // o HRTF "externaliza" e confunde frente/tras no curto alcance; o caminho direto
    // domina quando perto pra a voz soar "do seu lado/na cabeca", sem parecer atras.
    const pannerGain = actx.createGain();
    const directGain = actx.createGain(); directGain.gain.value = 0;
    // low-pass no caminho HRTF: abafa quem esta ATRAS (reforca frente/tras, reduz a
    // confusao classica do HRTF) — frente = aberto, atras = abafado.
    const lp = new BiquadFilterNode(actx, { type: 'lowpass', frequency: 18000, Q: 0.5 });
    src.connect(panner).connect(lp).connect(pannerGain).connect(masterGain);
    src.connect(directGain).connect(masterGain);
    const flatGain = actx.createGain(); flatGain.gain.value = 0; // canal guild/global: som PLANO (sem panner)
    src.connect(flatGain).connect(masterGain);
    const a = new Audio(); a.muted = true; a.srcObject = e.streams[0]; // keep-alive do stream
    if (a.setSinkId) { a.setSinkId(cfg.outputDeviceId || 'default').catch(() => {}); }
    peers[id] = Object.assign(peers[id] || { x:0,y:0,z:0,yaw:0 }, { panner, pannerGain, directGain, lp, flatGain, audio: a });
    applyPeerChannel(peers[id]); updatePlayers(); log(t('lg_hearing_peer', { id: id.slice(0,8) }), 'ok');
  };
  pc.onicecandidate = e => { if (e.candidate) ws.send(JSON.stringify({ event: 'candidate', data: JSON.stringify(e.candidate) })); };

  ws.onopen = async () => {
    const sock = ws; // fixa o socket desta conexao: onopen e' async e o ws (module-level)
                     // pode ser trocado/anulado por stop()/reconnect durante os awaits.
    // anti-spoof: manda o FGuid do player (escrito pelo mod) no auth. O servidor
    // correlaciona com a REST do Palworld. Vazio = ainda nao entrou no mundo ->
    // o servidor cai pra correlacao por IP+proximidade. Auth e' SEMPRE a 1a msg.
    let uid = '';
    try { uid = await window.go.main.App.PlayerID(); } catch (_) {}
    if (sock.readyState !== 1) return; // desconectou durante o await -> aborta sem erro
    sock.send(JSON.stringify({ event: 'auth', data: s.password, user: uid || '' }));
    sendMeta(); // canais: manda guild (codigo/auto) + canal atual (proximity no inicio)
    // o FGuid demora ~6s pra replicar apos entrar no mundo, mas o auto-connect
    // dispara em ~50ms -> o auth quase sempre vai com user vazio. Entao, ate ter um
    // id, fica relendo e manda um {identify} quando ele aparecer (uma vez). Tambem
    // cobre o caso de conectar a voz ANTES de entrar no mundo.
    if (identifyTimer) { clearInterval(identifyTimer); identifyTimer = null; }
    let sentUser = uid || '';
    if (!sentUser) {
      // handle local (h): o callback so mexe no PROPRIO timer/socket, nunca no de
      // outra conexao (evita um callback atrasado matar o timer de um reconnect).
      const h = setInterval(async () => {
        if (sock.readyState !== 1) { clearInterval(h); if (identifyTimer === h) identifyTimer = null; return; }
        let id = '';
        try { id = await window.go.main.App.PlayerID(); } catch (_) {}
        if (id && id !== sentUser) {
          sentUser = id;
          if (sock.readyState === 1) sock.send(JSON.stringify({ event: 'identify', user: id }));
          clearInterval(h); if (identifyTimer === h) identifyTimer = null;
        }
      }, 1000);
      identifyTimer = h;
    }
    try {
      // MIC NATIVO: a captura roda em Go via WASAPI com AudioCategory_Other (NAO
      // Communications), entregue por um WS local de PCM mono float32 @48k. Sem
      // getUserMedia -> o codec nao entra em "modo comunicacao" -> nao degrada o
      // resto do audio do sistema. (Esse era o problema do WebRTC do WebView2.)
      const purl = await window.go.main.App.StartMicCapture(cfg.micDeviceId || '');
      if (!purl || purl.startsWith('erro')) throw new Error(purl || 'sem captura nativa');
      const micNode = new AudioWorkletNode(actx, 'mic-feed', { numberOfInputs: 0, numberOfOutputs: 1, outputChannelCount: [1] });
      // limpeza do mic (sem reabrir o problema do codec, pois roda no WebAudio/worklet):
      // passa-alta corta zumbido/hum; compressor nivela a voz. O noise gate (silencia
      // o ruido de fundo no silencio) mora no worklet mic-feed.js.
      const hp   = new BiquadFilterNode(actx, { type: 'highpass', frequency: 90, Q: 0.707 });
      const comp = new DynamicsCompressorNode(actx, { threshold: -28, knee: 18, ratio: 3, attack: 0.004, release: 0.18 });
      micGain = actx.createGain(); micGain.gain.value = micMuted ? 0 : inputVol;
      const micDest = actx.createMediaStreamDestination();
      const analyser = actx.createAnalyser(); analyser.fftSize = 1024;
      const monitorGain = actx.createGain(); monitorGain.gain.value = 0; monitorGain.connect(actx.destination);
      micProc = { micNode, hp, comp, micGain, micDest, analyser, monitorGain };
      micDest.stream.getAudioTracks().forEach(t => pc.addTrack(t, micDest.stream));
      if (proc.rnnoise) await initRNNoise();
      applyProc();   // monta a cadeia conforme os toggles + envia gate/sensibilidade ao worklet
      startMeter();
      // recebe o PCM nativo do Go -> [RNNoise] -> worklet
      micWs = new WebSocket(purl);
      micWs.binaryType = 'arraybuffer';
      micWs.onmessage = ev => { let f = new Float32Array(ev.data);
        if (proc.rnnoise && rnn && f.length === rnn.FRAME) f = rnn.process(f); // supressao IA
        micNode.port.postMessage(f, [f.buffer]); };
      micWs.onerror = () => log(t('lg_mic_ws_err'), 'warn');
      populateDevices();
      log(t('lg_mic_ok'), 'ok');
    } catch (err) { log(t('lg_no_mic', { err: err.name }), 'err'); }
    setConn('on');
    posTimer = setInterval(() => {
      if (ws && ws.readyState === 1)
        ws.send(JSON.stringify({ event: 'pos', data: `${me.x.toFixed(1)},${me.y.toFixed(1)},${me.z.toFixed(1)},${me.yaw.toFixed(1)}` }));
    }, 50); // 20Hz: dados mais finos; o setTargetAtTime suaviza o intervalo
  };

  ws.onmessage = async ev => {
    const m = JSON.parse(ev.data);
    if (m.event === 'error')      { log(t('lg_error', { data: m.data }), 'err'); wantConnected = false; return; } // senha errada etc -> nao reconecta
    if (m.event === 'hello')      { gotHello = true; wasEverConnected = true; reconnectDelay = 2000; hideFallback(); log(t('lg_connected', { id: m.data.slice(0,8) })); return; }
    if (m.event === 'serverinfo') { // "no compose e o padrao": nome + alcance do servidor
      try { const si = JSON.parse(m.data); $('mServer').textContent = si.name || '—';
            const r = parseFloat(si.range); if (r) applyRange(r); } catch (_) {} return;
    }
    if (m.event === 'offer') {
      const off = JSON.parse(m.data); off.sdp = tuneOpus(off.sdp); // FEC/mono no encoder local
      await pc.setRemoteDescription(off);
      const ans = await pc.createAnswer(); ans.sdp = tuneOpus(ans.sdp);
      await pc.setLocalDescription(ans);
      ws.send(JSON.stringify({ event: 'answer', data: JSON.stringify(ans) }));
      setAudioBitrate(); startNetAdapt(); // bitrate inicial + adaptacao automatica p/ rede ruim
    }
    if (m.event === 'candidate') { try { await pc.addIceCandidate(JSON.parse(m.data)); } catch (_) {} }
    if (m.event === 'pos') {
      const [x, y, z, yaw] = m.data.split(',').map(Number);
      applyPeerChannel(peers[m.id] = Object.assign(peers[m.id] || {}, { x, y, z, yaw }));
    }
    if (m.event === 'peermeta') { // canais: guild + canal atual de OUTRO peer
      try { const md = JSON.parse(m.data);
            applyPeerChannel(peers[m.id] = Object.assign(peers[m.id] || {}, { guild: md.guild, channel: md.channel })); } catch (_) {}
    }
  };

  ws.onerror = () => { log(t('lg_ws_err')); };
  ws.onclose = () => {
    stop();
    if (wantConnected && wasEverConnected && lastServer) {
      // QUEDA DE REDE: reconecta sozinho com backoff. Nao dispara em saida manual,
      // sair do jogo (posLost) ou senha errada (esses zeram wantConnected).
      log(t('lg_reconnecting', { s: Math.round(reconnectDelay / 1000) }), 'warn');
      setConn('connecting');
      clearTimeout(reconnectTimer);
      reconnectTimer = setTimeout(() => { if (wantConnected) start(lastServer); }, reconnectDelay);
      reconnectDelay = Math.min(Math.round(reconnectDelay * 1.6), 15000);
    } else if (!wasEverConnected) {
      const s2 = selectedServer(); showFallback(s2 ? s2.name : ''); log(t('lg_not_found'));
    }
  };
}

function stop() {
  if (posTimer) { clearInterval(posTimer); posTimer = null; }
  if (identifyTimer) { clearInterval(identifyTimer); identifyTimer = null; }
  if (pc) { try { pc.close(); } catch (_) {} pc = null; }
  if (ws) { try { ws.close(); } catch (_) {} ws = null; }
  for (const id in peers) {
    try { if (peers[id].audio) peers[id].audio.srcObject = null; } catch (_) {} // solta o <audio> keep-alive
    try { peers[id].panner && peers[id].panner.disconnect(); } catch (_) {}
    delete peers[id];
  }
  // para a captura NATIVA (Go/WASAPI) e fecha o WS de PCM
  if (micWs) { try { micWs.close(); } catch (_) {} micWs = null; }
  stopNetAdapt(); curBitrate = 48000;
  stopMeter(); micProc = null;
  try { window.go.main.App.StopMicCapture(); } catch (_) {}
  // fecha o AudioContext (start() recria no proximo connect, recarregando o worklet).
  if (actx) { try { actx.close(); } catch (_) {} actx = null; listener = null; masterGain = null; micGain = null; }
  updatePlayers();
  setConn('off');
}

// ----- processamento do mic (toggles + gate + sensibilidade + medidor + monitor) -----
// remonta a cadeia: micNode(worklet) -> [passa-alta] -> [compressor] -> micGain -> micDest(WebRTC)
// + tap pro medidor (nivel de ENTRADA) + saida de monitor (ouvir o proprio mic).
function applyProc(){
  if (!micProc) return;
  const { micNode, hp, comp, micGain, micDest, analyser, monitorGain } = micProc;
  [micNode, hp, comp, micGain].forEach(n => { try { n.disconnect(); } catch (_) {} });
  let node = micNode;
  if (proc.highpass) { node.connect(hp); node = hp; }
  if (proc.comp)     { node.connect(comp); node = comp; }
  node.connect(micGain);
  micGain.connect(micDest);            // -> WebRTC (o que os outros ouvem)
  micGain.connect(monitorGain);        // -> monitor (voce se ouvir)
  micNode.connect(analyser);           // tap cru pro medidor de nivel
  monitorGain.gain.value = proc.monitor ? 1 : 0;
  micNode.port.postMessage({ gate: proc.gate, open: sensToThreshold(proc.sens) }); // gate no worklet
}
function startMeter(){
  stopMeter(); if (!micProc) return;
  const an = micProc.analyser, buf = new Float32Array(an.fftSize);
  const fill = $('meterFill'), thr = $('meterThr');
  const tick = () => {
    an.getFloatTimeDomainData(buf);
    let sum = 0; for (let i = 0; i < buf.length; i++) sum += buf[i] * buf[i];
    const db = 20 * Math.log10(Math.sqrt(sum / buf.length) + 1e-9);
    if (fill) fill.style.width = Math.max(0, Math.min(100, (db + 60) / 60 * 100)) + '%';
    const tdb = 20 * Math.log10(sensToThreshold(proc.sens) + 1e-9);
    if (thr) thr.style.left = Math.max(0, Math.min(100, (tdb + 60) / 60 * 100)) + '%';
    meterRAF = requestAnimationFrame(tick);
  };
  tick();
}
function stopMeter(){ if (meterRAF) cancelAnimationFrame(meterRAF); meterRAF = 0;
  const fill = $('meterFill'); if (fill) fill.style.width = '0%'; }
function bindProcUI(){
  const c = id => $(id);
  if (c('pHighpass')) c('pHighpass').checked = proc.highpass;
  if (c('pComp'))     c('pComp').checked     = proc.comp;
  if (c('pGate'))     c('pGate').checked     = proc.gate;
  if (c('pRnnoise'))  c('pRnnoise').checked  = proc.rnnoise;
  if (c('pMonitor'))  c('pMonitor').checked  = proc.monitor;
  if (c('pSens'))   { c('pSens').value = proc.sens; c('pSensVal').textContent = proc.sens; }
  c('pHighpass') && (c('pHighpass').onchange = e => { proc.highpass = e.target.checked; saveProc(); applyProc(); });
  c('pComp')     && (c('pComp').onchange     = e => { proc.comp = e.target.checked; saveProc(); applyProc(); });
  c('pGate')     && (c('pGate').onchange     = e => { proc.gate = e.target.checked; saveProc(); applyProc(); });
  c('pRnnoise')  && (c('pRnnoise').onchange  = async e => { proc.rnnoise = e.target.checked; saveProc(); if (proc.rnnoise) { c('pRnnoise').disabled = true; await initRNNoise(); c('pRnnoise').disabled = false; } });
  c('pMonitor')  && (c('pMonitor').onchange  = e => { proc.monitor = e.target.checked; if (micProc) micProc.monitorGain.gain.value = proc.monitor ? 1 : 0; });
  c('pSens')     && (c('pSens').oninput      = e => { proc.sens = +e.target.value; c('pSensVal').textContent = proc.sens; saveProc(); applyProc(); });
}
bindProcUI();
bindChannelUI();

// ----- config / servidores -----
function applyOutput() { if (masterGain) masterGain.gain.value = deafened ? 0 : masterVolume; }
function applyInput()  { if (micGain) micGain.gain.value = micMuted ? 0 : inputVol; }
function applyVolume(v)   { masterVolume = v; applyOutput(); }
function applyInputVol(v) { inputVol = v; applyInput(); }
function updateAudioBtns() {
  const mm = $('muteMic'), df = $('deafen');
  mm.textContent = micMuted ? t('mute_off') : t('mute_on'); mm.classList.toggle('muted', micMuted);
  df.textContent = deafened ? t('deaf_off') : t('deaf_on'); df.classList.toggle('muted', deafened);
}
function applyRange(m) { voiceRangeMeters = m; for (const id in peers) { const p = peers[id]; if (p.panner) p.panner.maxDistance = m; } }

function renderServerList() {
  const sel = $('cfgServerSel'); sel.innerHTML = '';
  cfg.servers.forEach((s, i) => {
    const o = document.createElement('option'); o.value = i; o.textContent = s.name || s.url || ('servidor ' + (i+1));
    if (i === cfg.selected) o.selected = true; sel.appendChild(o);
  });
  if (cfg.servers.length === 0) { const o = document.createElement('option'); o.textContent = '(nenhum — clique + Novo)'; sel.appendChild(o); }
}
function fillServerFields(s) {
  s = s || { name:'', url:'', password:'', voiceRangeMeters:50 };
  $('cfgName').value = s.name || ''; $('cfgServer').value = s.url || '';
  $('cfgPassword').value = s.password || ''; $('cfgRange').value = s.voiceRangeMeters || 50;
}
function readServerFields() {
  return { name: $('cfgName').value.trim(), url: $('cfgServer').value.trim(),
           password: $('cfgPassword').value, voiceRangeMeters: parseFloat($('cfgRange').value) || 50 };
}

// ----- dispositivos -----
// Os NOMES dos devices de SAIDA vem do navegador, que so revela os labels apos uma
// permissao de mic. Como a captura virou nativa (sem getUserMedia), destravamos os
// labels UMA vez (rapido, fora da chamada): abre o mic e fecha na hora. Nao reintroduz
// o problema do codec na voz (a chamada segue 100% nativa).
let labelsUnlocked = false;
async function unlockDeviceLabels() {
  if (labelsUnlocked) return;
  try { const t = await navigator.mediaDevices.getUserMedia({ audio: true }); t.getTracks().forEach(x => x.stop()); labelsUnlocked = true; }
  catch (_) {}
}
async function populateDevices() {
  const mic = $('cfgMic'), out = $('cfgOutput'); mic.innerHTML = ''; out.innerHTML = '';
  const addOpt = (sel, id, label, selId) => { const o = document.createElement('option'); o.value = id;
    o.textContent = label || (id ? id.slice(0,10) : 'Padrão'); if (id === selId) o.selected = true; sel.appendChild(o); };
  // MIC: devices NATIVOS do Windows (WASAPI, via Go) — e' o que a captura realmente usa.
  addOpt(mic, '', 'Padrão do Windows', cfg.micDeviceId);
  try { const mics = await window.go.main.App.ListMicDevices();
    (mics || []).forEach(d => addOpt(mic, d.id, d.name, cfg.micDeviceId)); } catch (_) {}
  // SAIDA: devices do navegador (a saida usa setSinkId/WebAudio).
  addOpt(out, '', 'Padrão', cfg.outputDeviceId);
  let devices = []; try { devices = await navigator.mediaDevices.enumerateDevices(); } catch (_) {}
  let hasLabels = false;
  devices.forEach(d => { if (d.kind === 'audiooutput') { if (d.label) hasLabels = true; addOpt(out, d.deviceId, d.label, cfg.outputDeviceId); } });
  $('cfgDevHint').textContent = hasLabels ? '' : '"Atualizar dispositivos" pra ver os nomes da saída';
}
async function refreshDevices() {
  labelsUnlocked = false; await unlockDeviceLabels(); // re-scan: força destravar os nomes
  await populateDevices();
}

// pega TUDO da UI -> objeto cfg
function gatherConfig() {
  if (cfg.servers.length === 0) cfg.servers.push(readServerFields());
  else cfg.servers[cfg.selected] = readServerFields();
  cfg.volume = parseFloat($('cfgVolume').value);
  cfg.inputVolume = parseFloat($('cfgInputVol').value);
  cfg.micDeviceId = $('cfgMic').value || '';
  cfg.outputDeviceId = $('cfgOutput').value || '';
  cfg.autoConnect = $('cfgAuto').checked;
  cfg.autoDetect = $('cfgAutoDetect').checked;
  cfg.autoPort = parseInt($('cfgAutoPort').value) || 8765;
  cfg.autoPassword = $('cfgAutoPassword').value;
  return cfg;
}

// ----- UI wiring -----
$('cfgVolume').addEventListener('input', e => { const v = parseFloat(e.target.value); $('cfgVolumeVal').textContent = Math.round(v*100)+'%'; applyVolume(v); });
$('cfgInputVol').addEventListener('input', e => { const v = parseFloat(e.target.value); $('cfgInputVolVal').textContent = Math.round(v*100)+'%'; applyInputVol(v); });
$('muteMic').onclick = () => { micMuted = !micMuted; applyInput(); updateAudioBtns(); };
$('deafen').onclick  = () => { deafened = !deafened; applyOutput(); updateAudioBtns(); };
$('cfgRefreshDevices').addEventListener('click', refreshDevices);
// mic escolhido aplica na proxima conexao (reconecte pra trocar o device em uso)
$('cfgMic').addEventListener('change', e => { cfg.micDeviceId = e.target.value || ''; });
$('cfgServerSel').addEventListener('change', e => { cfg.selected = parseInt(e.target.value) || 0; fillServerFields(selectedServer()); });
$('cfgAdd').addEventListener('click', () => { cfg.servers.push({ name:'Novo', url:'', password:'', voiceRangeMeters:50 }); cfg.selected = cfg.servers.length-1; renderServerList(); fillServerFields(selectedServer()); });
$('cfgDel').addEventListener('click', () => { if (!cfg.servers.length) return; cfg.servers.splice(cfg.selected,1); cfg.selected = 0; renderServerList(); fillServerFields(selectedServer()); });

$('cfgSave').addEventListener('click', async () => {
  gatherConfig(); applyVolume(cfg.volume); renderServerList();
  if (cfg.outputDeviceId && actx && actx.setSinkId) { try { await actx.setSinkId(cfg.outputDeviceId); } catch (_) {} }
  try { await window.go.main.App.SaveConfig(cfg); $('cfgSaved').textContent = 'salvo ✓'; setTimeout(() => $('cfgSaved').textContent = '', 2000); }
  catch (err) { log(t('lg_save_err', {err})); }
});

$('fbConfig').addEventListener('click', () => { hideFallback(); showSettings(); });
$('fbIgnore').addEventListener('click', hideFallback);

// botoes manuais: Conectar (servidor selecionado) / Desconectar
$('go').onclick = () => { hideFallback(); connectSelected(); };
$('leave').onclick = () => { wantConnected = false; clearTimeout(reconnectTimer); stop(); }; // Sair manual -> nao reconecta

// ----- boot -----
(async () => {
  try {
    cfg = await window.go.main.App.GetConfig();
    if (!cfg.servers) cfg.servers = [];
    $('cfgLang').value = lang; $('cfgLang').onchange = e => setLang(e.target.value);
    applyI18n(); // traduz a UI pro idioma salvo/do sistema
    masterVolume = (cfg.volume == null) ? 1.0 : cfg.volume;
    $('cfgVolume').value = masterVolume; $('cfgVolumeVal').textContent = Math.round(masterVolume*100)+'%';
    inputVol = (cfg.inputVolume == null) ? 1.0 : cfg.inputVolume;
    $('cfgInputVol').value = inputVol; $('cfgInputVolVal').textContent = Math.round(inputVol*100)+'%';
    $('cfgAuto').checked = cfg.autoConnect !== false;
    $('cfgAutoDetect').checked = !!cfg.autoDetect;
    $('cfgAutoPort').value = cfg.autoPort || 8765;
    $('cfgAutoPassword').value = cfg.autoPassword || '';
    renderServerList(); fillServerFields(selectedServer());
    await unlockDeviceLabels(); // destrava os nomes da saída uma vez (fora da chamada)
    await populateDevices();
    const s = selectedServer();
    if (s) applyRange(s.voiceRangeMeters || 50);
    log(t('lg_ready'));
    applyOverlayStyle(); // tira a aba da taskbar ja na abertura
    if (!cfg.autoDetect && (!s || !s.url || !s.password)) { showSettings(); log(t('lg_add_server')); }
  } catch (err) {
    log(t('lg_cfg_err', {err})); showSettings();
  }
})();
