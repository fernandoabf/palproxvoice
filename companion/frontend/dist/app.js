// PalProxVoice Companion — frontend vanilla (Wails v2)
// Posicao vem do evento Wails "pos"; "posLost" = saiu do mundo -> desconecta.
// Config via window.go.main.App.GetConfig/SaveConfig (inclui mic + saida de audio).

// ----- world: posicao em CENTIMETROS (mesmo formato do mod M1: x,y,z,yaw) -----
const me = { x: 0, y: 0, z: 0, yaw: 0 };  // yaw em graus
const CM = 100;          // cm -> m na hora de espacializar
const peers = {};        // id -> { x,y,z,yaw, panner, audio }
let posFromGame = false; // ja recebemos posicao do jogo?
let inGame = false;      // true entre o 1o "pos" e o "posLost"

// runtime config (defaults; sobrescritos por GetConfig no load)
let voiceRangeMeters = 50;
let masterVolume = 1.0;

const logEl = document.getElementById('log');
const log = m => { logEl.textContent += m + "\n"; logEl.scrollTop = logEl.scrollHeight; };

// yaw(graus) -> vetor "pra frente" no plano X-Z do Web Audio.
const fwd = yaw => { const r = yaw * Math.PI / 180; return [Math.cos(r), 0, Math.sin(r)]; };
(() => { const [fx,,fz] = fwd(0); console.assert(Math.abs(fx-1)<1e-9 && Math.abs(fz)<1e-9, "fwd(0) deve ser +X"); })();

let actx, listener, masterGain;
function audioPos(p) { return [p.x/CM, p.z/CM, p.y/CM]; }  // game(x,y,z) -> audio(x=X, y=altura=Z, z=Y)

function placeListener() {
  if (!listener) return;
  const [ax, ay, az] = audioPos(me);
  const [fx, fy, fz] = fwd(me.yaw);
  if (listener.positionX) {
    listener.positionX.value = ax; listener.positionY.value = ay; listener.positionZ.value = az;
    listener.forwardX.value = fx; listener.forwardY.value = fy; listener.forwardZ.value = fz;
    listener.upX.value = 0; listener.upY.value = 1; listener.upZ.value = 0;
  } else {
    listener.setPosition(ax, ay, az);
    listener.setOrientation(fx, fy, fz, 0, 1, 0);
  }
}
function placePanner(p) {
  if (!p.panner) return;
  const [ax, ay, az] = audioPos(p);
  if (p.panner.positionX) { p.panner.positionX.value = ax; p.panner.positionY.value = ay; p.panner.positionZ.value = az; }
  else p.panner.setPosition(ax, ay, az);
}

// ----- desenho (top-down) centrado em voce -----
const cv = document.getElementById('stage'), ctx = cv.getContext('2d');
function draw() {
  ctx.clearRect(0, 0, cv.width, cv.height);
  const cx = cv.width/2, cy = cv.height/2, S = 0.03; // px/cm
  const dot = (p, color, label) => {
    const X = cx + (p.x - me.x)*S, Y = cy + (p.y - me.y)*S;
    ctx.fillStyle = color; ctx.beginPath(); ctx.arc(X, Y, 7, 0, 7); ctx.fill();
    const [fx,,fz] = fwd(p.yaw);
    ctx.strokeStyle = color; ctx.beginPath(); ctx.moveTo(X, Y); ctx.lineTo(X+fx*16, Y+fz*16); ctx.stroke();
    ctx.fillStyle = '#cdd6e0'; ctx.font = '11px monospace'; ctx.fillText(label, X+10, Y-8);
  };
  for (const id in peers) if (peers[id].panner) dot(peers[id], '#e06363', id.slice(0,4));
  dot(me, '#5aa9e6', 'voce');
  document.getElementById('status').textContent =
    `voce: ${me.x.toFixed(0)}, ${me.y.toFixed(0)}, ${me.z.toFixed(0)}  yaw ${me.yaw.toFixed(0)}` +
    (posFromGame ? '   [posicao: JOGO]' : '   [posicao: aguardando jogo]');
  requestAnimationFrame(draw);
}
requestAnimationFrame(draw);

