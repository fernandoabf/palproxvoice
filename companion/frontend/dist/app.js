// PalProxVoice Companion — frontend (Wails v2)
// Padrao: auto-conecta no servidor selecionado ao entrar no jogo, config escondida.
// Se falhar -> mostra fallback (Configurar/Ignorar). Multi-servidor fica na config.

const me = { x: 0, y: 0, z: 0, yaw: 0 };
const CM = 100;
const peers = {};
let posFromGame = false;
let inGame = false;

let voiceRangeMeters = 50;
let masterVolume = 1.0;

// config completa (lista de servidores + global)
let cfg = { servers: [], selected: 0, volume: 1.0, micDeviceId: '', outputDeviceId: '', autoConnect: true };

const $ = id => document.getElementById(id);
const log = m => { const e = $('log'); e.textContent += m + "\n"; e.scrollTop = e.scrollHeight; };

const fwd = yaw => { const r = yaw * Math.PI / 180; return [Math.cos(r), 0, Math.sin(r)]; };
(() => { const [fx,,fz] = fwd(0); console.assert(Math.abs(fx-1)<1e-9 && Math.abs(fz)<1e-9, "fwd(0) +X"); })();

let actx, listener, masterGain;
function audioPos(p) { return [p.x/CM, p.z/CM, p.y/CM]; }
function placeListener() {
  if (!listener) return;
  const [ax, ay, az] = audioPos(me); const [fx, fy, fz] = fwd(me.yaw);
  if (listener.positionX) {
    listener.positionX.value = ax; listener.positionY.value = ay; listener.positionZ.value = az;
    listener.forwardX.value = fx; listener.forwardY.value = fy; listener.forwardZ.value = fz;
    listener.upX.value = 0; listener.upY.value = 1; listener.upZ.value = 0;
  } else { listener.setPosition(ax, ay, az); listener.setOrientation(fx, fy, fz, 0, 1, 0); }
}
function placePanner(p) {
  if (!p.panner) return;
  const [ax, ay, az] = audioPos(p);
  if (p.panner.positionX) { p.panner.positionX.value = ax; p.panner.positionY.value = ay; p.panner.positionZ.value = az; }
  else p.panner.setPosition(ax, ay, az);
}

// ----- mapa top-down centrado em voce -----
const cv = $('stage'), ctx = cv.getContext('2d');
function draw() {
  ctx.clearRect(0, 0, cv.width, cv.height);
  const cx = cv.width/2, cy = cv.height/2, S = 0.03;
  const dot = (p, color, label) => {
    const X = cx + (p.x - me.x)*S, Y = cy + (p.y - me.y)*S;
    ctx.fillStyle = color; ctx.beginPath(); ctx.arc(X, Y, 7, 0, 7); ctx.fill();
    const [fx,,fz] = fwd(p.yaw);
    ctx.strokeStyle = color; ctx.beginPath(); ctx.moveTo(X, Y); ctx.lineTo(X+fx*16, Y+fz*16); ctx.stroke();
    ctx.fillStyle = '#cdd6e0'; ctx.font = '11px monospace'; ctx.fillText(label, X+10, Y-8);
  };
  for (const id in peers) if (peers[id].panner) dot(peers[id], '#e06363', id.slice(0,4));
  dot(me, '#5aa9e6', 'voce');
  $('status').textContent = posFromGame ? 'no jogo' : 'aguardando o jogo…';
  requestAnimationFrame(draw);
}
requestAnimationFrame(draw);

// ----- eventos do backend: "pos" e "posLost" -----
function onPos(data) {
  if (!data) return;
  const t = String(data).trim(); if (!t) return;
  const [x, y, z, yaw] = t.split(',').map(Number);
  if ([x, y, z, yaw].some(Number.isNaN)) return;
  me.x = x; me.y = y; me.z = z; me.yaw = yaw;
  posFromGame = true; placeListener();
  if (!inGame) { inGame = true; if (cfg.autoConnect) onGameEnter(); }
}

// ao entrar no jogo: tenta auto-detectar o IP (Direct Connect); senao, servidor salvo
async function onGameEnter() {
  if (cfg.autoDetect) {
    let ip = '';
    try { ip = await window.go.main.App.GameServerIP(); } catch (_) {}
    if (ip) {
      log('servidor do jogo detectado: ' + ip);
      start({ name: 'auto ' + ip, url: 'ws://' + ip + ':' + (cfg.autoPort || 8765), password: cfg.autoPassword || '', voiceRangeMeters: 50 });
      return;
    }
    log('não detectei o IP (entrou pela lista?) — usando servidor salvo');
  }
  connectSelected();
}
if (window.runtime && window.runtime.EventsOn) {
  window.runtime.EventsOn('pos', onPos);
  window.runtime.EventsOn('posLost', () => {
    inGame = false; posFromGame = false;
    if (ws) { log('saiu do servidor — voz desconectada'); stop(); }
  });
} else { log('aviso: fora do Wails (sem posição do jogo)'); }

// ----- WebSocket / WebRTC -----
let ws = null, pc = null, posTimer = null, gotHello = false;

