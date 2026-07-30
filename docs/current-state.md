# Current state

**Assessment date:** 2026-07-30 (updated after Phase 2 hardening)
**Verdict:** InstantMeet is a tested single-node MVP with production Compose (Caddy/TLS/TURN), OAuth deploy gates, LiveKit room cleanup, and basic request metrics. Remaining work is operator Google OAuth on the live host and optional multi-instance scale.

| Lens | Estimate | Meaning |
|------|----------|---------|
| Stated MVP scope | **~95%** | Primary two-user product loop is covered end to end |
| Production-ready self-host | **~88%** | Prod stack + docs + cleanup/metrics; host must set Google OAuth |
| Blended | **~91%** | Strong tested core; live sign-in blocked until Google credentials are set |

## Product intent (in scope)

InstantMeet aims to be a self-hostable, one-click ephemeral meeting app:

- Google sign-in
- Instant room creation and join by friendly ID
- Host waiting room and moderation
- LiveKit A/V and screen share
- Ephemeral chat
- No meeting history, recordings, scheduling, uploads, or paywalls

Anything outside that list is explicitly out of product scope unless called out below as hardening or polish for the MVP itself.

## What is complete

### Authentication and accounts

- Google OAuth 2.0 login and callback
- JWT session cookies with auth middleware
- Optional `DEV_AUTH_ENABLED` demo login (dev UI only)
- Users upserted into PostgreSQL (`users` table only)
- Production boot refuses empty Google credentials / default secrets

### Meeting lifecycle

- One-click create with human-friendly room IDs
- Join flow; host auto-admitted; others enter waiting room
- Admit / reject waiting participants
- 100-person admission cap
- Leave / end meeting with in-memory teardown
- Best-effort LiveKit `DeleteRoom` on end / last leave
- Host mute, remove participant, end meeting

### Media and realtime

- LiveKit token minting after admission
- Audio, video, and screen share in the meeting UI
- Pre-join local A/V preview with device selection
- In-call Settings panel (mic/camera/speaker)
- Session-scoped device prefs (survive waiting → admitted)
- Membership-gated, push-only WebSocket events with reconnect backoff
- LiveKit reconnect / media recovery without forced home redirect
- Origin-checked WebSocket connections
- Ephemeral in-meeting chat (1–1000 chars)

### Frontend surfaces

- Home: login, create meeting, join by code/link
- Waiting room UI with device setup
- Pre-join “Join now” gate before LiveKit connect
- In-call people, chat, and settings panels
- Invite link copy
- Host controls in the people panel
- Connection status banner

### Ops / packaging

- Local Docker Compose: Postgres, Redis, LiveKit, Nginx, Go API, React, Prometheus, Grafana
- Production Compose: Caddy TLS, embedded LiveKit TURN, hardened env, Prometheus + Grafana (localhost-only)
- Distroless backend image; Nginx-served frontend
- `/healthz` (public); `/metrics` scraped by Prometheus on the Docker network (not edged)
- Prometheus retention capped at 15d / 5GB; Grafana dashboard **InstantMeet Overview**
- Structured request logs with status and latency
- Low-cardinality HTTP + meeting/WebSocket gauges for ops dashboards
- Graceful Go shutdown
- Backend unit tests
- Frontend unit tests for auth, API, and meeting WebSocket behavior
- Playwright two-user create → admit → chat → teardown smoke test
- Unified backend, frontend, and e2e GitHub Actions quality workflow
- `scripts/deploy.sh` + `scripts/smoke-production.sh`
- Lazy-loaded meeting route; initial JavaScript reduced from ~856 kB to ~240 kB
- Architecture, API, and [deploy](deploy.md) docs aligned with Caddy/prod

## Partial / unfinished

| Item | Status | Notes |
|------|--------|--------|
| Meeting state enum | Overspecified | `created` / `destroyed` exist; runtime mainly `waiting` → `active` → delete |
| Redis | Provisioned only | Documented for future multi-instance; unused by app today |
| Meeting media chunk | Large but deferred | LiveKit is isolated in the lazy meeting route; home does not download it |
| Live Google OAuth on toplanti.online | Operator pending | API returns 503 until `GOOGLE_*` are set and redeployed |

## Intentionally absent (product scope)

These are **not** gaps relative to the README thesis:

- Scheduling / calendar
- Meeting history or past rooms
- Recordings
- File uploads
- Subscriptions / payments
- Artificial time limits

## Gaps that still matter for the MVP

1. **Auth breadth:** Google-only (plus local demo); no email/passwordless path for self-hosters without Google
2. **Meeting UX polish:** No noise/background options, reactions, raise hand, or keyboard shortcuts
3. **Single-node ceiling:** Cannot run multiple API replicas without Redis-backed store + WS pub/sub
4. **Observability depth:** Counters + structured logs only; no distributed tracing
5. **Host Google OAuth:** Production sign-in requires operator credentials in `.env.production`

## Test and CI signal

| Area | Signal |
|------|--------|
| Backend `go test ./...` | Present (api, auth, meeting, websocket, db, config, livekit) |
| CI (`.github/workflows/quality.yml`) | Backend build/test, frontend lint/build/unit, then e2e |
| Frontend tests | Vitest: API client, auth hook, meeting socket |
| Frontend CI | `npm ci`, lint, production build, unit tests |
| E2E | Playwright Chromium: two authenticated contexts through teardown |
| Prod smoke script | `scripts/smoke-production.sh` checks healthz and Google redirect |

## Deployment readiness

**Ready for:** local demos and single-host public deploy once Google OAuth is configured (see [deploy.md](deploy.md)).

**Not ready for without more work:** multi-instance API, zero-ops cloud SaaS, compliance-heavy deployments, or “Zoom-class” feature parity.

See [roadmap.md](roadmap.md) for prioritized next work (Phase 2 mostly complete; Phase 3 only if multi-instance is needed).
