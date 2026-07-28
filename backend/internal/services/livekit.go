package services

import (
	"time"

	"github.com/instantmeet/instantmeet/backend/internal/config"
	"github.com/instantmeet/instantmeet/backend/internal/models"
	lkauth "github.com/livekit/protocol/auth"
)

type LiveKit struct{ cfg config.Config }

func NewLiveKit(cfg config.Config) *LiveKit { return &LiveKit{cfg: cfg} }

func (l *LiveKit) Token(room string, user models.User, canPublish bool) (string, error) {
	canSubscribe := true
	grant := &lkauth.VideoGrant{RoomJoin: true, Room: room, CanPublish: &canPublish, CanSubscribe: &canSubscribe}
	return lkauth.NewAccessToken(l.cfg.LiveKitAPIKey, l.cfg.LiveKitAPISecret).
		SetIdentity(user.ID).SetName(user.DisplayName).SetValidFor(6 * time.Hour).AddGrant(grant).ToJWT()
}
func (l *LiveKit) PublicURL() string { return l.cfg.LiveKitPublicURL }
