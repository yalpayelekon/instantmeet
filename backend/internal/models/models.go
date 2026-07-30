package models

import "time"

type User struct {
	ID          string `json:"id"`
	GoogleID    string `json:"-"`
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
	Avatar      string `json:"avatar"`
}

type Participant struct {
	UserID        string    `json:"userId"`
	DisplayName   string    `json:"displayName"`
	Avatar        string    `json:"avatar"`
	IsHost        bool      `json:"isHost"`
	JoinedAt      time.Time `json:"joinedAt"`
	MicEnabled    bool      `json:"micEnabled"`
	CameraEnabled bool      `json:"cameraEnabled"`
	ScreenSharing bool      `json:"screenSharing"`
}

type WaitingParticipant struct {
	Participant
	RequestedAt time.Time `json:"requestedAt"`
}

type ChatMessage struct {
	ID          string    `json:"id"`
	UserID      string    `json:"userId"`
	DisplayName string    `json:"displayName"`
	Text        string    `json:"text"`
	SentAt      time.Time `json:"sentAt"`
}

type MeetingState string

const (
	MeetingCreated   MeetingState = "created"
	MeetingWaiting   MeetingState = "waiting"
	MeetingActive    MeetingState = "active"
	MeetingEnding    MeetingState = "ending"
	MeetingDestroyed MeetingState = "destroyed"
)

type Meeting struct {
	ID           string                         `json:"id"`
	HostID       string                         `json:"hostId"`
	CreatedAt    time.Time                      `json:"createdAt"`
	Participants map[string]*Participant        `json:"participants"`
	WaitingRoom  map[string]*WaitingParticipant `json:"waitingRoom"`
	LiveKitRoom  string                         `json:"liveKitRoom"`
	Chat         []ChatMessage                  `json:"chat"`
	State        MeetingState                   `json:"state"`
}
