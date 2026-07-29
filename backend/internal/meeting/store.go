package meeting

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"sync"
	"time"

	"github.com/instantmeet/instantmeet/backend/internal/models"
)

var ErrNotFound = errors.New("meeting not found")

type Store struct {
	mu       sync.RWMutex
	meetings map[string]*models.Meeting
}

func NewStore() *Store { return &Store{meetings: make(map[string]*models.Meeting)} }

func (s *Store) Create(host models.User) *models.Meeting {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := friendlyID()
	m := &models.Meeting{
		ID: id, Secret: token(24), HostID: host.ID, CreatedAt: time.Now().UTC(),
		Participants: map[string]*models.Participant{}, WaitingRoom: map[string]*models.WaitingParticipant{},
		LiveKitRoom: "instantmeet-" + id, Chat: []models.ChatMessage{}, State: models.MeetingWaiting,
	}
	s.meetings[id] = m
	return clone(m)
}

func (s *Store) Get(id string) (*models.Meeting, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m, ok := s.meetings[id]
	if !ok {
		return nil, false
	}
	return clone(m), true
}

func (s *Store) Update(id string, fn func(*models.Meeting) error) (*models.Meeting, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, ok := s.meetings[id]
	if !ok {
		return nil, ErrNotFound
	}
	if err := fn(m); err != nil {
		return nil, err
	}
	return clone(m), nil
}

func (s *Store) Delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if m := s.meetings[id]; m != nil {
		m.State = models.MeetingDestroyed
		for key := range m.Participants {
			delete(m.Participants, key)
		}
		for key := range m.WaitingRoom {
			delete(m.WaitingRoom, key)
		}
		m.Chat = nil
		delete(s.meetings, id)
	}
}

func friendlyID() string {
	const chars = "abcdefghjkmnpqrstuvwxyz23456789"
	b := make([]byte, 9)
	_, _ = rand.Read(b)
	for i := range b {
		b[i] = chars[int(b[i])%len(chars)]
	}
	return string(b[0:3]) + "-" + string(b[3:6]) + "-" + string(b[6:9])
}

func token(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func clone(m *models.Meeting) *models.Meeting {
	c := *m
	c.Participants = make(map[string]*models.Participant, len(m.Participants))
	for k, v := range m.Participants {
		p := *v
		c.Participants[k] = &p
	}
	c.WaitingRoom = make(map[string]*models.WaitingParticipant, len(m.WaitingRoom))
	for k, v := range m.WaitingRoom {
		p := *v
		c.WaitingRoom[k] = &p
	}
	c.Chat = make([]models.ChatMessage, len(m.Chat))
	copy(c.Chat, m.Chat)
	return &c
}
