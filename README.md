# PalProxVoice

**Self-hosted 3D proximity voice chat for Palworld** — think Simple Voice Chat
(Minecraft), but for Palworld. You hear whoever's near you in the game world,
louder the closer they are, with real direction (3D / HRTF audio). No rooms, no
channels: one pool per server + password + proximity. No third-party service —
everything runs on your own infra.

🇧🇷 **[Leia em Português](README.pt-BR.md)**

> Status: **active alpha** (milestone **V1**) — validated end-to-end with real
> people over the internet. Grab the [latest release](../../releases) · history in
> [CHANGELOG.md](CHANGELOG.md) · design in [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)
> · plan in [docs/ROADMAP.md](docs/ROADMAP.md).

> 🎥 **Demo:** _(coming soon — ~20s clip of the 3D proximity voice; listen with **headphones** 🎧)_
<!-- When you record the clip, save it to docs/demo.gif and uncomment:
<p align="center">
  <img src="docs/demo.gif" alt="3D proximity voice in Palworld" width="680">
</p>
-->

## Features

- **3D positional voice** (Web Audio / HRTF) — left/right/front/back + distance, following your in-game camera.
- **Native low-latency mic capture** (WASAPI) — does **not** degrade the rest of your system audio (the classic browser/`getUserMedia` pitfall).
- **Optional AI noise suppression** (RNNoise) + noise gate + compressor + a "hear my mic" monitor.
- **Pick your microphone and output by name.**
- **Built for bad internet** — Opus FEC + DTX, automatic **adaptive bitrate**, and **auto-reconnect** on drops.
- **No rooms** — one shared pool per server + password; who you hear is 100% proximity.
- **One-click installer** (UE4SS + mod + companion + auto-start).

## How it works

Three pieces. The game never talks to the voice server directly — the companion bridges it.

| # | Piece | Where | What it does |
|---|------|------|-----------|
| **mod** | UE4SS (Lua) | each player's PC (Windows) | reads position + facing and writes to a local file |
| **companion** | Wails app (Go + WebView2) | each player's PC | reads the position, sends the mic, receives others and **spatializes in 3D** (Web Audio/HRTF) |
| **server** | Go + pion/webrtc | owner's VPS | SFU: each mic uploads once and is relayed to all; position relay. No rooms. |

Each audio track carries `StreamID = peer id`, so the companion matches **audio ↔ position**.

---

## For players

1. Download the installer from the [latest release](../../releases) (`PalProxVoice-Setup.exe`).
2. Run it — it finds Palworld (or click **Search a drive** to scan), installs UE4SS + the mod + the companion, and sets up auto-start.
3. Launch the game. The companion **finds the server by itself** and connects the voice. Use headphones. 🎧

> **"Windows protected your PC"?** That's SmartScreen warning that the app isn't
> code-signed yet — normal for a new/open-source binary. Click **More info → Run
> anyway**. (Code signing is on the roadmap.)

No installer? Grab `PalProxVoice-UE4SS.zip` (UE4SS + mod, extract into
`Pal\Binaries\<Win64|WinGDK>\`) and `palproxvoice.exe` (companion) from the release.

---

## For server owners

The voice server is a Go container. Run it next to your Palworld (same VPS or another).

```bash
cp .env.example .env          # set VOICE_PASSWORD and PUBLIC_IP=<vps-public-ip>
docker compose up -d --build
```

- **Port:** the companion auto-connects to `game-ip:8765` by default. Expose the
  voice on that port (map host `8765` → container `8080`) **or** adjust `autoPort`
  in the companion config to match yours.
- **Mic needs a secure context:** for browser access, put the voice behind a TLS
  domain (e.g. Dokploy) and use `wss://`. The desktop companion connects by direct IP.
- **Media (audio):** open **UDP 50000–50010** in the firewall. `PUBLIC_IP` makes
  pion announce your IP (no TURN).
- **No rooms:** everyone who joins with the password is one pool; who you hear is
  100% proximity.

TLS/firewall details and an optional test Palworld: see
[docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) and `docker-compose.palworld-test.yml`.

---

## For developers

### Repo layout

```
mod/PalProxVoice/    UE4SS mod (Lua) — reads position+yaw, writes C:\Users\Public\palproxvoice_pos.txt
companion/           Wails desktop app (Go+WebView2) — 3D voice + auto-connect   (BUILD.md)
server/              Go voice server (SFU + position relay) + Dockerfile + web/ (test client)
mod-live/            [V1] optional C++ mod — current session IP over a socket (scaffold, build on Windows)
bridge/              [legacy] file→HTTP bridge for a plain browser; the companion already does this
installer/           Inno Setup installer (UE4SS + mod + companion + auto-start)
docs/                ARCHITECTURE.md · ROADMAP.md
docker-compose*.yml  voice · test palworld · local.sh
```

### Run locally (no game, no VPS)

```bash
cp .env.example .env          # VOICE_PASSWORD=test, PUBLIC_IP empty
docker compose up --build     # voice only
```

Open `http://localhost:8088` in 2 tabs (headphones!), same password, **Join**. Move
with `W A S D`, turn with `← →` — the other tab changes side and volume. This
exercises the SFU + spatialization without Palworld. (Port = `HTTP_PORT` from
`.env`; on WSL2 use 8088.) `./local.sh` brings up the voice **+** a test Palworld together.

### Build the companion

Windows (WebView2) or via GitHub Actions on a `v*` tag push. See
[companion/BUILD.md](companion/BUILD.md). Go tests: `go test ./...` in `companion/`.

### Contributing

Issues and PRs welcome. The coupling with the game is thin and isolated on purpose
(Palworld 1.0 ships 2026-07-10 and big updates break UE4SS mods) — check the locked
decisions in the [ROADMAP](docs/ROADMAP.md) before touching `mod/`.

---

## Roadmap (summary)

- **V1** — auto-connect: the companion finds the server by your current IP and connects on its own.
- **V1.5** — anti-spoof: source-agnostic protocol + reconciliation (`verify`/`strict`) by `userId` + IP-match.
- **V2** — server-side: mod on the server (Proton) with authoritative position+yaw; no UE4SS on the client.

Details in [docs/ROADMAP.md](docs/ROADMAP.md).

## License

See [LICENSE](LICENSE).
