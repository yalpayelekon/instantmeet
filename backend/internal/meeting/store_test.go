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
	if _, ok := s.Get(m.ID); !ok {
		t.Fatal("meeting was not stored")
	}
	s.Delete(m.ID)
	if _, ok := s.Get(m.ID); ok {
		t.Fatal("meeting was not deleted")
	}
}