function setConn(state, label) {
  const el = $('conn'); el.textContent = label; el.className = 'badge ' + (state ? 'on' : 'off');
  $('go').disabled = state;
  $('leave').disabled = !state;
}
function showFallback(name) { $('fbName').textContent = name || ''; $('fallback').classList.add('show'); }
function hideFallback() { $('fallback').classList.remove('show'); }

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
  if (!s || !s.url) { $('cfgSection').open = true; log('configure um servidor'); return; }
  start(s);
}

async function start(s) {
  if (ws) return;
  const url = wsURLFrom(s.url);
  if (!url) { log('servidor sem URL'); return; } // senha vazia = passwordless (ok)
  hideFallback();
  gotHello = false;
  voiceRangeMeters = s.voiceRangeMeters || 50;
  setConn(false, 'conectando…');

  if (!actx) {
    actx = new (window.AudioContext || window.webkitAudioContext)();
    listener = actx.listener;
    masterGain = actx.createGain(); masterGain.gain.value = masterVolume; masterGain.connect(actx.destination);
  }
  if (actx.state === 'suspended') { try { await actx.resume(); } catch (_) {} }
  if (cfg.outputDeviceId && actx.setSinkId) { try { await actx.setSinkId(cfg.outputDeviceId); } catch (_) {} }
  placeListener();

  ws = new WebSocket(url);
  pc = new RTCPeerConnection({ iceServers: [{ urls: 'stun:stun.l.google.com:19302' }] });

  pc.ontrack = e => {
    const id = e.streams[0].id;
    if (peers[id] && peers[id].panner) return;
    const src = actx.createMediaStreamSource(e.streams[0]);
    const panner = new PannerNode(actx, { panningModel: 'HRTF', distanceModel: 'inverse', refDistance: 2, maxDistance: voiceRangeMeters, rolloffFactor: 1 });
    src.connect(panner).connect(masterGain);
    const a = new Audio(); a.muted = true; a.srcObject = e.streams[0];
    peers[id] = Object.assign(peers[id] || { x:0,y:0,z:0,yaw:0 }, { panner, audio: a });
    placePanner(peers[id]); log('ouvindo peer ' + id.slice(0,8));
  };
  pc.onicecandidate = e => { if (e.candidate) ws.send(JSON.stringify({ event: 'candidate', data: JSON.stringify(e.candidate) })); };

  ws.onopen = async () => {
    ws.send(JSON.stringify({ event: 'auth', data: s.password }));
    try {
      const constraints = { audio: cfg.micDeviceId ? { deviceId: { exact: cfg.micDeviceId } } : true };
      const mic = await navigator.mediaDevices.getUserMedia(constraints);
      mic.getTracks().forEach(t => pc.addTrack(t, mic));
      populateDevices();
      log('mic ok');
    } catch (err) { log('ERRO microfone: ' + err); }
    setConn(true, 'conectado');
    posTimer = setInterval(() => {
      if (ws && ws.readyState === 1)
        ws.send(JSON.stringify({ event: 'pos', data: `${me.x.toFixed(1)},${me.y.toFixed(1)},${me.z.toFixed(1)},${me.yaw.toFixed(1)}` }));
    }, 100);
  };

  ws.onmessage = async ev => {
    const m = JSON.parse(ev.data);
    if (m.event === 'error')      { log('ERRO: ' + m.data); return; }
    if (m.event === 'hello')      { gotHello = true; hideFallback(); log('conectado (id ' + m.data.slice(0,8) + ')'); return; }
    if (m.event === 'serverinfo') { // "no compose e o padrao": nome + alcance do servidor
      try { const si = JSON.parse(m.data); $('srvName').textContent = si.name ? '· ' + si.name : '';
            const r = parseFloat(si.range); if (r) applyRange(r); } catch (_) {} return;
    }
    if (m.event === 'offer') {
      await pc.setRemoteDescription(JSON.parse(m.data));
      const ans = await pc.createAnswer(); await pc.setLocalDescription(ans);
      ws.send(JSON.stringify({ event: 'answer', data: JSON.stringify(ans) }));
    }
    if (m.event === 'candidate') { try { await pc.addIceCandidate(JSON.parse(m.data)); } catch (_) {} }
    if (m.event === 'pos') {
      const [x, y, z, yaw] = m.data.split(',').map(Number);
      placePanner(peers[m.id] = Object.assign(peers[m.id] || {}, { x, y, z, yaw }));
    }
  };

  ws.onerror = () => { log('erro de websocket'); };
  ws.onclose = () => {
    const failed = !gotHello;
    stop();
    if (failed) { const s2 = selectedServer(); showFallback(s2 ? s2.name : ''); log('falhou — não achei o servidor'); }
  };
}

function stop() {
  if (posTimer) { clearInterval(posTimer); posTimer = null; }
  if (pc) { try { pc.close(); } catch (_) {} pc = null; }
  if (ws) { try { ws.close(); } catch (_) {} ws = null; }
  for (const id in peers) { try { peers[id].panner && peers[id].panner.disconnect(); } catch (_) {} delete peers[id]; }
  setConn(false, 'desconectado');
}

