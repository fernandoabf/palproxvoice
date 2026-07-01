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

**You need:** Palworld (**Steam _or_ Game Pass**) + headphones + **a friend or server owner running the PalProxVoice voice server** (see [For server owners](#-for-server-owners) — if nobody set one up yet, send them that link).

1. Download **`PalProxVoice-Setup.exe`** from the [latest release](../../releases).
2. **Run it.** Windows asks for **administrator permission** (blue prompt) — click **Yes** (needed to install the mod inside the game). On **Steam** it finds Palworld automatically; if it doesn't (e.g. **Game Pass**), click **Search a drive** and pick the Palworld folder.
3. **Open Palworld and play.** The voice **connects by itself** — no IP or port to type. If your server has a voice password, ask the owner for it. Use headphones. 🎧

That's it. Whoever's near you in the world, you hear — louder the closer, and with direction (left/right/front/back).

> **Got "Windows protected your PC"?** Just Windows warning about new apps that aren't paid-code-signed. Click **More info → Run anyway**. It's safe (open-source).

<details><summary><b>No installer? (manual setup)</b></summary>

From the same release, grab `PalProxVoice-UE4SS.zip` (extract into `Pal\Binaries\<Win64 or WinGDK>\`) and `palproxvoice.exe` (the companion). Run the companion — it auto-connects the same way.
</details>

### 🗣️ Voice channels
Like the game's chat. Press **Alt+V** while playing to switch which channel **you talk on** (the current one shows in the PalProxVoice app):

- **📍 Proximity** _(default)_ — talk to who's near you, in 3D.
- **🛡️ Guild** — talk to your guild, at any distance.
- **🌐 Global** — talk to the whole server.

You always **hear** guild (same guild) and global; proximity only when someone's close.

### ✨ What you get
- **3D positional voice** — direction + distance, following your in-game camera.
- **Zero setup** — connects to your current server on its own. Nothing to type.
- **Doesn't ruin your other audio** — your music and the game's sound stay normal while you talk (the classic browser voice-chat problem is fixed).
- **AI noise suppression** (optional) — cuts background noise like a fan or keyboard. Plus a mic-tuning panel for the picky.
- **Pick your mic and output by name.**
- **Built for bad internet** — quality adjusts itself + auto-reconnect on drops.
- **Runs with the game** — opens with Palworld, hides when you close it. You don't open anything.

---

## 🧩 Versions

PalProxVoice grows in layers. The **base** works on any server; each version adds **stronger anti-cheat / lower delay** without breaking the ones below. As a player you don't pick — the server owner does.

| Tag | Status | What it adds | Anti-spoof | Server needs |
|:---:|:---:|---|:---:|---|
| **Base + V1** | 🟢 **working** | proximity voice + **auto-connect** (finds the server IP and the voice port by itself) | — _trusts client_ | just the voice container |
| **V1.5** | 🟢 **coded** _(opt-in)_ | **REST reconciliation** — a spoofer can't teleport into your ears. Keeps client `Z`+`yaw`. | ✅ horizontal | + Palworld REST reachable internally |
| **V2** | 🟡 **server-side works** _(exp.)_ | **server-side authoritative** pos+yaw+FGuid @5Hz via a **UE4SS mod on the server** (Windows `.exe` under **Proton** on Linux). Zero client trust. _Voice e2e (2 players) not tested yet._ | ✅✅ full 3D | + UE4SS on the dedicated server |
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
- **Voice port:** `.env`'s `HTTP_PORT` is the **host** port (default **8765**). The companion **probes ports 8765–8768** and connects to the first PalProxVoice it finds — so expose it on one of those.
- **Browser mic** needs HTTPS (`wss://`, e.g. behind Dokploy). The **desktop companion connects by direct IP — no TLS/proxy needed** (that's the normal path).

### 2️⃣ V1.5 — anti-spoof (REST), opt-in
Set in `.env`: `PPV_AUTH_MODE=verify` (or `strict`) + `PPV_REST_URL=http://<palworld-service>:8212` + `PPV_REST_PASS=<Palworld AdminPassword>`. The REST is an **admin** API — **never** expose it publicly; the voice reaches it **inside the docker network**. So the voice and Palworld must share a network: this compose creates the **`palprox`** network — bring the **voice up first** (it creates it), then put Palworld in it as `external` (see [`docker-compose.palworld.example.yml`](docker-compose.palworld.example.yml)). The voice reconciles each player's reported position with the REST and ignores sustained lies; it keeps the client's height + facing (the REST has neither). → [docs/ANTI-SPOOF-DEPLOY.md](docs/ANTI-SPOOF-DEPLOY.md)

### 3️⃣ V2 — server-side authoritative (experimental)
A **UE4SS mod on the dedicated server** writes a `fguid,x,y,z,yaw` feed @5Hz; the voice consumes it (zero client trust). On Linux the Windows `.exe` runs under **Proton**. One command (Dokploy or local):
```bash
docker compose -f docker-compose.v2.yml up -d --build
```
- **Shifted ports** (to coexist with a prod Palworld): game **8311/udp**, query **27115/udp**, voice **8766**, media **UDP 50100–50110** — open those.
- **Status:** the server-side is validated in-game (server listens, mod loads, feed writes real pos+FGuid); the **voice end-to-end with 2 players isn't tested yet**. Run on a **throwaway host** first.

> The fragile part is **UE4SS-under-Proton** — we solved the whole chain (steamclient symlink, headless GUI console, esync/fsync, NetDriver timeout, mod rate). Full notes + the Windows-native path: **[deploy/v2-experimental/README.md](deploy/v2-experimental/README.md)**.

### 🗣️ Voice channels (any version)
Proximity / guild / global work out of the box. **Guild membership** comes from the player's mod (auto) or a shared **guild code** in the companion (fallback). No extra server config.

---

## 🧑‍💻 For developers

### Repo layout
```
mod/PalProxVoice/        UE4SS mod (CLIENT) — reads pos+yaw+FGuid -> C:\Users\Public\palproxvoice_*.txt
mod-server/              [V2] UE4SS mod (SERVER) — authoritative pos+yaw+FGuid feed (fguid,x,y,z,yaw) @5Hz
companion/               Wails app (Go+WebView2) — 3D voice, auto-connect (IP detect + port probe + ETW),
                         voice channels, native WASAPI mic
server/                  Go voice server — SFU + position/channel relay + anti-spoof (V1.5 REST, V2 feed)
deploy/v2-experimental/  [V2] Palworld MODDED server (Proton+UE4SS): linux/ (Docker) + windows/ (install.ps1)
v3-linux-native/         [V3] native-Linux memory reader (scaffold) — on the experimental/v3-linux-native branch
installer/               Inno Setup installer (UE4SS + mod + companion + auto-start)
docs/                    ARCHITECTURE · ROADMAP · ANTI-SPOOF-DEPLOY
docker-compose.yml                    voice server (Base / V1 / V1.5)
docker-compose.v2.yml                 [V2] Palworld modded (Proton) + voice — for Dokploy
docker-compose.palworld.example.yml   example: a Palworld with REST on the `palprox` network (for V1.5)
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
| **V2** | server-side authoritative pos+yaw via UE4SS (Proton) | 🟡 server-side works |
| **Channels** | proximity / guild / global + global push-to-switch (Alt+V) | 🟢 built |
| **Realtime IP** | ETW reads the current server IP live from kernel UDP | 🟢 built |
| **V3** | native-Linux external memory reader (no Proton/Wine) | 🔬 research |
| **next** | code signing · auto-guild finalize · in-game voice-address announce | 🔜 planned |

Full detail in [docs/ROADMAP.md](docs/ROADMAP.md).

## 📄 License

See [LICENSE](LICENSE).
