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
