# InstantMeet

InstantMeet is an open-source, self-hostable video meeting app for one-click,
ephemeral conversations. It has no meeting history, recordings, scheduling,
uploads, subscriptions, or artificial time limit.

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
- Docker Compose stack with PostgreSQL, Redis, LiveKit, Nginx, Go, and React

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

- Terminate TLS at Nginx or a cloud load balancer. WebRTC camera/microphone
  access requires HTTPS outside localhost.
- Set `LIVEKIT_PUBLIC_URL` to the browser-reachable `wss://` endpoint.
- Provide a TURN server for restrictive networks.
- Run one backend instance for this MVP. Horizontal scaling requires moving
  presence/signaling coordination to Redis.
- Rotate the example LiveKit credentials before exposing the stack.

See [docs/architecture.md](docs/architecture.md) and
[docs/api.md](docs/api.md) for design and endpoint details.

## License

MIT — see [LICENSE](LICENSE).
