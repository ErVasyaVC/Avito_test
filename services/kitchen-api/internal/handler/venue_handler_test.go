package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"kitchen-api/internal/domain"
	"kitchen-api/internal/service"
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

func TestVenueHandler_List(t *testing.T) {
	venueID := uuid.New()
	repo := &fakeVenueRepo{venues: map[uuid.UUID]domain.Venue{venueID: {ID: venueID, Name: "Траттория"}}}
	h := NewVenueHandler(service.NewVenueService(repo))

	req := httptest.NewRequest(http.MethodGet, "/venues", nil)
	rec := httptest.NewRecorder()
	h.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var resp []venueResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp) != 1 || resp[0].Name != "Траттория" {
		t.Errorf("response = %+v", resp)
	}
}

func TestVenueHandler_List_InvalidCategoryID(t *testing.T) {
	h := NewVenueHandler(service.NewVenueService(&fakeVenueRepo{venues: map[uuid.UUID]domain.Venue{}}))

	req := httptest.NewRequest(http.MethodGet, "/venues?category_id=not-a-uuid", nil)
	rec := httptest.NewRecorder()
	h.List(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestVenueHandler_GetByID(t *testing.T) {
	venueID := uuid.New()
	repo := &fakeVenueRepo{venues: map[uuid.UUID]domain.Venue{venueID: {ID: venueID, Name: "Траттория"}}}
	h := NewVenueHandler(service.NewVenueService(repo))

	req := httptest.NewRequest(http.MethodGet, "/venues/"+venueID.String(), nil)
	req.SetPathValue("venueId", venueID.String())
	rec := httptest.NewRecorder()
	h.GetByID(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestVenueHandler_GetByID_NotFound(t *testing.T) {
	h := NewVenueHandler(service.NewVenueService(&fakeVenueRepo{venues: map[uuid.UUID]domain.Venue{}}))

	missingID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/venues/"+missingID.String(), nil)
	req.SetPathValue("venueId", missingID.String())
	rec := httptest.NewRecorder()
	h.GetByID(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestVenueHandler_GetByID_InvalidID(t *testing.T) {
	h := NewVenueHandler(service.NewVenueService(&fakeVenueRepo{}))

	req := httptest.NewRequest(http.MethodGet, "/venues/not-a-uuid", nil)
	req.SetPathValue("venueId", "not-a-uuid")
	rec := httptest.NewRecorder()
	h.GetByID(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
