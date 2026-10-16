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

type fakeMenuItemRepo struct {
	byVenue map[uuid.UUID][]domain.MenuItem
}

func (f *fakeMenuItemRepo) ListByVenue(_ context.Context, venueID uuid.UUID) ([]domain.MenuItem, error) {
	return f.byVenue[venueID], nil
}

func (f *fakeMenuItemRepo) Upsert(_ context.Context, item domain.MenuItem) error {
	if f.byVenue == nil {
		f.byVenue = make(map[uuid.UUID][]domain.MenuItem)
	}
	f.byVenue[item.VenueID] = append(f.byVenue[item.VenueID], item)
	return nil
}

func TestMenuHandler_GetByVenue(t *testing.T) {
	venueID := uuid.New()
	venues := &fakeVenueRepo{venues: map[uuid.UUID]domain.Venue{venueID: {ID: venueID, Name: "Траттория"}}}
	menu := &fakeMenuItemRepo{byVenue: map[uuid.UUID][]domain.MenuItem{
		venueID: {{ID: uuid.New(), VenueID: venueID, Name: "Пицца", Price: "450.00", IsAvailable: true}},
	}}
	h := NewMenuHandler(service.NewVenueService(venues), service.NewMenuService(menu))

	req := httptest.NewRequest(http.MethodGet, "/venues/"+venueID.String()+"/menu", nil)
	req.SetPathValue("venueId", venueID.String())
	rec := httptest.NewRecorder()
	h.GetByVenue(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var resp []menuItemResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp) != 1 || resp[0].Name != "Пицца" {
		t.Errorf("response = %+v", resp)
	}
}

func TestMenuHandler_GetByVenue_VenueNotFound(t *testing.T) {
	venues := &fakeVenueRepo{venues: map[uuid.UUID]domain.Venue{}}
	h := NewMenuHandler(service.NewVenueService(venues), service.NewMenuService(&fakeMenuItemRepo{}))

	missingID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/venues/"+missingID.String()+"/menu", nil)
	req.SetPathValue("venueId", missingID.String())
	rec := httptest.NewRecorder()
	h.GetByVenue(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestMenuHandler_GetByVenue_InvalidVenueID(t *testing.T) {
	h := NewMenuHandler(service.NewVenueService(&fakeVenueRepo{}), service.NewMenuService(&fakeMenuItemRepo{}))

	req := httptest.NewRequest(http.MethodGet, "/venues/not-a-uuid/menu", nil)
	req.SetPathValue("venueId", "not-a-uuid")
	rec := httptest.NewRecorder()
	h.GetByVenue(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