// ----- config / servidores -----
function applyVolume(v) { masterVolume = v; if (masterGain) masterGain.gain.value = v; }
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
async function populateDevices() {
  let devices = []; try { devices = await navigator.mediaDevices.enumerateDevices(); } catch (_) {}
  const mic = $('cfgMic'), out = $('cfgOutput'); mic.innerHTML = ''; out.innerHTML = '';
  const addOpt = (sel, id, label, selId) => { const o = document.createElement('option'); o.value = id;
    o.textContent = label || (id ? id.slice(0,10) : 'Padrão'); if (id === selId) o.selected = true; sel.appendChild(o); };
  addOpt(mic, '', 'Padrão', cfg.micDeviceId); addOpt(out, '', 'Padrão', cfg.outputDeviceId);
  let hasLabels = false;
  devices.forEach(d => { if (d.label) hasLabels = true;
    if (d.kind === 'audioinput')  addOpt(mic, d.deviceId, d.label, cfg.micDeviceId);
    if (d.kind === 'audiooutput') addOpt(out, d.deviceId, d.label, cfg.outputDeviceId); });
  $('cfgDevHint').textContent = hasLabels ? '' : '"Atualizar dispositivos" pra ver os nomes';
}
async function refreshDevices() {
  try { const t = await navigator.mediaDevices.getUserMedia({ audio: true }); t.getTracks().forEach(x => x.stop()); }
  catch (e) { log('permissão de mic negada: ' + e); }
  await populateDevices();
}

// pega TUDO da UI -> objeto cfg
function gatherConfig() {
  if (cfg.servers.length === 0) cfg.servers.push(readServerFields());
  else cfg.servers[cfg.selected] = readServerFields();
  cfg.volume = parseFloat($('cfgVolume').value);
  cfg.micDeviceId = $('cfgMic').value || '';
  cfg.outputDeviceId = $('cfgOutput').value || '';
  cfg.autoConnect = $('cfgAuto').checked;
  cfg.autoDetect = $('cfgAutoDetect').checked;
  cfg.autoPort = parseInt($('cfgAutoPort').value) || 8765;
  cfg.autoPassword = $('cfgAutoPassword').value;
  return cfg;
}

// ----- UI wiring -----
$('cfgVolume').addEventListener('input', e => { const v = parseFloat(e.target.value); $('cfgVolumeVal').textContent = v.toFixed(2); applyVolume(v); });
$('cfgRefreshDevices').addEventListener('click', refreshDevices);
$('cfgServerSel').addEventListener('change', e => { cfg.selected = parseInt(e.target.value) || 0; fillServerFields(selectedServer()); });
$('cfgAdd').addEventListener('click', () => { cfg.servers.push({ name:'Novo', url:'', password:'', voiceRangeMeters:50 }); cfg.selected = cfg.servers.length-1; renderServerList(); fillServerFields(selectedServer()); });
$('cfgDel').addEventListener('click', () => { if (!cfg.servers.length) return; cfg.servers.splice(cfg.selected,1); cfg.selected = 0; renderServerList(); fillServerFields(selectedServer()); });

$('cfgSave').addEventListener('click', async () => {
  gatherConfig(); applyVolume(cfg.volume); renderServerList();
  if (cfg.outputDeviceId && actx && actx.setSinkId) { try { await actx.setSinkId(cfg.outputDeviceId); } catch (_) {} }
  try { await window.go.main.App.SaveConfig(cfg); $('cfgSaved').textContent = 'salvo ✓'; setTimeout(() => $('cfgSaved').textContent = '', 2000); }
  catch (err) { log('erro ao salvar: ' + err); }
});

$('fbConfig').addEventListener('click', () => { hideFallback(); $('cfgSection').open = true; });
$('fbIgnore').addEventListener('click', hideFallback);

// botoes manuais: Conectar (no servidor selecionado da lista) / Desconectar
$('go').onclick = () => { hideFallback(); connectSelected(); };
$('leave').onclick = stop;

// ----- boot -----
(async () => {
  try {
    cfg = await window.go.main.App.GetConfig();
    if (!cfg.servers) cfg.servers = [];
    masterVolume = (cfg.volume == null) ? 1.0 : cfg.volume;
    $('cfgVolume').value = masterVolume; $('cfgVolumeVal').textContent = masterVolume.toFixed(2);
    $('cfgAuto').checked = cfg.autoConnect !== false;
    $('cfgAutoDetect').checked = !!cfg.autoDetect;
    $('cfgAutoPort').value = cfg.autoPort || 8765;
    $('cfgAutoPassword').value = cfg.autoPassword || '';
    renderServerList(); fillServerFields(selectedServer());
    await populateDevices();
    const s = selectedServer();
    if (s) applyRange(s.voiceRangeMeters || 50);
    log('pronto — esperando entrar no jogo');
    if (!cfg.autoDetect && (!s || !s.url || !s.password)) { $('cfgSection').open = true; log('adicione um servidor na config'); }
  } catch (err) {
    log('erro ao carregar config: ' + err); $('cfgSection').open = true;
  }
})();
