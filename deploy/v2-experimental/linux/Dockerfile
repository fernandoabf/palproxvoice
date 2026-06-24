# PalProxVoice V2 — imagem do servidor Palworld MODADO (Linux + Proton + UE4SS)
# ⚠️ EXPERIMENTAL / NÃO TESTADO. Base de Proton-em-docker é a parte mais frágil;
# se der trabalho, considere usar o Dekita/palhub-server (imagem comprovada) e só
# montar o mod PalProxVoiceServer nele.
FROM debian:bookworm-slim

ARG GE_PROTON=GE-Proton9-20

# python3 = OBRIGATORIO: o launcher do Proton e' um script Python (sem ele:
#   "/usr/bin/env: 'python3': No such file or directory").
# O resto = libs que o wine do GE-Proton linka em runtime (headless, sem grafico):
# freetype (fontes), gnutls (TLS/HTTPS do Steam), libgl1/libvulkan1 (o wine PROBA
# mesmo headless e aborta se faltam). lib32* porque o wine roda 32 e 64 bits.
RUN dpkg --add-architecture i386 && apt-get update && apt-get install -y --no-install-recommends \
      python3 locales \
      curl ca-certificates unzip tar xz-utils zstd \
      lib32gcc-s1 libgcc-s1 lib32stdc++6 \
      libfreetype6 libfreetype6:i386 \
      libgnutls30 libgnutls30:i386 \
      libgl1 libgl1:i386 libvulkan1 libvulkan1:i386 \
    && rm -rf /var/lib/apt/lists/*

# steamcmd
RUN mkdir -p /steamcmd && \
    curl -fsSL https://steamcdn-a.akamaihd.net/client/installer/steamcmd_linux.tar.gz \
    | tar -xz -C /steamcmd

# GE-Proton (ajuste GE_PROTON pra uma versão recente se precisar)
RUN mkdir -p /proton && \
    curl -fsSL "https://github.com/GloriousEggroll/proton-ge-custom/releases/download/${GE_PROTON}/${GE_PROTON}.tar.gz" \
    | tar -xz -C /proton --strip-components=1

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