// ----- eventos do backend Wails: "pos" (atualiza) e "posLost" (saiu) -----
function onPos(data) {
  if (!data) return;
  const t = String(data).trim();
  if (!t) return;
  const [x, y, z, yaw] = t.split(',').map(Number);
  if ([x, y, z, yaw].some(Number.isNaN)) return;
  me.x = x; me.y = y; me.z = z; me.yaw = yaw;
  posFromGame = true;
  placeListener();
  if (!inGame) { inGame = true; autoConnectIfConfigured(); } // entrou no jogo -> conecta
}
if (window.runtime && window.runtime.EventsOn) {
  window.runtime.EventsOn('pos', onPos);
  window.runtime.EventsOn('posLost', () => {           // posicao parou -> saiu do servidor
    inGame = false; posFromGame = false;
    if (ws) { log('saiu do servidor — voz desconectada'); stop(); }
  });
} else {
  log('aviso: window.runtime indisponivel (rodando fora do Wails?)');
}

// ----- WebSocket / WebRTC -----
let ws = null;
let pc = null;
let posTimer = null;

function setConn(state, label) {
  const el = document.getElementById('conn');
  el.textContent = label;
  el.className = 'badge ' + (state ? 'on' : 'off');
  document.getElementById('go').disabled = state;
  document.getElementById('leave').disabled = !state;
}

function wsURLFrom(serverURL) {
  let u = (serverURL || '').trim();
  if (!u) return '';
  if (u.startsWith('https://'))      u = 'wss://' + u.slice('https://'.length);
  else if (u.startsWith('http://'))  u = 'ws://'  + u.slice('http://'.length);
  else if (!u.startsWith('ws://') && !u.startsWith('wss://')) u = 'ws://' + u;
  u = u.replace(/\/+$/, '');
  if (!u.endsWith('/ws')) u += '/ws';
  return u;
}

function autoConnectIfConfigured() {
  if (ws) return;
  const cfg = readCfgForm();
  if (cfg.serverUrl && cfg.password) { log('entrou no jogo — conectando voz'); start(cfg.serverUrl, cfg.password); }
}

