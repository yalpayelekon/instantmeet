# Deploy checklist

Operator guide for a **single-node** InstantMeet deployment. The MVP keeps
meeting state in one Go process; do not run multiple API replicas until Redis
backing lands (see [architecture.md](architecture.md) and [roadmap.md](roadmap.md)).

## Before you expose the stack

1. Copy `.env.example` to `.env` and fill real values.
2. Set `JWT_SECRET` to a random string of at least 32 characters.
3. Configure Google OAuth (`GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`) and add the
   production callback URL (`https://<your-host>/api/auth/google/callback`).
4. Replace example LiveKit API key/secret (`LIVEKIT_API_KEY`, `LIVEKIT_API_SECRET`).
5. Set `FRONTEND_URL` to the public site origin (scheme + host, no trailing slash).
6. Set `LIVEKIT_PUBLIC_URL` to the **browser-reachable** LiveKit WebSocket URL
   (`wss://…`). Local Docker defaults are not valid on the public internet.
7. Leave `DEV_AUTH_ENABLED=false` on any shared or public host.
8. Terminate **TLS** at Nginx or a load balancer. Camera/microphone access
   requires HTTPS outside `localhost`.
9. Provide a **TURN** server for restrictive NATs (corporate firewalls, some
   mobile carriers). Without TURN, some guests will fail to establish WebRTC.
10. Confirm `/healthz` returns `{"status":"ok"}` behind the reverse proxy.

## Suggested Compose launch

```sh
docker compose up --build -d
```

Services: Nginx (static + proxy), Go API, PostgreSQL (users only), Redis
(unused by the app today), LiveKit.

## TLS

- Prefer terminating TLS at Nginx or a cloud load balancer in front of Compose.
- Forward `X-Forwarded-Proto` / `X-Forwarded-For` if your proxy strips them so
  cookies and redirects stay correct.
- WebSocket upgrade must pass through for `/ws` (see `docker/nginx.conf`).

## LiveKit and media

- Browsers connect directly to LiveKit using `LIVEKIT_PUBLIC_URL`.
- The API only mints short-lived join tokens after admission.
- Size LiveKit CPU/bandwidth for your expected concurrent publishers (up to the
  100-person admission cap).
- Empty-room / departure timeouts on LiveKit are a second cleanup layer after
  API teardown.

## TURN (recommended for public deploys)

Run coturn (or a managed TURN service) and point LiveKit at it via LiveKit’s
ICE/TURN configuration. Without TURN, WebRTC often fails when both peers are
behind symmetric NAT.

Document the TURN host, ports (UDP/TCP 3478, TLS 5349 as applicable), and
credentials in your private ops notes—do not commit secrets.

## Secrets rotation

Rotate before any public exposure:

- `JWT_SECRET`
- LiveKit API key/secret
- Google OAuth client secret
- Postgres password in `DATABASE_URL`

After rotating JWT, all existing sessions are invalidated (users must sign in
again).

## Single-node limit

Run **one** backend instance for this MVP. Horizontal API scaling requires:

- Redis-backed meeting store
- Distributed WebSocket pub/sub
- Idempotent LiveKit room cleanup

Redis is already in Compose for that future path and is unused by the app today.

## Smoke test after deploy

1. Open `https://<host>` over HTTPS.
2. Sign in with Google.
3. Create a meeting, pass pre-join device check, enter the room.
4. Join from a second browser/network; admit from the host people panel.
5. Send chat, toggle mic/camera, open Settings and switch devices.
6. End the meeting; confirm both clients return home and a new join fails for
   the old room id.
