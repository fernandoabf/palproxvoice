// PalProxVoiceLive — mod C++ do RE-UE4SS (alvo: UE 5.1 / Palworld).
//
// Escreve o IP da sessao ATUAL em C:\Users\Public\palproxvoice_server.txt,
// chamando UNetConnection::LowLevelGetRemoteAddress(true). Essa funcao NAO e
// UFUNCTION (o Lua nao alcanca — testamos), mas e uma virtual normal: em C++,
// com a declaracao da classe (vtable na ordem certa), o compilador resolve a
// chamada sozinho.
//
// O companion le esse arquivo em GameServerIPLive() (serverdetect.go).
// Contrato preservado: grava "ip:porta" (ASCII) SOMENTE quando o valor muda.
//
// NAO FOI COMPILADO/TESTADO aqui (precisa Windows + RE-UE4SS + MSVC). Ver README.md.

#include <fstream>
#include <string>

#include <Mod/CppUserModBase.hpp>
#include <DynamicOutput/DynamicOutput.hpp>
#include <Unreal/UObjectGlobals.hpp>
#include <Unreal/UObject.hpp>
#include <Unreal/FString.hpp>

// ---------------------------------------------------------------------------
// [AJUSTE 1] — Declaracao de UNetConnection::LowLevelGetRemoteAddress.
//
// O RE-UE4SS NAO traz um header UNetConnection com esse metodo (LowLevelGetRemoteAddress
// nao e UFUNCTION e nao esta no SDK first-party do UE4SS — confirmado). Voce tem
// duas opcoes; escolha UMA e deixe a outra comentada:
//
//   OPCAO A (recomendada) — usar o dump CXX do proprio jogo:
//     No jogo, com UE4SS injetado, gere os CXX headers (CTRL+H por padrao). Eles
//     saem em <Palworld>/Binaries/.../UE4SS_CXX_Gen/cxx_headers/ (ou Output do UE4SS).
//     Aponte o #include pro NetConnection.hpp gerado e adicione a pasta cxx_headers
//     ao include path (ver CMakeLists/xmake). O dump tem padding impreciso, mas a
//     ORDEM das virtuais (vtable) vem do binario real do jogo -> chamar a virtual
//     funciona. Descomente:
//
//   // #include "Engine/Classes/Engine/NetConnection.hpp"   // do dump CXX
//
//   OPCAO B (sem SDK) — shim minimo declarado a mao (abaixo). So precisa que a
//     ORDEM das virtuais ate LowLevelGetRemoteAddress bata com a do UE 5.1.
//     Verifique o indice com Dumpers/live editor antes de confiar em producao.
// ---------------------------------------------------------------------------

using namespace RC;
using namespace RC::Unreal;

#ifndef PPV_USE_CXX_SDK
// OPCAO B — shim minimo. NAO herda de UObject do UE4SS de proposito: so precisamos
// do ponteiro de objeto (que ja temos via reflexao) e da vtable. A ordem das
// virtuais de UNetConnection (UObject -> UPlayer -> UNetConnection) em UE 5.1
// coloca LowLevelGetRemoteAddress logo apos LowLevelDescribe/LowLevelSend. Como a
// ordem exata e fragil, deixamos a chamada por slot de vtable explicito e isolado
// num helper, pra ser facil de re-verificar/ajustar por update do jogo.
//
// IMPORTANTE: confirme PPV_VTBL_LOWLEVELGETREMOTEADDRESS no live editor / dump CXX
// do build atual do Palworld antes de usar a OPCAO B em producao.
namespace ppv {
    // FString do UE4SS ja esta incluido; reaproveitamos o tipo real.
    using FStringT = RC::Unreal::FString;

    // Indice da virtual LowLevelGetRemoteAddress na vtable de UNetConnection.
    // [AJUSTE] verificar por versao do engine. Valor abaixo e PLACEHOLDER.
    static constexpr int PPV_VTBL_LOWLEVELGETREMOTEADDRESS = -1;

