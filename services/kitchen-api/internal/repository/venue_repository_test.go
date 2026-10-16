package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"kitchen-api/internal/domain"
)

func TestVenueRepository_ListAndGetByID(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	truncateAll(t, pool)

	catA := uuid.New()
	catB := uuid.New()
	_, err := pool.Exec(ctx, "INSERT INTO categories (id, name) VALUES ($1, 'Итальянская кухня'), ($2, 'Азиатская кухня')", catA, catB)
	if err != nil {
		t.Fatalf("insert categories: %v", err)
	}

	venueA := uuid.New()
	venueB := uuid.New()
	_, err = pool.Exec(ctx,
		"INSERT INTO venues (id, name, api_base_url, is_active) VALUES ($1, 'Траттория', 'http://venue-service:8081', true), ($2, 'Дом лапши', 'http://venue-service:8081', true)",
		venueA, venueB)
	if err != nil {
		t.Fatalf("insert venues: %v", err)
	}

	_, err = pool.Exec(ctx, "INSERT INTO venue_categories (venue_id, category_id) VALUES ($1, $2), ($3, $4)", venueA, catA, venueB, catB)
	if err != nil {
		t.Fatalf("insert venue_categories: %v", err)
	}

	repo := NewVenueRepository(pool)

	all, err := repo.List(ctx, nil)
	if err != nil {
		t.Fatalf("List(nil) error = %v", err)
	}
	if len(all) != 2 {
		t.Errorf("List(nil) = %d venues, want 2", len(all))
	}

	filtered, err := repo.List(ctx, &catA)
	if err != nil {
		t.Fatalf("List(catA) error = %v", err)
	}
	if len(filtered) != 1 || filtered[0].ID != venueA {
		t.Errorf("List(catA) = %+v, want only venueA", filtered)
	}
	if len(filtered[0].Categories) != 1 || filtered[0].Categories[0].ID != catA {
		t.Errorf("venueA categories = %+v, want [catA]", filtered[0].Categories)
	}

	got, err := repo.GetByID(ctx, venueA)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got.Name != "Траттория" {
		t.Errorf("GetByID() name = %q, want Траттория", got.Name)
	}

	if _, err := repo.GetByID(ctx, uuid.New()); !errors.Is(err, domain.ErrVenueNotFound) {
		t.Errorf("GetByID() for missing venue error = %v, want ErrVenueNotFound", err)
	}
}
