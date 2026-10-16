package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"kitchen-api/internal/service"
)

func TestMenuSyncHandler_Sync(t *testing.T) {
	repo := &fakeMenuItemRepo{}
	h := NewMenuSyncHandler(service.NewMenuService(repo))

	venueID := uuid.New()
	itemID := uuid.New()
	body, _ := json.Marshal(syncMenuItemRequest{Name: "Лазанья", Price: "470.00", IsAvailable: true})

	req := httptest.NewRequest(http.MethodPut, "/internal/venues/"+venueID.String()+"/menu/"+itemID.String(), bytes.NewReader(body))
	req.SetPathValue("venueId", venueID.String())
	req.SetPathValue("itemId", itemID.String())
	rec := httptest.NewRecorder()
	h.Sync(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusNoContent, rec.Body.String())
	}
	if len(repo.byVenue[venueID]) != 1 {
		t.Errorf("item was not upserted into repo")
	}
}

func TestMenuSyncHandler_Sync_InvalidVenueID(t *testing.T) {
	h := NewMenuSyncHandler(service.NewMenuService(&fakeMenuItemRepo{}))

	req := httptest.NewRequest(http.MethodPut, "/internal/venues/not-a-uuid/menu/"+uuid.New().String(), bytes.NewReader([]byte("{}")))
	req.SetPathValue("venueId", "not-a-uuid")
	req.SetPathValue("itemId", uuid.New().String())
	rec := httptest.NewRecorder()
	h.Sync(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestMenuSyncHandler_Sync_InvalidBody(t *testing.T) {
	h := NewMenuSyncHandler(service.NewMenuService(&fakeMenuItemRepo{}))

	req := httptest.NewRequest(http.MethodPut, "/internal/venues/"+uuid.New().String()+"/menu/"+uuid.New().String(), bytes.NewReader([]byte("not json")))
	req.SetPathValue("venueId", uuid.New().String())
	req.SetPathValue("itemId", uuid.New().String())
	rec := httptest.NewRecorder()
	h.Sync(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