async function start(serverURL, password) {
  if (ws) { log('ja conectado/conectando'); return; }
  const url = wsURLFrom(serverURL);
  if (!url || !password) { log('preencha Server URL e Senha'); return; }
  const cfg = readCfgForm();

  setConn(false, 'conectando...');

  if (!actx) {
    actx = new (window.AudioContext || window.webkitAudioContext)();
    listener = actx.listener;
    masterGain = actx.createGain();
    masterGain.gain.value = masterVolume;
    masterGain.connect(actx.destination);
  }
  if (actx.state === 'suspended') { try { await actx.resume(); } catch (_) {} }
  // saida de audio escolhida (Chrome/WebView2 recente)
  if (cfg.outputDeviceId && actx.setSinkId) { try { await actx.setSinkId(cfg.outputDeviceId); } catch (e) { log('saida nao aplicada: ' + e); } }
  placeListener();

  ws = new WebSocket(url);
  pc = new RTCPeerConnection({ iceServers: [{ urls: 'stun:stun.l.google.com:19302' }] });

  pc.ontrack = e => {
    const id = e.streams[0].id;
    if (peers[id] && peers[id].panner) return;
    const src = actx.createMediaStreamSource(e.streams[0]);
    const panner = new PannerNode(actx, {
      panningModel: 'HRTF', distanceModel: 'inverse',
      refDistance: 2, maxDistance: voiceRangeMeters, rolloffFactor: 1,
    });
    src.connect(panner).connect(masterGain);
    const a = new Audio(); a.muted = true; a.srcObject = e.streams[0];
    peers[id] = Object.assign(peers[id] || { x:0,y:0,z:0,yaw:0 }, { panner, audio: a });
    placePanner(peers[id]);
    log('ouvindo peer ' + id.slice(0,8));
  };

  pc.onicecandidate = e => {
    if (e.candidate) ws.send(JSON.stringify({ event: 'candidate', data: JSON.stringify(e.candidate) }));
  };

  ws.onopen = async () => {
    ws.send(JSON.stringify({ event: 'auth', data: password }));
    try {
      // microfone escolhido (ou padrao)
      const constraints = { audio: cfg.micDeviceId ? { deviceId: { exact: cfg.micDeviceId } } : true };
      const mic = await navigator.mediaDevices.getUserMedia(constraints);
      mic.getTracks().forEach(t => pc.addTrack(t, mic));
      populateDevices(cfg.micDeviceId, cfg.outputDeviceId); // permissao concedida -> labels aparecem
      log('mic ok, conectando...');
    } catch (err) {
      log('ERRO ao abrir microfone: ' + err);
    }
    setConn(true, 'conectado');
    posTimer = setInterval(() => {
      if (ws && ws.readyState === 1)
        ws.send(JSON.stringify({ event: 'pos', data: `${me.x.toFixed(1)},${me.y.toFixed(1)},${me.z.toFixed(1)},${me.yaw.toFixed(1)}` }));
    }, 100);
  };

  ws.onmessage = async ev => {
    const m = JSON.parse(ev.data);
    if (m.event === 'error')  { log('ERRO: ' + m.data); return; }
    if (m.event === 'hello')  { log('conectado. meu id = ' + m.data.slice(0,8)); return; }
    if (m.event === 'offer')  {
      await pc.setRemoteDescription(JSON.parse(m.data));
      const ans = await pc.createAnswer();
      await pc.setLocalDescription(ans);
      ws.send(JSON.stringify({ event: 'answer', data: JSON.stringify(ans) }));
    }
    if (m.event === 'candidate') { try { await pc.addIceCandidate(JSON.parse(m.data)); } catch (_) {} }
    if (m.event === 'pos') {
      const [x, y, z, yaw] = m.data.split(',').map(Number);
      const p = peers[m.id] = Object.assign(peers[m.id] || {}, { x, y, z, yaw });
      placePanner(p);
    }
  };

  ws.onclose = () => { log('ws fechou'); stop(); };
  ws.onerror = () => { log('erro de websocket'); };
}

function stop() {
  if (posTimer) { clearInterval(posTimer); posTimer = null; }
  if (pc) { try { pc.close(); } catch (_) {} pc = null; }
  if (ws) { try { ws.close(); } catch (_) {} ws = null; }
  for (const id in peers) {
    try { peers[id].panner && peers[id].panner.disconnect(); } catch (_) {}
    delete peers[id];
  }
  setConn(false, 'desconectado');
}

// ----- config (Wails bindings) -----
function applyVolume(v) {
  masterVolume = v;
  if (masterGain) masterGain.gain.value = v;
}
function applyRange(m) {
  voiceRangeMeters = m;
  for (const id in peers) { const p = peers[id]; if (p.panner) p.panner.maxDistance = m; }
}

