# HTTP and WebSocket API

All `/api` endpoints except login and callback require the HttpOnly JWT cookie.

| Method | Path | Purpose |
|---|---|---|
| GET/POST | `/api/login/google` | Start Google OAuth |
| GET | `/api/auth/google/callback` | Finish OAuth |
| POST | `/api/logout` | Clear session |
| GET | `/api/me` | Current user |
| POST | `/api/meetings` | Create meeting |
| GET | `/api/meetings/{id}` | Read public meeting state |
| POST | `/api/meetings/{id}/join` | Request entry / enter as host |
| POST | `/api/meetings/{id}/leave` | Leave |
| POST | `/api/meetings/{id}/admit` | Admit waiting user (host) |
| POST | `/api/meetings/{id}/reject` | Reject waiting user (host) |
| POST | `/api/meetings/{id}/mute` | Request participant mute (host) |
| POST | `/api/meetings/{id}/remove` | Remove participant (host) |
| POST | `/api/meetings/{id}/chat` | Send ephemeral chat |
| POST | `/api/meetings/{id}/media` | Update media state |
| POST | `/api/meetings/{id}/end` | End and destroy meeting (host) |
| GET | `/ws?meetingId={id}` | Meeting event stream |

WebSocket event types include `meeting.updated`, `meeting.ended`,
`participant.admitted`, `participant.rejected`, `participant.removed`,
`participant.muted`, `participant.media`, and `chat.message`.

