# Build do PalProxVoiceLive (mod C++ do RE-UE4SS) — caminho CMAKE (padrao atual).
#
# Estrutura esperada (igual aos exemplos cppmods/EventViewerMod do RE-UE4SS):
#
#   RE-UE4SS/
#     cppmods/
#       CMakeLists.txt           -> add_subdirectory(PalProxVoiceLive)  <- ADICIONE
#       PalProxVoiceLive/        <- copie ESTA pasta pra ca
#         CMakeLists.txt         <- este arquivo
#         dllmain.cpp
#
# Configure/buildar a partir da raiz do RE-UE4SS, ex.:
#   cmake -B build -G "Visual Studio 17 2022" -DCMAKE_BUILD_TYPE=Game__Shipping__Win64
#   cmake --build build --target PalProxVoiceLive

set(TARGET PalProxVoiceLive)

add_library(${TARGET} SHARED
    dllmain.cpp
)

target_include_directories(${TARGET} PRIVATE .)

# UE4SS traz, transitivamente, os includes de Mod/, Unreal/, DynamicOutput/ e o ImGui.
target_link_libraries(${TARGET} PUBLIC UE4SS)

# [AJUSTE 1 - OPCAO A] Para usar o dump CXX do jogo (NetConnection.hpp), descomente:
# target_include_directories(${TARGET} PRIVATE "C:/caminho/para/UE4SS_CXX_Gen/cxx_headers")
# target_compile_definitions(${TARGET} PRIVATE PPV_USE_CXX_SDK)
