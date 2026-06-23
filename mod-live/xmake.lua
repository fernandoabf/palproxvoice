-- Build do PalProxVoiceLive (mod C++ do UE4SS).
-- Copie esta pasta pra dentro de RE-UE4SS/Mods/PalProxVoiceLive/ e use o build
-- do RE-UE4SS (que ja define o target "UE4SS" e o SDK). Ver README.md.
--
-- NAO testado aqui (sem Windows/MSVC). Ajuste conforme a versao do RE-UE4SS.

add_rules("mode.debug", "mode.release")

target("PalProxVoiceLive")
    set_kind("shared")
    set_languages("cxx20")
    add_files("dllmain.cpp")

    -- dependencias do RE-UE4SS (o build raiz do UE4SS injeta isso normalmente):
    add_deps("UE4SS")
    add_packages("imgui")  -- conforme o setup padrao de mods C++ do UE4SS
