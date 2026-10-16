package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"kitchen-api/internal/domain"
)

type fakeMenuItemRepo struct {
	byVenue map[uuid.UUID][]domain.MenuItem
	synced  []domain.MenuItem
}

func (f *fakeMenuItemRepo) ListByVenue(_ context.Context, venueID uuid.UUID) ([]domain.MenuItem, error) {
	return f.byVenue[venueID], nil
}

func (f *fakeMenuItemRepo) Upsert(_ context.Context, item domain.MenuItem) error {
	f.synced = append(f.synced, item)
	return nil
}

func TestMenuService_ListByVenue(t *testing.T) {
	venueID := uuid.New()
	repo := &fakeMenuItemRepo{byVenue: map[uuid.UUID][]domain.MenuItem{
		venueID: {{ID: uuid.New(), Name: "Пицца Маргарита", IsAvailable: true}},
	}}
	svc := NewMenuService(repo)

	got, err := svc.ListByVenue(context.Background(), venueID)
	if err != nil {
		t.Fatalf("ListByVenue() error = %v", err)
	}
	if len(got) != 1 || got[0].Name != "Пицца Маргарита" {
		t.Errorf("ListByVenue() = %+v", got)
	}
}

func TestMenuService_Sync(t *testing.T) {
	repo := &fakeMenuItemRepo{byVenue: map[uuid.UUID][]domain.MenuItem{}}
	svc := NewMenuService(repo)

	removedAt := time.Now()
	item := domain.MenuItem{ID: uuid.New(), Name: "Лазанья", IsAvailable: false, RemovedAt: &removedAt}
	if err := svc.Sync(context.Background(), item); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if len(repo.synced) != 1 || repo.synced[0].Name != "Лазанья" {
		t.Errorf("Sync() did not upsert item, got %+v", repo.synced)
	}
}
