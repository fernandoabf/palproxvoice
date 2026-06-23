# PalProxVoiceLive (mod C++ — opcional)

Detecta o IP do servidor **da sessão atual, em tempo real**, e escreve em
`C:\Users\Public\palproxvoice_server.txt`. O companion lê via
`GameServerIPLive()` ([../companion/serverdetect.go](../companion/serverdetect.go)).

> ⚠️ **Não foi compilado/testado** (precisa Windows + MSVC + código do RE-UE4SS).
> É um scaffold com a lógica certa; os 2 pontos `[AJUSTE]` no `dllmain.cpp` podem
> precisar de acerto conforme a versão do UE4SS.

## Por que C++ (e não Lua)

`UNetConnection::LowLevelGetRemoteAddress` **não é UFUNCTION** → reflexão (Lua) não
alcança (testado). É virtual normal: em C++, com o header da classe, o compilador
resolve a vtable sozinho. Robusto a updates que **não** mudem a versão do engine
(um *engine bump* no Palworld 1.0 pode exigir regerar o header).

## Quando usar

Só se precisar trocar de servidor e detectar **na hora**, sem depender do save.
Caso contrário, o caminho do save (`GameServerIPFromSave`, já pronto e testado)
resolve o "qual servidor" sem nenhum mod extra.

## Build (Windows)

1. Clona o [RE-UE4SS](https://github.com/UE4SS-RE/RE-UE4SS) e segue o setup de build (xmake + MSVC).
2. Copia esta pasta pra `RE-UE4SS/Mods/PalProxVoiceLive/`.
3. Gera os headers do SDK (uma vez): no jogo com UE4SS, dump dos CXX headers
   (GUI do UE4SS) → ajusta o `#include` do `NetConnection.hpp` no `dllmain.cpp`.
4. `xmake build PalProxVoiceLive` → sai `PalProxVoiceLive.dll`.

## Instalar

```
Pal/Binaries/WinGDK/Mods/PalProxVoiceLive/
  dlls/main.dll        <- a DLL buildada, renomeada pra main.dll
  enabled.txt          <- arquivo vazio (liga o mod)
```

Sobe o jogo, entra no servidor: o arquivo
`C:\Users\Public\palproxvoice_server.txt` aparece com `ip:porta`.
