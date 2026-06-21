# Build do Companion (Wails v2)

O companion é um app **Wails v2** (Go + WebView2) que junta a bridge e o
navegador num único `.exe`. Frontend é **vanilla** (sem npm/framework); o Go
embuti `frontend/dist` via `//go:embed`.

O build final só roda no **Windows** (precisa do WebView2). Em outras
plataformas, use a GitHub Action (seção [Release pela Action](#release-pela-action)).

---

## Build local no Windows

### 1. Instalar Go (1.23+)

Baixe de <https://go.dev/dl> o instalador `go1.23.x.windows-amd64.msi` e execute.

```powershell
go version
# go version go1.23.x windows/amd64
```

### 2. Instalar a Wails CLI

```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@latest

wails version
# v2.x.x
```

> Se `wails` não for reconhecido, adicione `%USERPROFILE%\go\bin` ao PATH.

### 3. Instalar o WebView2 Runtime (obrigatório)

Já vem no Windows 11 e na maioria dos Windows 10 atualizados. Se faltar:

```powershell
# Winget (Windows 10+)
winget install Microsoft.WebView2Runtime
```

Ou baixe o **Evergreen Runtime Installer (x64)** em
<https://developer.microsoft.com/microsoft-edge/webview2/>.

### 4. Validar o setup

```powershell
cd companion
wails doctor
# tudo verde: Go, Node, WebView2, platform windows/amd64
```

### 5. Compilar o executável

```powershell
cd companion
wails build -platform windows/amd64 -o palproxvoice.exe
```

Saída em `companion/build/bin/palproxvoice.exe`:

- **Standalone** — o usuário final não precisa de Go instalado.
- Usa o **WebView2 Runtime** do sistema.
- Tamanho típico: **30–50 MB**.

---

## Release pela Action

O workflow `.github/workflows/release.yml` builda no `windows-latest` e dispara:

- **Em tag `v*`** (push de tag) — cria o GitHub Release e anexa o `.exe`.
- **Manual** (`workflow_dispatch`) — só gera o artifact, sem release.

Roda no free tier (~3–5 min, com cache de Go), `working-directory: ./companion`.

### Publicar uma versão

```bash
git tag v1.0.0
git push origin v1.0.0
```

A Action compila e o `palproxvoice.exe` aparece:

- no **Release** da tag (download direto), e
- como **artifact** `windows-executable` (retém 30 dias).
