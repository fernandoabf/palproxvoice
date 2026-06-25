# PalProxVoice — mod NATIVO de Linux (EXPERIMENTAL)

Extrair pos+yaw+FGuid de TODOS os players direto do **PalServer NATIVO de Linux**
(`PalServer-Linux-Shipping`, ELF, depot Linux) **sem UE4SS, sem Proton, sem Wine** —
escrevendo o **mesmo feed** que o voz já consome do V2 (`fguid,x,y,z,yaw,nome`).

> ⚠️ EXPERIMENTAL. O Proton+UE4SS (`deploy/v2-experimental/linux`) é a produção que
> JÁ funciona. Isto é o caminho puro-Linux — mais frágil (offsets quebram a cada patch
> e você mantém sozinho), mas elimina a stack Wine.

## Por que leitor de memória (e não Blueprint/REST/sniff)
Pesquisa 2026-06 (4 ângulos). Único caminho que entrega X,Y,**Z**,**yaw**,FGuid a 5-20Hz:
- **Leitor de memória externo** (`process_vm_readv` + AOB scan do `GUObjectArray`). ✅
- REST → só X/Y, sem Z/yaw, ~1-2Hz → vira fallback (= V1.5).
- Blueprint/Pak → sandbox, sem saída de dados. ❌
- Sniff de replicação → Oodle + serializer proprietário, 2-6 meses. ❌
- RCON/save → sem coords / stale. ❌

**Peças prontas pra compor** (você é o 1º a juntar, não a inventar):
[KittyMemoryEx](https://github.com/MJx0/KittyMemoryEx) (read/scan ELF Linux),
[AndUEDumper](https://github.com/Pixel-Mqx/AndUEDumper) (dumper externo UE5 x86_64),
[Dumper-7](https://github.com/Encryqed/Dumper-7) (walk do GUObjectArray p/ portar),
[cheat-engine-linux](https://github.com/cheat-engine/cheat-engine) (ptrace/proc).
Offsets do `.exe` Windows **NÃO servem** (Clang vs MSVC) → RE do zero no ELF.

## Fase 0 — VALIDAÇÃO (este dir): provar que pos+yaw+FGuid são leríveis
Anexa um **script Frida** no PalServer-Linux rodando e acha o `GUObjectArray` + `FNamePool`.

```bash
# 1) sobe o PalServer NATIVO (thijsvanloef) + toolbox Frida (mesma PID namespace)
docker compose -f linux-native/docker-compose.yml up -d
# 2) entra com um player no server (porta 8411) pra ter PlayerControllers vivos
# 3) exec no toolbox e anexa o Frida no processo do PalServer
docker compose -f linux-native/docker-compose.yml exec frida bash
#   dentro:
frida -n PalServer-Linux-Shipping-Cmd -l /probe.js   # ou: frida -p <PID> -l /probe.js
```

**O loop da Fase 0 é iterativo:** roda o `probe.js`, lê os logs (módulos, âncoras,
candidatos a GUObjectArray), refina os patterns no `probe.js`, repete. Meta: imprimir
`PlayerController -> x,y,z,yaw,FGuid` correto. Quando isso aparecer, a Fase 0 venceu.

## Fase 1 — MVP (depois): daemon co-locado
Porta o que a Fase 0 descobriu pra um daemon C++/Rust que:
- acha o PID, lê `/proc/pid/maps` (base PIE/ASLR),
- AOB scan 1x + **dump de offsets por reflection em runtime** (NÃO hardcode),
- loop 10-20Hz, `process_vm_readv` em batch → escreve o feed,
- watchdog: validação semântica falhou pós-update → re-scan + alerta.
Base reutilizável: KittyMemoryEx + walk portado do Dumper-7.

## Permissões (docker)
O leitor precisa de `ptrace`: `--cap-add=SYS_PTRACE` (+ `seccomp:unconfined` em alguns
hosts) e mesma **PID namespace** do PalServer (`pid: "service:palworld"`). Já no compose.
