# Anti-spoof (V1.5) — guia de deploy do `verify`

Como ligar e testar o anti-spoof em produção. O motor já está commitado
(`server/antispoof.go`), **desligado por padrão** (`PPV_AUTH_MODE=off`).

## TL;DR

```env
PPV_AUTH_MODE=verify
PPV_REST_URL=http://palworld-server:8212   # nome do service do Palworld na rede docker
PPV_REST_USER=admin
PPV_REST_PASS=<ADMIN_PASSWORD do Palworld>
# opcionais (defaults entre parenteses):
# PPV_POS_TOL_CM=5000      (~50 m de tolerancia)
# PPV_DIVERGE_POLLS=4      (polls seguidos divergindo p/ agir)
# PPV_REST_POLL_MS=2000    (intervalo do poll da REST)
# PPV_BAN=0                (1 = politica B: derruba + bloqueia no nivel da voz)
```

O `verify` **nunca** pune com REST stale/laggada e só age em spoof **sustentado**
(N polls seguidos). Fast-travel/montaria se resolve sozinho.

---

## Pré-requisitos

1. **REST API do Palworld ligada** no `PalWorldSettings.ini`:
   ```ini
   RESTAPIEnabled=True
   RESTAPIPort=8212
   AdminPassword="<senha forte>"
   ```
   A REST é **API de admin** — **nunca** exponha pública. O voz fala com ela
   **por dentro** da rede docker.

2. **Voz e Palworld na MESMA rede docker.** O container de voz precisa resolver
   `palworld-server` (o nome do service do Palworld). Se rodam em compose/projetos
   separados (ex.: Dokploy), use uma rede externa:
   ```bash
   docker network create palprox
   ```
   Nos **dois** compose:
   ```yaml
   services:
     voice:        # (ou palworld-server)
       networks: [palprox]
   networks:
     palprox:
       external: true
   ```
   Confirme que o `PPV_REST_URL` usa o **nome do service** do Palworld (não
   `localhost`, não IP do host).

---

## Passo a passo

1. **Configure as env vars** (`.env` ou `environment:` no compose) com o bloco do TL;DR.
2. **Suba o voz:** `docker compose up -d --build voice`
3. **Confirme nos logs** que ligou:
   ```
   anti-spoof: verify  rest=http://palworld-server:8212 poll=2000ms tol=5000cm diverge=4 ban=false
   ```
   - Se vir `anti-spoof: ... pedido mas PPV_REST_URL vazio -> caindo pra OFF`
     → faltou `PPV_REST_URL`.
   - Se vir `anti-spoof: OFF` → `PPV_AUTH_MODE` não está `verify`.
4. **Valide a conectividade com a REST** (de dentro do container de voz):
   ```bash
   docker exec -it palproxvoice sh -c \
     'wget -qO- --user=admin --password=$PPV_REST_PASS http://palworld-server:8212/v1/api/players'
   ```
   Tem que voltar JSON com `players[]` (cada um com `userId`, `playerId`, `ip`,
   `location_x/y`). Se der timeout → rede docker (passo 2 dos pré-requisitos).

---

## Como testar com spoof real

1. Entre no jogo com o mod (posição honesta). Confirme que a voz proximal
   funciona normal (`PPV_AUTH_MODE=off` x `verify` deve soar igual quando honesto —
   o `verify` repassa a posição do cliente enquanto ela bate com a REST).
2. **Forçe um spoof:** edite o `main.lua`/`palproxvoice_pos.txt` pra reportar uma
   posição falsa longe da real (> tolerância, default 50 m), mantendo a posição
   real no jogo.
3. **Espere `PPV_DIVERGE_POLLS` polls** (default 4 × 2 s ≈ 8 s). A partir daí:
   - **Política A (padrão):** o voz **ignora a mentira** — repassa a última
     posição válida (ou a da REST). O trapaceiro **não** consegue se
     "teleportar" pros teus ouvidos. Sem log (silencioso, não-punitivo).
   - **Política B (`PPV_BAN=1`):** além de ignorar, **derruba e bloqueia**
     (userId+ip) no nível da voz. Log esperado:
     ```
     anti-spoof: SPOOF de <peerID> (ip=<ip> user="<userId>") -> bloqueado (politica B)
     ```
     (bloqueio é em memória, **não** bane do jogo; reinício do voz limpa.)
4. **Volte a posição honesta** → na política A, em ≤ tolerância o `divergeN` zera e
   volta a repassar o do cliente (baixa latência).

### Caso "mesma casa" (mesmo IP, namorada/2 PCs)
Sem `userId` do cliente, o voz correlaciona por **IP + proximidade**: com vários
jogadores no mesmo IP, casa pelo **mais próximo** da posição reportada e o
`verify` mantém honesto dali pra frente. Cobre o caso razoavelmente **sem** o
probe do mod. O `userId`/`playerId` (campo `user` no auth) é robustez extra —
adicione depois que o probe do mod disser como ler (ver
`mod/PalProxVoice/scripts/probe_id.lua`).

---

## Notas de comportamento (do `antispoof.go`)

- **`off`** → repassa posição do cliente (legado).
- **`verify`** → repassa a do cliente; só troca quando diverge **sustentado**.
- **`strict`** → posição **sempre** da REST (yaw continua do cliente; a REST não
  tem direção de câmera).
- **Nunca pune** quando: REST stale (`restAt` velho > 4× poll), erro/timeout da
  REST, formato de posição inesperado, ou a REST ainda não "vê" o peer (acabou de
  entrar / trocando de mundo).
- `location_x/y` da REST está na **mesma escala (cm)** que o mod escreve →
  reconciliação é distância 2D pura, sem conversão.

## Rollback
`PPV_AUTH_MODE=off` (ou remover a var) + `docker compose up -d voice`. Volta ao
comportamento legado na hora, sem rebuild.
