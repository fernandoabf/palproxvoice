<div align="center">

# 🎙️ PalProxVoice

**Self-hosted 3D proximity voice chat for Palworld**

_Think Simple Voice Chat (Minecraft), but for Palworld — you hear whoever's near you in the game world, louder the closer they are, with real direction (3D / HRTF audio). No rooms, no channels, no third-party service: everything runs on your own infra._

![status](https://img.shields.io/badge/status-active%20alpha-2ea44f)
![platform](https://img.shields.io/badge/platform-Windows%20%C2%B7%20Linux-1f6feb)
![game](https://img.shields.io/badge/Palworld-Steam%20%C2%B7%20Game%20Pass-orange)
![self-hosted](https://img.shields.io/badge/self--hosted-yes-success)
![voice](https://img.shields.io/badge/audio-3D%20HRTF%20%2B%20Opus-blueviolet)
![license](https://img.shields.io/badge/license-see%20LICENSE-lightgrey)

🇧🇷 **[Leia em Português](README.pt-BR.md)** · [Releases](../../releases) · [CHANGELOG](CHANGELOG.md) · [Architecture](docs/ARCHITECTURE.md) · [Roadmap](docs/ROADMAP.md)

</div>

> 🎥 **Demo:** _(coming soon — ~20s clip of the 3D proximity voice; listen with **headphones** 🎧)_

---

## 🧩 Versions / Milestones

PalProxVoice grows in layers. The **base** (proximity voice) works on every server; each version adds **better anti-spoof / lower latency** without breaking the ones below it.

| Tag | Status | What it adds | Anti-spoof | Needs | Lives in |
|:---:|:---:|---|:---:|---|---|
| **Base + V1** | 🟢 **working** | proximity voice + **auto-connect** (companion finds the server by your IP and **probes the voice port** by itself) | — (trusts client) | client mod + companion | `companion/` · `server/` · `mod/` |
| **V1.5** | 🟢 **coded** _(off by default)_ | **anti-spoof via REST** — reconciles client position with the Palworld REST API (`verify`/`strict`). A spoofer can't teleport into your ears. Keeps client `Z`+`yaw` (REST has neither). | ✅ horizontal | + Palworld REST reachable internally | `server/antispoof.go` · `PPV_AUTH_MODE` |
| **V2** | 🟢 **working** _(experimental)_ | **server-side authoritative** pos+yaw+FGuid @ 5Hz, read by a **UE4SS mod on the server** (Windows `.exe` under **Proton** on Linux). Zero client position trust. | ✅✅ full 3D | + UE4SS on the dedicated server | `mod-server/` · `deploy/v2-experimental/` · `docker-compose.v2.yml` |
| **V3** | 🔬 **research** _(Phase 0)_ | V2's authoritative data but on the **NATIVE Linux** server — an **external memory reader** (`process_vm_readv` + AOB scan), **no UE4SS, no Proton, no Wine**. | ✅✅ full 3D | + `CAP_SYS_PTRACE` | branch [`experimental/v3-linux-native`](../../tree/experimental/v3-linux-native) |

> **Which do I want?** Most people: **Base + V1** (it just works). Want cheat-proof proximity on a normal server: turn on **V1.5**. Want authoritative 3D positions with minimal delay: **V2** (validated end-to-end). **V3** is the pure-Linux experiment for the brave.

---

## ✨ Features

- **3D positional voice** (Web Audio / HRTF) — left/right/front/back + distance, following your in-game camera.
- **Zero-config auto-connect** — the companion detects the server IP from the game and **finds the voice port by itself** (probe). No typing addresses.
- **Native low-latency mic capture** (WASAPI) — does **not** degrade the rest of your system audio (the classic browser/`getUserMedia` pitfall).
- **Optional AI noise suppression** (RNNoise) + noise gate + compressor + a "hear my mic" monitor.
- **Pick your microphone and output by name.**
- **Built for bad internet** — Opus FEC + DTX, automatic **adaptive bitrate**, and **auto-reconnect** on drops.
- **No rooms** — one shared pool per server + password; who you hear is 100% proximity.
- **Anti-spoof, layered** — from REST reconciliation (V1.5) to fully server-authoritative positions (V2/V3).
- **One-click installer** (UE4SS + mod + companion + auto-start). Works on **Steam and Game Pass** (WinGDK).

---

## 🔧 How it works

Three pieces. The game never talks to the voice server directly — the companion bridges it.

| # | Piece | Where | What it does |
|---|------|------|-----------|
| **mod** | UE4SS (Lua) | each player's PC (Windows/Game Pass) | reads position + facing + FGuid, writes local files |
| **companion** | Wails app (Go + WebView2) | each player's PC | reads the position, sends the mic, receives others and **spatializes in 3D** (Web Audio / HRTF); **auto-connects** |
| **server** | Go + pion/webrtc | owner's VPS | SFU: each mic uploads once and is relayed to all; position relay + anti-spoof. No rooms. |

Each audio track carries `StreamID = peer id`, so the companion matches **audio ↔ position**. With **V2/V3**, the position becomes **server-authoritative** (a mod/reader on the server writes a feed the voice consumes), so the client can't lie about where it is.

---

## 🎮 For players

1. Download the installer from the [latest release](../../releases) (`PalProxVoice-Setup.exe`).
2. Run it — it finds Palworld (or click **Search a drive** to scan), installs UE4SS + the mod + the companion, sets up auto-start. Works on **Steam** and **Game Pass / Microsoft Store** (WinGDK).
3. Launch the game. The companion **finds the server by itself** (IP + voice port) and connects. Use headphones. 🎧

> **"Windows protected your PC"?** SmartScreen warning that the app isn't code-signed yet — normal for a new/open-source binary. **More info → Run anyway.** (Code signing is on the roadmap.)

No installer? Grab `PalProxVoice-UE4SS.zip` (UE4SS + mod, extract into `Pal\Binaries\<Win64|WinGDK>\`) and `palproxvoice.exe` (companion) from the release.

---

## 🖥️ For server owners

The voice server is a Go container. Run it next to your Palworld (same VPS or another).

### Base / V1 (everyone)
```bash
cp .env.example .env          # set VOICE_PASSWORD and PUBLIC_IP=<vps-public-ip> (or "auto")
docker compose up -d --build
```
- The companion **auto-detects** the game IP and **probes** common voice ports (`8765`, `8766`, …) — expose the voice on one of them.
- **Media (audio):** open **UDP 50000–50010** in the firewall. `PUBLIC_IP` makes pion announce your IP (no TURN). `PUBLIC_IP=auto` self-detects.
- **Mic in a browser** needs a secure context — put the voice behind a TLS domain (`wss://`). The desktop companion connects by direct IP.

### V1.5 — anti-spoof (REST)
Set `PPV_AUTH_MODE=verify` (or `strict`) + `PPV_REST_URL`/`PPV_REST_PASS` so the voice reaches the **Palworld REST API internally** (admin API — **never** expose it publicly). The voice reconciles each player's reported position with the REST and ignores sustained lies. See [docs/ANTI-SPOOF-DEPLOY.md](docs/ANTI-SPOOF-DEPLOY.md).

### V2 — server-side authoritative (experimental)
A **UE4SS mod on the dedicated server** writes a `fguid,x,y,z,yaw,name` feed @ 5Hz; the voice consumes it (zero client trust). On Linux the Windows `.exe` runs under **Proton**. One-shot deploy (Dokploy or local):
```bash
docker compose -f docker-compose.v2.yml up -d --build
```
Full notes (the Proton gotchas we solved: steamclient symlink, headless GUI console, esync/fsync, NetDriver timeout): [deploy/v2-experimental/README.md](deploy/v2-experimental/README.md).

### V3 — native Linux (research)
No Proton/UE4SS — an external memory reader on the native `thijsvanloef`-style server. See the [`experimental/v3-linux-native`](../../tree/experimental/v3-linux-native) branch.

---

## 🧑‍💻 For developers

### Repo layout
```
mod/PalProxVoice/        UE4SS mod (CLIENT) — reads pos+yaw+FGuid -> C:\Users\Public\palproxvoice_*.txt
mod-server/              [V2] UE4SS mod (SERVER) — authoritative pos+yaw+FGuid feed @ 5Hz
companion/               Wails desktop app (Go+WebView2) — 3D voice + auto-connect (IP detect + port probe)
server/                  Go voice server — SFU + position relay + anti-spoof (V1.5 REST, V2 feed)
deploy/v2-experimental/  [V2] Palworld server MODADO (Proton+UE4SS): linux/ (Docker) + windows/ (install.ps1)
v3-linux-native/         [V3] native-Linux memory reader (Phase 0) — on the experimental/v3-linux-native branch
installer/               Inno Setup installer (UE4SS + mod + companion + auto-start)
docs/                    ARCHITECTURE · ROADMAP · ANTI-SPOOF-DEPLOY
docker-compose.yml       voice server (Base / V1 / V1.5)
docker-compose.v2.yml    [V2] Palworld modado (Proton) + voice — for Dokploy
```

### Run locally (no game, no VPS)
```bash
cp .env.example .env          # VOICE_PASSWORD=test, PUBLIC_IP empty
docker compose up --build     # voice server only (API; no built-in web client)
```
The server is **API-only** — connect with the **companion** pointed at `localhost:<HTTP_PORT>`. You need two clients (two machines) to hear the SFU + spatialization.

### Build the companion
Windows (WebView2) or via GitHub Actions on a `v*` tag push. See [companion/BUILD.md](companion/BUILD.md). Go tests: `go test ./...` in `companion/`.

### Contributing
Issues and PRs welcome. The coupling with the game is thin and isolated on purpose (Palworld 1.0 ships 2026-07-10 and big updates break UE4SS mods) — check the locked decisions in the [ROADMAP](docs/ROADMAP.md) before touching `mod/`.

---

## 🗺️ Roadmap

| | Milestone | Status |
|:---:|---|:---:|
| **V1** | auto-connect: the companion finds the server by your IP + **probes the voice port** | 🟢 done |
| **V1.5** | anti-spoof: source-agnostic protocol + REST reconciliation (`verify`/`strict`) | 🟢 coded |
| **V2** | server-side: UE4SS mod on the server (Proton) with authoritative pos+yaw; client needs no UE4SS for position | 🟢 working |
| **V3** | native Linux: external memory reader (`process_vm_readv`), no Proton/Wine | 🔬 research |
| **next** | code signing · push-to-talk · in-game voice-address announce (server → client RPC) | 🔜 planned |

Details in [docs/ROADMAP.md](docs/ROADMAP.md).

## 📄 License

See [LICENSE](LICENSE).
