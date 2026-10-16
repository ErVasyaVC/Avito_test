package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"kitchen-api/internal/domain"
)

type fakeVenueRepo struct {
	venues map[uuid.UUID]domain.Venue
}

func (f *fakeVenueRepo) List(_ context.Context, categoryID *uuid.UUID) ([]domain.Venue, error) {
	var result []domain.Venue
	for _, v := range f.venues {
		if categoryID == nil {
			result = append(result, v)
			continue
		}
		for _, c := range v.Categories {
			if c.ID == *categoryID {
				result = append(result, v)
				break
			}
		}
	}
	return result, nil
}

func (f *fakeVenueRepo) GetByID(_ context.Context, id uuid.UUID) (domain.Venue, error) {
	v, ok := f.venues[id]
	if !ok {
		return domain.Venue{}, domain.ErrVenueNotFound
	}
	return v, nil
}

func TestVenueService_List_FiltersByCategory(t *testing.T) {
	catA := domain.Category{ID: uuid.New(), Name: "Итальянская кухня"}
	catB := domain.Category{ID: uuid.New(), Name: "Азиатская кухня"}
	venueA := domain.Venue{ID: uuid.New(), Name: "Траттория", Categories: []domain.Category{catA}}
	venueB := domain.Venue{ID: uuid.New(), Name: "Дом лапши", Categories: []domain.Category{catB}}

	repo := &fakeVenueRepo{venues: map[uuid.UUID]domain.Venue{venueA.ID: venueA, venueB.ID: venueB}}
	svc := NewVenueService(repo)

	got, err := svc.List(context.Background(), &catA.ID)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(got) != 1 || got[0].ID != venueA.ID {
		t.Errorf("List(catA) = %+v, want only venueA", got)
	}

	all, err := svc.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List(nil) error = %v", err)
	}
	if len(all) != 2 {
		t.Errorf("List(nil) = %d venues, want 2", len(all))
	}
}

func TestVenueService_GetByID_NotFound(t *testing.T) {
	svc := NewVenueService(&fakeVenueRepo{venues: map[uuid.UUID]domain.Venue{}})

	_, err := svc.GetByID(context.Background(), uuid.New())
	if !errors.Is(err, domain.ErrVenueNotFound) {
		t.Errorf("GetByID() error = %v, want ErrVenueNotFound", err)
	}
}
