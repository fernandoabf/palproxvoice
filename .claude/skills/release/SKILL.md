---
name: release
description: Cortar uma nova versão do PalProxVoice — o que editar (AppVersion do instalador, CHANGELOG, status do README/ROADMAP) e o fluxo merge → tag vX.Y.Z → release pelo GitHub Actions. Use quando for lançar/publicar uma versão nova do projeto.
---

# Lançar uma nova versão do PalProxVoice

A **tag git `vX.Y.Z`** é o que dispara o build/release (workflow `.github/workflows/release.yml`,
em `on: push: tags: v*`). Ele builda o companion + watcher + zipa o mod + monta o bundle UE4SS +
compila o instalador (Inno Setup) e **cria o GitHub Release** com esses assets.

> Convenção: sufixo `-alpha`/`-beta` (ex.: `v0.10.0-alpha`) → o release sai como **prerelease**
> automaticamente (CI: `prerelease: contains(ref_name, '-')`).

## 1. O que EDITAR a cada versão

| Arquivo | O que mudar |
|---|---|
| **`installer/palproxvoice.iss`** | `AppVersion=X.Y.Z` (no `[Setup]`). |
| **`CHANGELOG.md`** | Adicionar a versão no topo do milestone atual, com **Adicionado / Alterado / Corrigido**. |
| **`README.md` + `README.pt-BR.md`** | Só se o **status/milestone** mudou (ex.: "alpha ativo" → "V1 lançada"). Mantenha os dois em sincronia. |
| **`docs/ROADMAP.md`** | Só se o milestone avançou (V1 → V1.5 → V2). |

> Não existe versão em `companion/wails.json` nem em código Go — não precisa mexer lá.
> `companion/build/` e `companion/frontend/wailsjs/` são **gerados** (gitignored) — não commitar.

## 2. Verificar o build (local, Windows)

Go fica em `C:\Program Files\Go\bin` e o wails em `%USERPROFILE%\go\bin` — adicione ao PATH:

```powershell
$env:Path = "C:\Program Files\Go\bin;$env:USERPROFILE\go\bin;" + $env:Path
cd companion; go build ./...; go test ./...
cd ../server; go build ./...; go test ./...
```

(Opcional, build completo local: `cd companion; wails build -platform windows/amd64 -o palproxvoice.exe`.)

## 3. Commitar o bump + CHANGELOG na `develop` e pushar

```powershell
git add installer/palproxvoice.iss CHANGELOG.md README.md README.pt-BR.md
git commit -m "release: vX.Y.Z (resumo)"
git push origin develop
```

## 4. Atualizar a `main` (homepage do repo)

A página inicial do GitHub mostra a `main`. Para a release/divulgação ficar com tudo atual,
**mergeie `develop` → `main`** (PR). ⚠️ Merge na branch principal **exige OK explícito do usuário** —
peça confirmação ("pode mergear") antes.

```
gh pr create --base main --head develop --title "release: vX.Y.Z develop -> main" --body "..."
gh pr merge <n> --merge
```

## 5. Tag → dispara o release

Tag no commit que tem o código (geralmente o topo da `main` após o merge, ou da `develop`):

```powershell
git fetch origin; git checkout main; git pull origin main   # ou develop
git tag -a vX.Y.Z-alpha -m "vX.Y.Z-alpha: <resumo>"
git push origin vX.Y.Z-alpha
```

## 6. Acompanhar o Actions até ficar verde

```
gh run watch <run-id> --exit-status
```

Confirme que rodou o passo **"Create GitHub Release and attach files"** (só roda em tag). Assets do
release: `palproxvoice.exe`, `palproxvoice-watcher.exe`, `PalProxVoice-mod.zip`,
`PalProxVoice-UE4SS.zip`, `PalProxVoice-Setup.exe`.

## Notas de ambiente (este setup)

- **gh** está instalado mas `gh auth login` é interativo (não roda headless). Reaproveite o token do
  Git Credential Manager: jogue `protocol=https` + `host=github.com` + linha vazia em
  `git credential fill`, pegue o `password=` e exporte como `$env:GH_TOKEN` pros comandos `gh`.
- O **push HTTPS** funciona pelo GCM no PowerShell (no Bash dá "auth failed").
- Pós-release: gravar/atualizar o **`docs/demo.gif`** e divulgar (Nexus, r/Palworld, Show HN).
- **Code signing** do `.exe` (mata o SmartScreen) está no roadmap — ideal antes de um 1.0/divulgação grande.
