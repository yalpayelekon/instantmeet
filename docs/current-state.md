# Current state

**Assessment date:** 2026-07-29 (updated after Phase 1)
**Verdict:** InstantMeet is a tested single-node MVP for ephemeral video meetings. Phase 0 polish and Phase 1 quality gates are complete. The next focus is production hardening, followed by multi-instance scale only if needed.

| Lens | Estimate | Meaning |
|------|----------|---------|
| Stated MVP scope | **~94%** | Primary two-user product loop is covered end to end |
| Production-ready self-host | **~72%** | CI and smoke coverage exist; production ops hardening remains |
| Blended | **~83%** | Strong tested core; next focus is Phase 2 deploy hardening |

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

### Meeting lifecycle

- One-click create with human-friendly room IDs
- Join flow; host auto-admitted; others enter waiting room
- Admit / reject waiting participants
- 100-person admission cap
- Leave / end meeting with in-memory teardown
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

- Docker Compose: Postgres, Redis, LiveKit, Nginx, Go API, React
- Distroless backend image; Nginx-served frontend
- `/healthz`
- Graceful Go shutdown
- Backend unit tests
- Frontend unit tests for auth, API, and meeting WebSocket behavior
- Playwright two-user create → admit → chat → teardown smoke test
- Unified backend, frontend, and e2e GitHub Actions quality workflow
- Lazy-loaded meeting route; initial JavaScript reduced from ~856 kB to ~240 kB
- Architecture, API, and [deploy](deploy.md) docs

## Partial / unfinished

| Item | Status | Notes |
|------|--------|--------|
| Meeting `Secret` field | Unused | Generated on create; not used for join/auth |
| Meeting state enum | Overspecified | `created` / `destroyed` exist; runtime mainly `waiting` → `active` → delete |
| Redis | Provisioned only | Documented for future multi-instance; unused by app today |
| Meeting media chunk | Large but deferred | LiveKit is isolated in the lazy meeting route; home does not download it |

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
4. **Production ops:** TURN/TLS are documented in [deploy.md](deploy.md) but still operator-owned
5. **Observability:** Health check and structured request logs only; no metrics or tracing
6. **Unused secret:** Room secret suggests a future private-link model that is not implemented

## Test and CI signal

| Area | Signal |
|------|--------|
| Backend `go test ./...` | Present (api, auth, meeting, websocket, db) |
| CI (`.github/workflows/quality.yml`) | Backend build/test, frontend lint/build/unit, then e2e |
| Frontend tests | Vitest: API client, auth hook, meeting socket |
| Frontend CI | `npm ci`, lint, production build, unit tests |
| E2E | Playwright Chromium: two authenticated contexts through teardown |

## Deployment readiness

**Ready for:** local demos, single-host self-hosting with careful TLS/TURN/LiveKit setup (see [deploy.md](deploy.md)).

**Not ready for without more work:** multi-instance API, zero-ops cloud SaaS, compliance-heavy deployments, or “Zoom-class” feature parity.

See [roadmap.md](roadmap.md) for prioritized next work (Phase 2: self-host production hardening).
