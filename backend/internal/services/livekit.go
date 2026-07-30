package services

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/instantmeet/instantmeet/backend/internal/config"
	"github.com/instantmeet/instantmeet/backend/internal/models"
	lkauth "github.com/livekit/protocol/auth"
	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"
)

const (
	deleteRoomTimeout  = 5 * time.Second
	deleteRoomAttempts = 2
)

type LiveKit struct {
	cfg   config.Config
	rooms *lksdk.RoomServiceClient
}

func NewLiveKit(cfg config.Config) *LiveKit {
	host := httpHost(cfg.LiveKitURL)
	return &LiveKit{
		cfg:   cfg,
		rooms: lksdk.NewRoomServiceClient(host, cfg.LiveKitAPIKey, cfg.LiveKitAPISecret),
	}
}

func (l *LiveKit) Token(room string, user models.User, canPublish bool) (string, error) {
	canSubscribe := true
	grant := &lkauth.VideoGrant{RoomJoin: true, Room: room, CanPublish: &canPublish, CanSubscribe: &canSubscribe}
	return lkauth.NewAccessToken(l.cfg.LiveKitAPIKey, l.cfg.LiveKitAPISecret).
		SetIdentity(user.ID).SetName(user.DisplayName).SetValidFor(6 * time.Hour).AddGrant(grant).ToJWT()
}

func (l *LiveKit) PublicURL() string { return l.cfg.LiveKitPublicURL }

// DeleteRoom best-effort removes the SFU room. The caller's context is ignored so
// client disconnect cannot cancel cleanup. Uses a short timeout and one retry.
// Missing rooms are treated as success.
func (l *LiveKit) DeleteRoom(_ context.Context, room string) error {
	if room == "" || l.rooms == nil {
		return nil
	}
	var err error
	for attempt := 1; attempt <= deleteRoomAttempts; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), deleteRoomTimeout)
		_, err = l.rooms.DeleteRoom(ctx, &livekit.DeleteRoomRequest{Room: room})
		cancel()
		if err == nil {
			return nil
		}
		if isRoomMissing(err) {
			slog.Debug("livekit room already gone", "room", room)
			return nil
		}
		slog.Warn("livekit room delete failed",
			"room", room,
			"attempt", attempt,
			"attempts", deleteRoomAttempts,
			"error", err,
		)
	}
	return err
}

func httpHost(url string) string {
	switch {
	case strings.HasPrefix(url, "wss://"):
		return "https://" + strings.TrimPrefix(url, "wss://")
	case strings.HasPrefix(url, "ws://"):
		return "http://" + strings.TrimPrefix(url, "ws://")
	default:
		return url
	}
}

func isRoomMissing(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found") ||
		strings.Contains(msg, "does not exist") ||
		strings.Contains(msg, "twirp error not_found")
}