// ----- dispositivos de audio (mic + saida) -----
async function populateDevices(selMic, selOut) {
  let devices = [];
  try { devices = await navigator.mediaDevices.enumerateDevices(); } catch (_) {}
  const mic = document.getElementById('cfgMic');
  const out = document.getElementById('cfgOutput');
  mic.innerHTML = ''; out.innerHTML = '';
  const addOpt = (sel, id, label, selId) => {
    const o = document.createElement('option');
    o.value = id; o.textContent = label || (id ? id.slice(0, 10) : 'Padrão');
    if (id === selId) o.selected = true;
    sel.appendChild(o);
  };
  addOpt(mic, '', 'Padrão', selMic);
  addOpt(out, '', 'Padrão', selOut);
  let hasLabels = false;
  devices.forEach(d => {
    if (d.label) hasLabels = true;
    if (d.kind === 'audioinput')  addOpt(mic, d.deviceId, d.label, selMic);
    if (d.kind === 'audiooutput') addOpt(out, d.deviceId, d.label, selOut);
  });
  document.getElementById('cfgDevHint').textContent = hasLabels ? '' : '"Atualizar dispositivos" pra ver os nomes';
}
async function refreshDevices() {
  try { const t = await navigator.mediaDevices.getUserMedia({ audio: true }); t.getTracks().forEach(x => x.stop()); }
  catch (e) { log('permissão de mic negada: ' + e); }
  const cfg = readCfgForm();
  await populateDevices(cfg.micDeviceId, cfg.outputDeviceId);
}

function readCfgForm() {
  return {
    serverUrl: document.getElementById('cfgServer').value.trim(),
    password:  document.getElementById('cfgPassword').value,
    voiceRangeMeters: parseFloat(document.getElementById('cfgRange').value) || 50,
    volume:    parseFloat(document.getElementById('cfgVolume').value),
    micDeviceId:    document.getElementById('cfgMic').value || '',
    outputDeviceId: document.getElementById('cfgOutput').value || '',
  };
}

function fillCfgForm(cfg) {
  document.getElementById('cfgServer').value   = cfg.serverUrl || '';
  document.getElementById('cfgPassword').value = cfg.password || '';
  document.getElementById('cfgRange').value    = cfg.voiceRangeMeters || 50;
  const vol = (cfg.volume == null) ? 1.0 : cfg.volume;
  document.getElementById('cfgVolume').value   = vol;
  document.getElementById('cfgVolumeVal').textContent = vol.toFixed(2);
  // mic/saida: populados por populateDevices (restaura a selecao salva)
}

// ----- UI wiring -----
document.getElementById('cfgVolume').addEventListener('input', e => {
  const v = parseFloat(e.target.value);
  document.getElementById('cfgVolumeVal').textContent = v.toFixed(2);
  applyVolume(v);
});
document.getElementById('cfgRefreshDevices').addEventListener('click', refreshDevices);

document.getElementById('cfgSave').addEventListener('click', async () => {
  const cfg = readCfgForm();
  applyRange(cfg.voiceRangeMeters);
  applyVolume(cfg.volume);
  if (cfg.outputDeviceId && actx && actx.setSinkId) { try { await actx.setSinkId(cfg.outputDeviceId); } catch (_) {} }
  try {
    await window.go.main.App.SaveConfig(cfg);
    const el = document.getElementById('cfgSaved');
    el.textContent = 'salvo ✓';
    setTimeout(() => { el.textContent = ''; }, 2000);
  } catch (err) {
    log('erro ao salvar config: ' + err);
  }
});

document.getElementById('go').onclick = () => {
  const cfg = readCfgForm();
  applyRange(cfg.voiceRangeMeters);
  applyVolume(cfg.volume);
  start(cfg.serverUrl, cfg.password);
};
document.getElementById('leave').onclick = stop;

// ----- boot: carrega config + lista dispositivos; conecta quando entrar no jogo -----
(async () => {
  try {
    const cfg = await window.go.main.App.GetConfig();
    fillCfgForm(cfg);
    applyRange(cfg.voiceRangeMeters || 50);
    masterVolume = (cfg.volume == null) ? 1.0 : cfg.volume;
    await populateDevices(cfg.micDeviceId, cfg.outputDeviceId);
    log('config carregada — esperando entrar no jogo');
    if (!cfg.serverUrl || !cfg.password) document.getElementById('cfgSection').open = true;
  } catch (err) {
    log('erro ao carregar config (fora do Wails?): ' + err);
    document.getElementById('cfgSection').open = true;
  }
})();
