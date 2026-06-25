'use strict';
// PalProxVoice — Fase 0: recon do PalServer-Linux (UE5.1 ELF) pra achar o GUObjectArray.
// Roda via: frida -n PalServer-Linux-Shipping-Cmd -l /probe.js
// Objetivo: provar que pos+yaw+FGuid sao leriveis no binario Linux. ITERATIVO: roda,
// le os logs, refina os patterns/offsets aqui, repete.

function log(s) { console.log('[ppv] ' + s); }
function hex(p) { return p && p.toString ? p.toString() : '' + p; }

// ---------- 1) modulo principal ----------
const mods = Process.enumerateModules();
log('modulos carregados: ' + mods.length + '  (arch=' + Process.arch + ', ptr=' + Process.pointerSize + ')');
let main = mods.find(m => /PalServer|Pal-Linux|Palworld/i.test(m.name));
if (!main) main = mods.reduce((a, b) => (b.size > (a ? a.size : 0) ? b : a), null); // maior = o exe
log('main: ' + main.name + ' @ ' + hex(main.base) + ' size=0x' + main.size.toString(16) + ' path=' + main.path);

// ---------- helpers ----------
function ptrInModule(p) { try { return p.compare(main.base) >= 0 && p.compare(main.base.add(main.size)) < 0; } catch (_) { return false; } }
function ptrReadable(p) { try { const r = Process.findRangeByAddress(p); return !!(r && r.protection[0] === 'r'); } catch (_) { return false; } }
function readPtr(p) { try { return p.readPointer(); } catch (_) { return null; } }
// scan de bytes no modulo principal (string ou hex). cb(addr) retorna 'stop' p/ parar.
function scanMain(pattern, cb, max) {
  let n = 0;
  Memory.scan(main.base, main.size, pattern, {
    onMatch(addr) { if (cb(addr) === 'stop') return 'stop'; if (++n >= (max || 50)) return 'stop'; },
    onComplete() {}
  });
  return n;
}
function strToPattern(s) { return s.split('').map(c => c.charCodeAt(0).toString(16).padStart(2, '0')).join(' '); }

// ---------- 2) ancora: "/Script/CoreUObject" (existe em todo binario UE; mora perto do FNamePool) ----------
log('--- ancorando FName "/Script/CoreUObject" ---');
let anchorAddr = null;
scanMain(strToPattern('/Script/CoreUObject'), addr => { anchorAddr = addr; log('  ancora @ ' + hex(addr)); return 'stop'; }, 3);
if (!anchorAddr) log('  (ancora nao achada no main — talvez nome de modulo diferente; ver lista acima)');

// ---------- 3) caca ao GUObjectArray (FUObjectArray / FChunkedFixedUObjectArray, UE5.1) ----------
// Layout alvo (x86_64): { FUObjectItem** Objects; ...; int32 MaxElements; int32 NumElements; ... }
// FUObjectItem = { UObject* Object; int32 Flags; int32 ClusterRootIndex; int32 SerialNumber; } (>=24B na UE5)
// Heuristica: varre as ranges r/w (data/bss) por um ponteiro Objects** cujo 1o chunk[0].Object
// seja um UObject plausivel (ponteiro alinhado, vtable apontando pro modulo).
function looksLikeUObject(p) {
  if (!p || p.isNull() || !ptrReadable(p)) return false;
  const vt = readPtr(p);              // UObject* -> vtable*
  return vt && ptrInModule(vt);       // vtable mora no .text/.rodata do exe
}
function validateChunkedArray(objectsPP) {
  // objectsPP = FUObjectItem** (array de chunks). chunk0 = primeiro bloco de FUObjectItem.
  const chunk0 = readPtr(objectsPP);
  if (!chunk0 || !ptrReadable(chunk0)) return false;
  let valid = 0;
  for (let i = 0; i < 8; i++) {        // checa os primeiros 8 UObjects do chunk0
    const item = chunk0.add(i * 24);   // FUObjectItem (24B). ajuste p/ 16 se a val. falhar
    if (looksLikeUObject(readPtr(item))) valid++;
  }
  return valid >= 5;                   // maioria dos primeiros slots = UObjects -> e' ele
}
log('--- cacando GUObjectArray (heuristica nas ranges r/w) ---');
let found = 0;
const ranges = Process.enumerateRanges('rw-').filter(r => r.size < 0x4000000); // pula mapas gigantes
for (const r of ranges) {
  if (found >= 3) break;
  for (let off = 0; off + 8 <= r.size; off += 8) {
    const slot = r.base.add(off);
    const cand = readPtr(slot);        // candidato a Objects** (ponteiro p/ array de chunks)
    if (!cand || cand.isNull() || !ptrReadable(cand)) continue;
    if (validateChunkedArray(cand)) {
      const maxEl = slot.add(8 * 1).readU32 ? slot : null; // NumElements/MaxElements ficam logo apos (ajustar offset)
      log('  CANDIDATO GUObjectArray.ObjObjects.Objects @ ' + hex(slot) + '  (chunks @ ' + hex(cand) + ')');
      try {
        // tenta ler MaxElements/NumElements nos int32 que costumam vir apos o ponteiro
        for (let k = 8; k <= 24; k += 4) log('    +0x' + k.toString(16) + ' int32=' + slot.add(k).readU32());
      } catch (_) {}
      found++;
      if (found >= 3) break;
    }
  }
}
if (!found) log('  nenhum candidato — refinar: tamanho do FUObjectItem (16 vs 24), ranges, ou ancorar via funcao.');

log('--- recon Fase 0 terminado. Proximo: dos candidatos, walk PlayerController->Pawn->RelativeLocation/Rotation + PlayerState.IndividualHandleId. ---');
