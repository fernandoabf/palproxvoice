-- Build do PalProxVoiceLive (mod C++ do RE-UE4SS) — caminho XMAKE (legado).
--
-- O RE-UE4SS atual usa CMake como build padrao (xmake esta marcado pra deprecar).
-- Se preferir CMake, use o CMakeLists.txt desta pasta (ver README). Este xmake.lua
-- e o equivalente pro fluxo antigo: o root xmake do RE-UE4SS faz includes("cppmods"),
-- e o cppmods/xmake.lua faz includes("<SeuMod>"). Ou seja:
--
--   RE-UE4SS/
--     xmake.lua            -> includes("cppmods")
--     cppmods/
--       xmake.lua          -> includes("PalProxVoiceLive")   <- ADICIONE esta linha
--       PalProxVoiceLive/  <- copie ESTA pasta pra ca
--         xmake.lua        <- este arquivo
--         dllmain.cpp
--
-- Build a partir da raiz do RE-UE4SS: `xmake build PalProxVoiceLive`.

target("PalProxVoiceLive")
    set_kind("shared")
    set_languages("cxxlatest") -- UE4SS exige C++ moderno (cxxlatest = /std:c++latest no MSVC)
    add_files("dllmain.cpp")

    -- Depende do target UE4SS (traz includes do Mod/, Unreal/, DynamicOutput/ e a vtable do CppUserModBase).
    -- imgui/ImGui e arrastado transitivamente pelo dep UE4SS; NAO adicione add_packages("imgui") a mao
    -- (o setup atual nao expoe esse package nesse nome e quebra o build).
    add_deps("UE4SS")

    -- [AJUSTE 1 - OPCAO A] Se for usar o dump CXX do jogo (NetConnection.hpp), aponte aqui
    -- pro caminho dos cxx_headers gerados e defina a macro que liga a OPCAO A no dllmain:
    -- add_includedirs("C:/caminho/para/UE4SS_CXX_Gen/cxx_headers")
    -- add_defines("PPV_USE_CXX_SDK")
