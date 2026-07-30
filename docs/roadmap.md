# Roadmap

Prioritized plan from the [current state](current-state.md) assessment (2026-07-30).
Phases are sequential where noted; items inside a phase can often ship in parallel.

Out of scope unless the product thesis changes: meeting history, recordings,
scheduling, uploads, and subscriptions (see README).

---

## Phase 0 — Close MVP polish ✅

Finished 2026-07-29. Shipped experience no longer has dead UI chrome.

| Item | Status |
|------|--------|
| Settings wired to device panel | Done |
| Device selection (mic/camera/speaker) | Done |
| Pre-join preview | Done |
| WS + LiveKit reconnect UX | Done |
| `docs/deploy.md` operator checklist | Done |

**Exit criteria met:** No inert chrome; guests configure devices before connect; reconnect banners; deploy docs for TLS/TURN.

---

## Phase 1 — Quality and confidence ✅

Finished 2026-07-29. Frontend unit tests, a two-user browser smoke test, complete CI gates, and route-level code splitting are in place.

| Item | Status |
|------|--------|
| Frontend unit tests for auth hook, API client, meeting socket | Done |
| Playwright smoke: authenticate → create → admit → chat → end | Done |
| Backend + frontend + e2e CI gates | Done |
| Lazy-load LiveKit meeting route | Done |

**Exit criteria met:** CI fails on a broken API, frontend contract, build, or primary UI path.

---

## Phase 2 — Self-host production hardening ✅

Finished 2026-07-30. Production Compose path, OAuth/secret gates, LiveKit cleanup,
metrics, and aligned deploy docs are in the repo. Live host still needs Google
credentials filled by the operator.

| Item | Status |
|------|--------|
| Example / prod TLS reverse-proxy (Caddy) documented | Done |
| Documented TURN (LiveKit embedded in prod Compose) | Done |
| Structured logging + request metrics | Done |
| LiveKit room cleanup on meeting end/leave | Done |
| Remove unused meeting `Secret` | Done |
| Secret rotation / production env template + deploy validation | Done |
| `/healthz` edged publicly; `/metrics` internal-only; prod smoke script | Done |

**Exit criteria:** A stranger can follow [deploy.md](deploy.md) and run a single-node public instance safely once Google OAuth is configured.

**Operator follow-up on toplanti.online:** set `GOOGLE_CLIENT_ID` /
`GOOGLE_CLIENT_SECRET`, redeploy, run `scripts/smoke-production.sh`, then the
manual two-user checklist.

---

## Phase 3 — Scale beyond one API process

Only needed if you want multiple backend replicas or HA.

| Item | Why | Effort |
|------|-----|--------|
| Redis-backed meeting store | Shared presence across instances | L |
| Redis pub/sub (or equivalent) for WebSocket fan-out | Cross-node event delivery | L |
| Idempotent LiveKit room delete under concurrent end/leave | Already best-effort; harden for multi-writer | M |
| Sticky sessions *or* fully shared hub design | Avoid split-brain membership | M |

**Exit criteria:** Two API replicas behind the edge proxy serve the same meeting correctly.

---

## Phase 4 — Optional product expansions

Still compatible with “ephemeral,” but not required for the thesis:

| Item | Notes |
|------|-------|
| Additional IdPs (OIDC/SAML, GitHub, email magic link) | Better for enterprise self-host |
| Raise hand / reactions | Low-persistence UX; keep ephemeral |
| Host lock meeting / lobby messages | Moderated rooms without history |
| Password or secret-link join | New invite-token design (old unused Secret removed) |
| Mobile-friendly layout pass | Responsive polish, not native apps |
| Captions via LiveKit / browser APIs | Accessibility |

Defer indefinitely unless strategy changes: calendar sync, CRM, billing, recording pipelines, breakout rooms, admin analytics dashboards.

---

Suggested sequencing:

```text
Phase 0 ✅ ──► Phase 1 ✅ ──► Phase 2 ✅ ──► operator Google OAuth on live host
                                         │
                                         └──► Phase 3 (Redis HA) if multi-instance needed
                                         └──► Phase 4 features as demand appears
```

## Progress tracking

Update this file when a phase exits. Keep [current-state.md](current-state.md) in sync (estimates + tables) when major milestones land.
