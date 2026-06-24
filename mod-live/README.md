# PalProxVoiceLive (mod C++ — IP do servidor em tempo real)

Detecta o IP do servidor **da sessão atual, em tempo real**, e escreve em
`C:\Users\Public\palproxvoice_server.txt` (formato `ip:porta`). O companion lê via
`GameServerIPLive()` ([../companion/serverdetect.go](../companion/serverdetect.go)) —
a fonte mais precisa da cadeia **live → save → ini**.

> ⚠️ **Ainda não compilado/testado.** O `dllmain.cpp` foi **corrigido** (os 2 `[AJUSTE]`
> resolvidos abaixo), mas a chamada da virtual ainda depende de **uma** confirmação no
> jogo (índice da vtable **ou** o header do dump CXX). Precisa de Windows + MSVC + a
> árvore do RE-UE4SS pra buildar.
>
> 💡 **Antes de montar a toolchain C++**, teste o **probe em Lua** (mais rápido) — ver
> "Alternativa em Lua" no fim. Se ele pegar o IP, talvez nem precise deste mod C++.

## Por que C++ (e não Lua puro)

O endereço do servidor mora em membros C++ **não-refletidos** (`UNetConnection::URL`,
`UWorld::URL`) → a reflexão do Lua lê `nil`. E `UNetConnection::LowLevelGetRemoteAddress`
**não é UFUNCTION** → o Lua também não a chama. Em C++, com a vtable, dá pra chamá-la.
(Os campos do `FURL` *são* refletidos; o probe Lua tenta a única brecha — ver abaixo.)

## Os 2 [AJUSTE] — resolvidos

**[AJUSTE 1] — declarar `LowLevelGetRemoteAddress`.** O RE-UE4SS **não** traz esse header
(o `#include <SDK/...NetConnection.hpp>` original não existia). Duas opções no `dllmain.cpp`,
via macro `PPV_USE_CXX_SDK`:
- **OPÇÃO A (recomendada):** gere o **dump CXX** do jogo (UE4SS, `CTRL+H`) → inclua o
  `NetConnection.hpp` gerado + defina `PPV_USE_CXX_SDK`. O padding do dump é impreciso,
  mas a **ordem das virtuais** vem do binário real → a chamada da virtual funciona.
- **OPÇÃO B (default, sem SDK):** shim que chama a virtual por **índice de slot** de vtable.
  O índice está em `-1` (placeholder **seguro**: não chama). Confirme o índice real no
  live editor / dump antes de usar — **índice errado = crash**.

**[AJUSTE 2] — pegar a property `NetConnection`.**
`GetValuePtrByPropertyNameInChain<UObject*>(STR("NetConnection"))` é a API atual
(confirmada na doc oficial do UE4SS); retorna `UObject**`. Já está correto no código.

## Build (Windows)

1. **Pré-req:** Visual Studio 2022 (MSVC, workload "Desktop C++"), CMake ≥ 3.22 (ou xmake), Git.
2. Clone o RE-UE4SS com submódulos: `git clone --recursive https://github.com/UE4SS-RE/RE-UE4SS`
3. Copie **esta pasta** pra **`RE-UE4SS/cppmods/PalProxVoiceLive/`** (em `cppmods/`, **não** `Mods/`).
4. Registre o mod e builde:
   - **CMake (padrão atual):** `add_subdirectory(PalProxVoiceLive)` em `cppmods/CMakeLists.txt` →
     `cmake -B build -G "Visual Studio 17 2022" -DCMAKE_BUILD_TYPE=Game__Shipping__Win64` →
     `cmake --build build --target PalProxVoiceLive`.
   - **xmake (legado):** `includes("PalProxVoiceLive")` em `cppmods/xmake.lua` → `xmake build PalProxVoiceLive`.
5. (Opção A do [AJUSTE 1]) gere o dump CXX no jogo (`CTRL+H`); descomente o `#include` do
   `NetConnection.hpp` + `PPV_USE_CXX_SDK` no `dllmain.cpp` e o `add_includedirs`/`add_defines` no build.
6. Renomeie a DLL gerada pra `main.dll`.

## Instalar

```
Pal/Binaries/WinGDK/Mods/PalProxVoiceLive/
  dlls/main.dll     <- a DLL buildada, renomeada
  enabled.txt       <- arquivo vazio (liga o mod)
```
Sobe o jogo, entra num servidor: `C:\Users\Public\palproxvoice_server.txt` aparece com
`ip:porta`. Confira no log do UE4SS a linha `[PalProxVoiceLive] unreal init ok`.

## Riscos
- **vtable frágil (Opção B):** o índice do slot depende da ordem das virtuais no UE 5.1;
  índice errado **crasha**. Default `-1` = não chama (seguro) — confirme antes de ligar.
- **Regen em update do Palworld:** engine bump muda vtable/layout → regerar o dump (Opção A) e recompilar.
- **Steam P2P/relay:** o endereço pode vir **hostname/vazio** em vez de IP roteável (Direct Connect dá IP).
- **Deprecação do xmake:** o RE-UE4SS migrou pra CMake; prefira o caminho CMake.

## Alternativa em Lua — testada e ❌ DESCARTADA

Tentamos ler o IP em Lua puro com o [`../mod/PalProxVoice/scripts/probe_server.lua`](../mod/PalProxVoice/scripts/probe_server.lua),
sondando `PendingNetGame.URL` e `NetConnection.URL`, em **Direct Connect** e pela **lista de
recentes**. Resultado (2026-06): `URL`/`.Host`/`.Port` voltam como **`ud<UObject>` (wrapper
morto)** e **nem `:get()` nem `:ToString()`** extraem valor — porque `UNetConnection::URL` e
`UWorld::URL` **não são UPROPERTY** (a reflexão do Lua não alcança; qualquer campo num wrapper
não-refletido encadeia outro wrapper morto).

**Conclusão: só o caminho C++ acima resolve o IP em tempo real.** Enquanto o mod C++ não é
buildado, a detecção por **save** (`GameServerIPFromSave`) já cobre o caso comum. O probe fica
no repo como registro do que foi tentado.
