# InstantMeet

InstantMeet is an open-source, self-hostable video meeting app for one-click,
ephemeral conversations. It has no meeting history, recordings, scheduling,
uploads, subscriptions, or artificial time limit.

## Live demo

Try InstantMeet at [toplanti.online](https://toplanti.online).

> The public demo URL may change. InstantMeet remains fully self-hostable, and
> public demo availability and capacity may vary during testing.

## What works

- Google OAuth 2.0 authentication and secure JWT cookies
- User accounts persisted in PostgreSQL (meetings are never stored)
- One-click meeting creation with shareable human-friendly IDs
- Host-controlled waiting room (admit/reject)
- LiveKit audio, video, and screen-share transport
- Ephemeral in-meeting chat
- Host mute, remove, and end-meeting controls
- 100-person admission limit
- Automatic in-memory teardown when the host ends or the last person leaves
- Membership-gated, push-only WebSocket meeting events
- Graceful Go server shutdown
- Docker Compose stack with PostgreSQL, Redis, LiveKit, edge proxy, Go, React,
  Prometheus, and Grafana (Nginx locally; Caddy + ACME TLS in production)

PostgreSQL stores user accounts only. Meeting state, participants, and chat are
kept in process memory and permanently discarded when a meeting ends. Redis is
included for the documented infrastructure and future multi-instance presence;
the single-node MVP deliberately does not persist meeting state there.

WebSocket connections require a valid session cookie and that the user is already
in the meeting (participant or waiting room). Clients cannot inject signaling
events over the socket; the API is the only source of meeting broadcasts.

## Quick start

1. Copy `.env.example` to `.env`.
2. Create a Google OAuth web client and fill in `GOOGLE_CLIENT_ID` and
   `GOOGLE_CLIENT_SECRET`. Add `http://localhost/api/auth/google/callback` as an
   authorized redirect URI.
3. Replace `JWT_SECRET` with a random value of at least 32 characters.
4. Run `docker compose up --build`.
   If you previously started Compose with the old UUID `users` schema, recreate
   the Postgres volume once (`docker compose down -v`) so `init.sql` applies.
5. Open [http://localhost](http://localhost).

For local UI work without Google credentials, set `DEV_AUTH_ENABLED=true` in
`.env`. The homepage exposes a demo-login link only in Vite development mode
(`npm run dev`). Production builds never show that link. Leave
`DEV_AUTH_ENABLED=false` for any shared or public deployment.

## Local development

Run infrastructure:

```sh
docker compose up postgres redis livekit
```

Run the API (set `FRONTEND_URL=http://localhost:5173` so WebSocket origin checks
match Vite):

```sh
cd backend
go mod download
set FRONTEND_URL=http://localhost:5173
go run ./cmd/server
```

Run the frontend:

```sh
cd frontend
npm install
npm run dev
```

The Vite server proxies API and WebSocket traffic to port 8080.

## Production notes

See **[docs/deploy.md](docs/deploy.md)** for the full operator checklist. Summary:

- Use `docker-compose.prod.yml` + Caddy for public hosts (automatic HTTPS).
- Set Google OAuth credentials before `scripts/deploy.sh` (required).
- LiveKit public URL is `wss://<LIVEKIT_DOMAIN>`; embedded TURN listens on UDP 3478.
- Run one backend instance for this MVP. Horizontal scaling requires moving
  presence/signaling coordination to Redis.
- Confirm `/healthz` returns JSON `{"status":"ok"}` and run
  `bash scripts/smoke-production.sh` after deploy.

See [docs/architecture.md](docs/architecture.md) and
[docs/api.md](docs/api.md) for design and endpoint details.

## Production deployment

The repository includes a hardened single-server deployment:

1. Point the root, `www`, and `livekit` DNS records to the server IPv4 address.
2. Clone the repository onto an Ubuntu server.
3. Run `sudo bash scripts/bootstrap-ubuntu.sh`.
4. Copy `.env.production.example` to `.env.production` (or run
   `scripts/init-production-env.sh`) and fill in fresh secrets **plus** Google
   OAuth credentials. Deploy refuses to start without Google.
5. Run `bash scripts/deploy.sh`.
6. Run `bash scripts/smoke-production.sh`, then the manual two-user checklist in
   [docs/deploy.md](docs/deploy.md).

Caddy obtains and renews public TLS certificates automatically. The production
firewall exposes HTTPS, LiveKit's ICE/TCP fallback, TURN/UDP, and the configured
WebRTC UDP range. PostgreSQL, Redis, application APIs, and LiveKit signaling
remain private to the Docker network.
Status and planning live in [docs/current-state.md](docs/current-state.md) and
[docs/roadmap.md](docs/roadmap.md).
Deploy checklist: [docs/deploy.md](docs/deploy.md).

## License

MIT — see [LICENSE](LICENSE).
