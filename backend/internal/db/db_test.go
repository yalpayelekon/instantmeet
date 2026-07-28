package db_test

import (
	"context"
	"errors"
	"testing"

	"github.com/instantmeet/instantmeet/backend/internal/db"
	"github.com/instantmeet/instantmeet/backend/internal/models"
)

type memoryUsers struct {
	byID map[string]models.User
}

func (m *memoryUsers) UpsertUser(_ context.Context, user models.User) error {
	if m.byID == nil {
		m.byID = map[string]models.User{}
	}
	m.byID[user.ID] = user
	return nil
}

func (m *memoryUsers) GetByID(_ context.Context, id string) (models.User, error) {
	u, ok := m.byID[id]
	if !ok {
		return models.User{}, errors.New("not found")
	}
	return u, nil
}

func TestUserRepositoryContract(t *testing.T) {
	var _ interface {
		UpsertUser(context.Context, models.User) error
		GetByID(context.Context, string) (models.User, error)
	} = (*db.Pool)(nil)

	repo := &memoryUsers{}
	user := models.User{ID: "sub-1", GoogleID: "sub-1", Email: "a@example.com", DisplayName: "A", Avatar: "x"}
	if err := repo.UpsertUser(context.Background(), user); err != nil {
		t.Fatal(err)
	}
	user.DisplayName = "Updated"
	if err := repo.UpsertUser(context.Background(), user); err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetByID(context.Background(), "sub-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.DisplayName != "Updated" || got.Email != "a@example.com" {
		t.Fatalf("got %#v", got)
	}
}
