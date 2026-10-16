package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"kitchen-api/internal/domain"
)

func TestMenuItemRepository_ListByVenueAndGetAvailable(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	truncateAll(t, pool)

	venueID := uuid.New()
	_, err := pool.Exec(ctx, "INSERT INTO venues (id, name, api_base_url, is_active) VALUES ($1, 'Тест', 'http://venue-service:8081', true)", venueID)
	if err != nil {
		t.Fatalf("insert venue: %v", err)
	}

	availableID := uuid.New()
	unavailableID := uuid.New()
	_, err = pool.Exec(ctx, `
		INSERT INTO menu_items (id, venue_id, external_id, name, price, is_available) VALUES
		($1, $2, 'ext-1', 'Пицца', '450.00', true),
		($3, $2, 'ext-2', 'Тирамису', '280.00', false)
	`, availableID, venueID, unavailableID)
	if err != nil {
		t.Fatalf("insert menu items: %v", err)
	}

	repo := NewMenuItemRepository(pool)

	all, err := repo.ListByVenue(ctx, venueID)
	if err != nil {
		t.Fatalf("ListByVenue() error = %v", err)
	}
	if len(all) != 2 {
		t.Errorf("ListByVenue() = %d items, want 2", len(all))
	}

	available, err := repo.GetAvailableByIDs(ctx, venueID, []uuid.UUID{availableID, unavailableID})
	if err != nil {
		t.Fatalf("GetAvailableByIDs() error = %v", err)
	}
	if len(available) != 1 || available[0].ID != availableID {
		t.Errorf("GetAvailableByIDs() = %+v, want only availableID", available)
	}
}

func TestMenuItemRepository_Upsert(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	truncateAll(t, pool)

	venueID := uuid.New()
	_, err := pool.Exec(ctx, "INSERT INTO venues (id, name, api_base_url, is_active) VALUES ($1, 'Тест', 'http://venue-service:8081', true)", venueID)
	if err != nil {
		t.Fatalf("insert venue: %v", err)
	}

	repo := NewMenuItemRepository(pool)
	itemID := uuid.New()

	item := domain.MenuItem{ID: itemID, VenueID: venueID, Name: "Лазанья", Price: "470.00", IsAvailable: true}
	if err := repo.Upsert(ctx, item); err != nil {
		t.Fatalf("Upsert() create error = %v", err)
	}

	item.Price = "499.00"
	item.IsAvailable = false
	if err := repo.Upsert(ctx, item); err != nil {
		t.Fatalf("Upsert() update error = %v", err)
	}

	items, err := repo.ListByVenue(ctx, venueID)
	if err != nil {
		t.Fatalf("ListByVenue() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("ListByVenue() = %d items, want 1 (upsert should not duplicate)", len(items))
	}
	if items[0].Price != "499.00" || items[0].IsAvailable {
		t.Errorf("item after upsert = %+v, want price=499.00 isAvailable=false", items[0])
	}
}
