# Deploy checklist

Operator guide for a **single-node** InstantMeet deployment. Meeting state lives
in one Go process; do not run multiple API replicas until Redis backing lands
(see [architecture.md](architecture.md) and [roadmap.md](roadmap.md)).

There are two Compose stacks:

| Stack | Compose file | Edge | Use |
|-------|--------------|------|-----|
| Local / CI | `docker-compose.yml` | Nginx (`docker/nginx.conf`) | Dev demos, Playwright |
| Production | `docker-compose.prod.yml` | Caddy (`docker/Caddyfile`) | Public host (TLS + TURN) |

## Production path (recommended)

### 1. DNS

Point these A records at the server IPv4 address:

- `toplanti.online` (or your `APP_DOMAIN`)
- `www.<APP_DOMAIN>`
- `livekit.<APP_DOMAIN>` (must match `LIVEKIT_DOMAIN`)

### 2. Bootstrap the host

On Ubuntu:

```sh
sudo bash scripts/bootstrap-ubuntu.sh
```

Opens UFW for HTTPS (`80`/`443`), LiveKit ICE/TCP (`7881`), TURN (`3478/udp`),
and the WebRTC UDP range (`40000–40100/udp`).

### 3. Create `.env.production`

```sh
bash scripts/init-production-env.sh you@example.com
```

Or copy [`.env.production.example`](../.env.production.example). The init script
fills random secrets and leaves Google empty on purpose.

### 4. Configure Google OAuth (required)

1. Google Cloud Console → **APIs & Services** → **Credentials**.
2. Create an **OAuth 2.0 Client ID** of type **Web application**.
3. Authorized JavaScript origins: `https://<APP_DOMAIN>`
4. Authorized redirect URI: `https://<APP_DOMAIN>/api/auth/google/callback`
5. Set `GOOGLE_CLIENT_ID` and `GOOGLE_CLIENT_SECRET` in `.env.production`.

`scripts/deploy.sh` and the production API refuse to start without these values.
`APP_ENV=production` also requires a ≥32-character `JWT_SECRET`, non-default
LiveKit keys, HTTPS `FRONTEND_URL`, and `wss://` `LIVEKIT_PUBLIC_URL` (Compose
sets the last two from `APP_DOMAIN` / `LIVEKIT_DOMAIN`).

### 5. Deploy

```sh
bash scripts/deploy.sh
```

This validates env, builds images, and starts Postgres, Redis, LiveKit, the Go
API, the React frontend, Caddy, Prometheus, and Grafana. Caddy obtains and renews
ACME certificates using `ACME_EMAIL`.

### 6. Automated smoke (API surface)

```sh
bash scripts/smoke-production.sh
# or: bash scripts/smoke-production.sh https://toplanti.online
```

Checks `/healthz` and that Google login redirects (or fails clearly
if OAuth is still empty on an older deploy).

### 7. Manual two-user smoke

1. Open `https://<APP_DOMAIN>` over HTTPS.
2. Sign in with Google.
3. Create a meeting, pass pre-join device check, enter the room.
4. Join from a second browser/network; admit from the host people panel.
5. Send chat, toggle mic/camera, open Settings and switch devices.
6. End the meeting; confirm both clients return home and a new join fails for
   the old room id.

## What production Compose already provides

- **TLS:** Caddy terminates HTTPS for the app and LiveKit hostnames.
- **TURN:** LiveKit embedded TURN on UDP `3478` (`docker/livekit.prod.yaml`).
  Restrictive NATs still need a real two-network media check after deploy.
- **WebSockets:** Caddy proxies `/ws` to the API.
- **Health:** `/healthz` (JSON) is proxied publicly for uptime checks.
- **Metrics + Grafana:** Prometheus scrapes the API `/metrics` on the Docker
  network (not edged via Caddy). Retention is capped at **15 days / 5 GB**.
  Grafana listens on `127.0.0.1:3000` only — open an SSH tunnel, then browse
  `http://127.0.0.1:3000` (admin / `GRAFANA_ADMIN_PASSWORD`). Provisioned
  dashboard: **InstantMeet Overview**.
- **Dev login:** Forced off (`DEV_AUTH_ENABLED=false`).

## Monitoring access (production)

```sh
ssh -L 3000:127.0.0.1:3000 user@your-host
# open http://127.0.0.1:3000
```

Prometheus UI is not published on the host. Inspect raw metrics with:

```sh
docker compose --env-file .env.production -f docker-compose.prod.yml \
  exec backend wget -qO- http://127.0.0.1:8080/metrics
```

If you already have a `.env.production` from before monitoring landed, add
`GRAFANA_ADMIN_PASSWORD` (≥12 chars) before the next `deploy.sh`.

## Local stack (optional)

```sh
cp .env.example .env
# fill Google or set DEV_AUTH_ENABLED=true for local UI only
docker compose up --build -d
```

Services: Nginx, Go API, PostgreSQL, Redis, LiveKit, frontend, Prometheus,
Grafana (`http://127.0.0.1:3000`, password from `GRAFANA_ADMIN_PASSWORD` or
`admin`). Prefer this for development; use the production path above for any
public host.

## Secrets rotation

Rotate before any public exposure (or on compromise):

- `JWT_SECRET` (invalidates all sessions)
- LiveKit API key/secret
- Google OAuth client secret
- `POSTGRES_PASSWORD` / `DATABASE_URL`
- `GRAFANA_ADMIN_PASSWORD`

After changing LiveKit keys, recreate the LiveKit container so `LIVEKIT_KEYS`
matches the API.

## Single-node limit

Run **one** backend instance for this MVP. Horizontal API scaling requires:

- Redis-backed meeting store
- Distributed WebSocket pub/sub
- Idempotent LiveKit room cleanup (DeleteRoom is already best-effort)

Redis is in Compose for that future path and is unused by the app today.

## LiveKit cleanup

On host `end` or last-participant `leave`, the API best-effort deletes the
LiveKit room (`instantmeet-<id>`). LiveKit `empty_timeout` /
`departure_timeout` remain a second cleanup layer if the admin call fails.