    inline bool call_low_level_get_remote_address(void* netconn, bool bAppendPort, FStringT& out) {
        if (!netconn || PPV_VTBL_LOWLEVELGETREMOTEADDRESS < 0) return false;
        // vtable* esta no offset 0 do objeto.
        using Fn = FStringT (*)(void* /*this*/, bool /*bAppendPort*/);
        void** vtbl = *reinterpret_cast<void***>(netconn);
        auto fn = reinterpret_cast<Fn>(vtbl[PPV_VTBL_LOWLEVELGETREMOTEADDRESS]);
        out = fn(netconn, bAppendPort);
        return true;
    }
}
#endif

static const wchar_t* kOutFile = L"C:/Users/Public/palproxvoice_server.txt";

class PalProxVoiceLive : public CppUserModBase {
public:
    PalProxVoiceLive() {
        ModName = STR("PalProxVoiceLive");
        ModVersion = STR("1.0");
        ModDescription = STR("Escreve o IP do servidor atual (LowLevelGetRemoteAddress).");
        ModAuthors = STR("PalProxVoice");
        // ModIntendedSDKVersion = STR("..."); // so se quiser fixar versao do UE4SS
    }

    ~PalProxVoiceLive() override {}

    int frame = 0;
    std::string last;
    bool ready = false; // so toca em UObjects depois de on_unreal_init

    // on_unreal_init e o ponto mais cedo seguro pra usar UObjectGlobals.
    auto on_unreal_init() -> void override {
        ready = true;
        Output::send<LogLevel::Verbose>(STR("[PalProxVoiceLive] unreal init ok\n"));
    }

    // on_update roda todo frame no game thread; throttle pra ~1x/5s (~60fps).
    auto on_update() -> void override {
        if (!ready) return;
        if (++frame % 300 != 0) return;
        write_server_ip();
    }

    void write_server_ip() {
        // [AJUSTE 2] — pegar o PlayerController. FindFirstOf acha a primeira
        // instancia NAO-CDO da classe pelo nome curto. Assinatura atual:
        //   UObject* FindFirstOf(StringViewType class_name)  (RC::Unreal::UObjectGlobals)
        UObject* pc = UObjectGlobals::FindFirstOf(STR("PlayerController"));
        if (!pc) return;

        // NetConnection e UPROPERTY refletida (confirmado no probe). So no CLIENTE
        // ela e != null; no host/local fica null -> sem IP remoto, ok.
        //
        // GetValuePtrByPropertyNameInChain<T>(name) retorna T* (ponteiro pro valor
        // dentro do objeto). A property guarda um UObject*, entao <UObject*> retorna
        // UObject** (ponteiro pro slot que contem o ponteiro). API atual do UE4SS.
        UObject** ncPtr = pc->GetValuePtrByPropertyNameInChain<UObject*>(STR("NetConnection"));
        if (!ncPtr || !*ncPtr) return;

        UObject* nc = *ncPtr;

#ifdef PPV_USE_CXX_SDK
        // OPCAO A: cast pro tipo do dump CXX e chama direto (compilador resolve vtable).
        auto* conn = static_cast<UNetConnection*>(nc);
        FString addr = conn->LowLevelGetRemoteAddress(true); // "ip:porta"
#else
        // OPCAO B: chamada por slot de vtable (ver helper acima).
        FString addr;
        if (!ppv::call_low_level_get_remote_address(nc, true, addr)) return;
#endif

        // FString -> std::string. GetCharArray() retorna const TCHAR* (wchar_t no Win).
        const TCHAR* chars = addr.GetCharArray();
        if (!chars) return;
        std::wstring ws(chars);
        if (ws.empty()) return;
        std::string s(ws.begin(), ws.end()); // ip:porta sao ASCII

        if (s.empty() || s == last) return;   // so grava quando muda
        last = s;

        std::ofstream f(kOutFile, std::ios::trunc);
        if (f) f << s;
    }
};

#define PALPROXVOICE_API __declspec(dllexport)
extern "C" {
PALPROXVOICE_API CppUserModBase* start_mod() { return new PalProxVoiceLive(); }
PALPROXVOICE_API void uninstall_mod(CppUserModBase* mod) { delete mod; }
}
