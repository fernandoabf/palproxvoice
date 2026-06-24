# PalProxVoice V2 — imagem do servidor Palworld MODADO (Linux + Proton + UE4SS)
# ⚠️ EXPERIMENTAL / NÃO TESTADO. Base de Proton-em-docker é a parte mais frágil;
# se der trabalho, considere usar o Dekita/palhub-server (imagem comprovada) e só
# montar o mod PalProxVoiceServer nele.
FROM debian:bookworm-slim

ARG GE_PROTON=GE-Proton9-20

RUN dpkg --add-architecture i386 && apt-get update && apt-get install -y --no-install-recommends \
      curl ca-certificates unzip tar xz-utils \
      lib32gcc-s1 libgcc-s1 libfreetype6 libfreetype6:i386 \
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
