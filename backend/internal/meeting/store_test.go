package meeting

import (
	"github.com/instantmeet/instantmeet/backend/internal/models"
	"testing"
)

func TestCreateAndDelete(t *testing.T) {
	s := NewStore()
	m := s.Create(models.User{ID: "host"})
	if m.HostID != "host" || m.State != models.MeetingWaiting {
		t.Fatalf("unexpected meeting: %#v", m)
	}
	if m.Chat == nil {
		t.Fatal("created meeting chat must serialize as an empty array")
	}
	stored, ok := s.Get(m.ID)
	if !ok {
		t.Fatal("meeting was not stored")
	}
	if stored.Chat == nil {
		t.Fatal("cloned meeting chat must serialize as an empty array")
	}
	s.Delete(m.ID)
	if _, ok := s.Get(m.ID); ok {
		t.Fatal("meeting was not deleted")
	}
}

func TestStats(t *testing.T) {
	s := NewStore()
	m := s.Create(models.User{ID: "host"})
	_, err := s.Update(m.ID, func(meeting *models.Meeting) error {
		meeting.Participants["host"] = &models.Participant{UserID: "host", DisplayName: "Host"}
		meeting.WaitingRoom["guest"] = &models.WaitingParticipant{Participant: models.Participant{UserID: "guest", DisplayName: "Guest"}}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	meetings, participants, waiting := s.Stats()
	if meetings != 1 || participants != 1 || waiting != 1 {
		t.Fatalf("Stats() = %d,%d,%d", meetings, participants, waiting)
	}
}
