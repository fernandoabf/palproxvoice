<div align="center">

# 🎙️ PalProxVoice

**Self-hosted 3D proximity voice chat for Palworld.**

_You hear whoever's near you in the game — louder the closer they are, with real direction (3D / HRTF audio). Like Minecraft's Simple Voice Chat, but for Palworld. No rooms, no third-party service: everything runs on your own infra._

![status](https://img.shields.io/badge/status-active%20alpha-2ea44f)
![platform](https://img.shields.io/badge/platform-Windows%20%C2%B7%20Linux-1f6feb)
![game](https://img.shields.io/badge/Palworld-Steam%20%C2%B7%20Game%20Pass-orange)
![self-hosted](https://img.shields.io/badge/self--hosted-yes-success)
![audio](https://img.shields.io/badge/audio-3D%20HRTF%20%2B%20Opus-blueviolet)
![license](https://img.shields.io/badge/license-see%20LICENSE-lightgrey)

🇧🇷 **[Leia em Português](README.pt-BR.md)** · [⬇️ Download](../../releases) · [📝 Changelog](CHANGELOG.md) · [🏗️ Architecture](docs/ARCHITECTURE.md) · [🗺️ Roadmap](docs/ROADMAP.md)

</div>

> 🎥 **Demo:** _(coming soon — ~20s of the 3D proximity voice; use **headphones** 🎧)_

---

## 🚀 Start here — pick your path

| You are… | Go to |
|---|---|
| 🎮 **I just want to talk to my friends** | **[For players](#-for-players)** — download, install, play (~2 min) |
| 🖥️ **I run a Palworld server** | **[For server owners](#-for-server-owners)** |
| 🧑‍💻 **I want to build / contribute** | **[For developers](#-for-developers)** |

---

## 🎮 For players

**You need:** Palworld (**Steam _or_ Game Pass**) + headphones.

1. Download **`PalProxVoice-Setup.exe`** from the [latest release](../../releases).
2. **Double-click it.** It finds Palworld on its own, installs everything, and sets it to open with the game.
3. **Open Palworld and play.** The voice **connects by itself** — no IP, no password, no port to type. Use headphones. 🎧

That's it. Whoever's near you in the world, you hear — louder the closer, and with direction (left/right/front/back).

> **Got "Windows protected your PC"?** That's just Windows warning about new apps that aren't paid-code-signed. Click **More info → Run anyway**. It's safe (the code is open-source).

<details><summary><b>No installer? (manual setup)</b></summary>

From the same release, grab `PalProxVoice-UE4SS.zip` (extract into `Pal\Binaries\<Win64 or WinGDK>\`) and `palproxvoice.exe` (the companion). Run the companion — it auto-connects the same way.
</details>

### 🗣️ Voice channels
Like the game's chat. Press **Alt+V** while playing to switch which channel you talk on:

- **📍 Proximity** _(default)_ — hear who's near you, in 3D.
- **🛡️ Guild** — talk to your guild, at any distance.
- **🌐 Global** — talk to the whole server.

You always **hear** guild (same guild) and global; proximity only when close.

### ✨ What you get
- **3D positional voice** — direction + distance, following your in-game camera.
- **Zero setup** — auto-connects to your current server. Nothing to type.
- **Doesn't wreck your other audio** — native mic capture (the classic browser voice-chat pitfall is fixed).
- **Optional AI noise suppression** (RNNoise), noise gate, compressor, "hear my mic" monitor.
- **Pick your mic and output by name.**
- **Built for bad internet** — auto-adjusting quality + auto-reconnect on drops.
- **Runs as an overlay** — appears with the game, hides when you close it.

---

## 🧩 Versions

PalProxVoice grows in layers. The **base** works on any server; each version adds **stronger anti-cheat / lower delay** without breaking the ones below. As a player you don't pick — the server owner does.

| Tag | Status | What it adds | Anti-spoof | Server needs |
|:---:|:---:|---|:---:|---|
| **Base + V1** | 🟢 **working** | proximity voice + **auto-connect** (finds the server IP and the voice port by itself) | — _trusts client_ | just the voice container |
| **V1.5** | 🟢 **coded** _(opt-in)_ | **REST reconciliation** — a spoofer can't teleport into your ears. Keeps client `Z`+`yaw`. | ✅ horizontal | + Palworld REST reachable internally |
| **V2** | 🟢 **working** _(experimental)_ | **server-side authoritative** pos+yaw+FGuid @5Hz via a **UE4SS mod on the server** (Windows `.exe` under **Proton** on Linux). Zero client trust. | ✅✅ full 3D | + UE4SS on the dedicated server |
| **V3** | 🔬 **research** _(scaffold only)_ | V2's data but on the **native Linux** server — an external memory reader, **no UE4SS / Proton / Wine**. | ✅✅ full 3D | _(not built yet)_ |

> **Which should I run?** Most people: **Base + V1** (it just works). Cheat-proof proximity on a normal server: turn on **V1.5**. Authoritative 3D positions with minimal delay: **V2**. **V3** is an early experiment — see [its branch](../../tree/experimental/v3-linux-native).

---

## 🔧 How it works

Three pieces. The game never talks to the voice server directly — the companion bridges it.

| Piece | Where | What it does |
|---|---|---|
| **mod** (UE4SS/Lua) | each player's PC (Steam **or** Game Pass) | reads position + facing + identity, writes local files |
| **companion** (Wails app) | each player's PC | reads the position, sends the mic, receives others, **spatializes in 3D**, and **auto-connects** |
| **voice server** (Go/pion) | owner's VPS | SFU: each mic uploads once and is relayed to all; position + channel relay. No rooms. |

Each audio track carries `StreamID = peer id`, so the companion matches **audio ↔ position**. With **V2/V3**, the position is **server-authoritative** (a mod/reader on the server writes a feed the voice consumes), so the client can't lie about where it is.

---

## 🖥️ For server owners

The voice server is one Go container. Run it next to your Palworld (same VPS or another). Everyone who joins with the password is one pool; who you hear is 100% proximity/channel — no rooms.

### 1️⃣ Base / V1 — everyone starts here
```bash
cp .env.example .env          # set VOICE_PASSWORD and PUBLIC_IP=<vps-ip>  (or PUBLIC_IP=auto)
docker compose up -d --build
```
- **Firewall:** open **UDP 50000–50010** (the audio). `PUBLIC_IP` makes the server announce your IP (no TURN); `auto` self-detects.
- **Auto-connect:** the companion detects the game's IP and **probes common voice ports** (8765, 8766, …) — just expose the voice on one of them.
- **Browser mic** needs HTTPS (`wss://`, e.g. behind Dokploy). The desktop companion connects by direct IP — no TLS needed.

### 2️⃣ V1.5 — anti-spoof (REST), opt-in
Set in `.env`: `PPV_AUTH_MODE=verify` (or `strict`) + `PPV_REST_URL` / `PPV_REST_PASS` so the voice reaches the **Palworld REST API internally** (it's an admin API — **never** expose it publicly). The voice reconciles each player's reported position with the REST and ignores sustained lies; it keeps the client's height + facing (the REST has neither). → [docs/ANTI-SPOOF-DEPLOY.md](docs/ANTI-SPOOF-DEPLOY.md)

### 3️⃣ V2 — server-side authoritative (experimental)
A **UE4SS mod on the dedicated server** writes a `fguid,x,y,z,yaw,name` feed @5Hz; the voice consumes it (zero client trust). On Linux the Windows `.exe` runs under **Proton**. One command (Dokploy or local):
```bash
docker compose -f docker-compose.v2.yml up -d --build
```
> The fragile part is **UE4SS-under-Proton** — we solved the whole chain (steamclient symlink, headless GUI console, esync/fsync, NetDriver timeout, mod rate). Full notes + the Windows-native path: **[deploy/v2-experimental/README.md](deploy/v2-experimental/README.md)**.

### 🗣️ Voice channels (any version)
Proximity / guild / global work out of the box. **Guild membership** comes from the player's mod (auto) or a shared **guild code** in the companion (fallback). No extra server config.

---

## 🧑‍💻 For developers

### Repo layout
```
mod/PalProxVoice/        UE4SS mod (CLIENT) — reads pos+yaw+FGuid -> C:\Users\Public\palproxvoice_*.txt
mod-server/              [V2] UE4SS mod (SERVER) — authoritative pos+yaw+FGuid+name feed @5Hz
companion/               Wails app (Go+WebView2) — 3D voice, auto-connect (IP detect + port probe + ETW),
                         voice channels, native WASAPI mic
server/                  Go voice server — SFU + position/channel relay + anti-spoof (V1.5 REST, V2 feed)
deploy/v2-experimental/  [V2] Palworld MODDED server (Proton+UE4SS): linux/ (Docker) + windows/ (install.ps1)
v3-linux-native/         [V3] native-Linux memory reader (scaffold) — on the experimental/v3-linux-native branch
installer/               Inno Setup installer (UE4SS + mod + companion + auto-start)
docs/                    ARCHITECTURE · ROADMAP · ANTI-SPOOF-DEPLOY
docker-compose.yml       voice server (Base / V1 / V1.5)
docker-compose.v2.yml    [V2] Palworld modded (Proton) + voice — for Dokploy
```

### Run locally (no game, no VPS)
```bash
cp .env.example .env          # VOICE_PASSWORD=test, PUBLIC_IP empty
docker compose up --build     # voice server only (API; no built-in web client)
```
The server is **API-only** — connect with the **companion** pointed at `localhost:<HTTP_PORT>`. You need two clients (two machines) to hear the SFU + spatialization.

### Build the companion
Windows (WebView2), or via GitHub Actions on a `v*` tag. See [companion/BUILD.md](companion/BUILD.md). Go tests: `go test ./...` in `companion/`.

### Contributing
Issues and PRs welcome. The coupling with the game is thin and isolated on purpose (Palworld 1.0 ships 2026-07-10 and big updates break UE4SS mods) — check the locked decisions in the [ROADMAP](docs/ROADMAP.md) before touching `mod/`.

---

## 🗺️ Roadmap (short)

| Milestone | What | Status |
|---|---|:---:|
| **V1** | auto-connect: find the server by IP + **probe the voice port** | 🟢 done |
| **V1.5** | anti-spoof via REST reconciliation (`verify`/`strict`) | 🟢 coded |
| **V2** | server-side authoritative pos+yaw via UE4SS (Proton) | 🟢 working |
| **Channels** | proximity / guild / global + global push-to-switch (Alt+V) | 🟢 built |
| **Realtime IP** | ETW reads the current server IP live from kernel UDP | 🟢 built |
| **V3** | native-Linux external memory reader (no Proton/Wine) | 🔬 research |
| **next** | code signing · auto-guild finalize · in-game voice-address announce | 🔜 planned |

Full detail in [docs/ROADMAP.md](docs/ROADMAP.md).

## 📄 License

See [LICENSE](LICENSE).
