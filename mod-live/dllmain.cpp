// PalProxVoiceLive — mod C++ do UE4SS.
//
// Escreve o IP da sessao ATUAL em C:\Users\Public\palproxvoice_server.txt,
// chamando UNetConnection::LowLevelGetRemoteAddress(true). Essa funcao NAO e
// UFUNCTION (por isso o Lua nao alcanca — testamos), mas e uma virtual normal:
// em C++, com o header da classe, o compilador resolve a vtable sozinho ->
// robusto a updates que NAO mudem a versao do engine.
//
// O companion le esse arquivo em GameServerIPLive() (serverdetect.go).
//
// NAO FOI COMPILADO/TESTADO aqui (precisa Windows + RE-UE4SS). Ver README.md.
// Dois pontos que podem precisar de ajuste por versao do UE4SS estao marcados [AJUSTE].

#include <fstream>
#include <string>

#include <Mod/CppUserModBase.hpp>
#include <Unreal/UObjectGlobals.hpp>
#include <Unreal/UObject.hpp>
#include <Unreal/FString.hpp>

// Header da classe gerado pelo dump CXX do UE4SS (define LowLevelGetRemoteAddress).
// Gere uma vez (ver README) e ajuste o include pro caminho real. [AJUSTE]
// Se nao quiser gerar, da pra declarar so a virtual no indice certo da vtable.
#include <SDK/Engine/Classes/Engine/NetConnection.hpp>

using namespace RC;
using namespace RC::Unreal;

static const wchar_t* kOutFile = L"C:/Users/Public/palproxvoice_server.txt";

class PalProxVoiceLive : public CppUserModBase {
public:
    PalProxVoiceLive() {
        ModName = STR("PalProxVoiceLive");
        ModVersion = STR("1.0");
        ModDescription = STR("Escreve o IP do servidor atual (LowLevelGetRemoteAddress).");
        ModAuthors = STR("PalProxVoice");
    }

    int frame = 0;
    std::string last;

    // chamado todo frame no game thread; throttle pra ~1x/5s (assumindo ~60fps).
    auto on_update() -> void override {
        if (++frame % 300 != 0) return;
        write_server_ip();
    }

    void write_server_ip() {
        // PlayerController existe so dentro de um mundo; fora dele, nada a fazer.
        UObject* pc = UObjectGlobals::FindFirstOf(STR("PlayerController"));
        if (!pc) return;

        // NetConnection e UPROPERTY refletida (confirmamos no probe). So no CLIENTE
        // ela e != null; no host/local fica null -> sem IP remoto, ok.
        UObject** ncPtr = pc->GetValuePtrByPropertyNameInChain<UObject*>(STR("NetConnection")); // [AJUSTE] assinatura varia por versao
        if (!ncPtr || !*ncPtr) return;

        auto* nc = static_cast<UNetConnection*>(*ncPtr);
        FString addr = nc->LowLevelGetRemoteAddress(true); // "ip:porta"
        const TCHAR* chars = addr.GetCharArray();
        if (!chars) return;

        std::wstring ws(chars);
        std::string s(ws.begin(), ws.end()); // IP/porta sao ASCII
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
