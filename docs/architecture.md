# Architecture

## Request path

Browser → Nginx → React static app or Go API → in-memory meeting store

Live media bypasses the API and travels through LiveKit over WebRTC. The API
issues short-lived room grants only after admission. WebSockets distribute
waiting-room, participant, chat, media-state, and termination events.

WebSockets are **server → client only**. The browser authenticates with the
session cookie, must already be in the meeting's participant list or waiting
room, and must present an `Origin` matching `FRONTEND_URL`. Inbound client
payloads are discarded; forged `chat.message` / `meeting.ended` events cannot
be injected over the socket. All mutations go through authenticated HTTP APIs,
which then call `Hub.Broadcast`.

## Privacy and lifecycle

Meeting state moves through `waiting → active → ending → destroyed`.
`created` is intentionally transient during creation. The meeting store owns
participants, the waiting room, and chat. `Delete` clears each collection
before removing the meeting entry. No meeting, participant, or message table
exists in PostgreSQL. Google (and optional demo) logins upsert rows into the
`users` table only.

The room is destroyed when:

- the host invokes `POST /api/meetings/{id}/end`; or
- the last active participant invokes `leave`.

LiveKit also has short empty/departure timeouts as a second cleanup layer.

## Scaling boundary

The MVP is intentionally single-process. A production 100-participant call
depends primarily on LiveKit sizing and network capacity, not the Go API.
Horizontal API scaling would require a Redis-backed meeting store, distributed
WebSocket pub/sub, and idempotent LiveKit room cleanup. Redis is present in
Compose for that future path and is unused by the single-node app today.
