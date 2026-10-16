package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"venue-service/internal/domain"
	"venue-service/internal/service"
)

type fakeMenuItemRepo struct {
	items map[uuid.UUID]domain.MenuItem
}

func newFakeMenuItemRepo() *fakeMenuItemRepo {
	return &fakeMenuItemRepo{items: make(map[uuid.UUID]domain.MenuItem)}
}

func (f *fakeMenuItemRepo) List(_ context.Context) ([]domain.MenuItem, error) {
	var result []domain.MenuItem
	for _, m := range f.items {
		result = append(result, m)
	}
	return result, nil
}

func (f *fakeMenuItemRepo) GetByID(_ context.Context, id uuid.UUID) (domain.MenuItem, error) {
	m, ok := f.items[id]
	if !ok {
		return domain.MenuItem{}, domain.ErrMenuItemNotFound
	}
	return m, nil
}

func (f *fakeMenuItemRepo) Create(_ context.Context, item domain.MenuItem) error {
	f.items[item.ID] = item
	return nil
}

func (f *fakeMenuItemRepo) Update(_ context.Context, item domain.MenuItem) error {
	if _, ok := f.items[item.ID]; !ok {
		return domain.ErrMenuItemNotFound
	}
	f.items[item.ID] = item
	return nil
}

func (f *fakeMenuItemRepo) SoftDelete(_ context.Context, id uuid.UUID) error {
	if _, ok := f.items[id]; !ok {
		return domain.ErrMenuItemNotFound
	}
	delete(f.items, id)
	return nil
}

type fakeMenuNotifier struct{}

func (fakeMenuNotifier) NotifyMenuItemChanged(_ context.Context, _ domain.MenuItem) error { return nil }

func TestMenuHandler_List(t *testing.T) {
	repo := newFakeMenuItemRepo()
	id := uuid.New()
	repo.items[id] = domain.MenuItem{ID: id, Name: "Мисо-суп", Price: "180.00", IsAvailable: true}
	h := NewMenuHandler(service.NewMenuService(repo, fakeMenuNotifier{}))

	req := httptest.NewRequest(http.MethodGet, "/menu", nil)
	rec := httptest.NewRecorder()
	h.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var resp []menuItemResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp) != 1 || resp[0].Name != "Мисо-суп" {
		t.Errorf("response = %+v", resp)
	}
}

func TestMenuHandler_Add(t *testing.T) {
	repo := newFakeMenuItemRepo()
	h := NewMenuHandler(service.NewMenuService(repo, fakeMenuNotifier{}))

	body, _ := json.Marshal(addMenuItemRequest{Name: "Гёдза", Price: "320.00"})
	req := httptest.NewRequest(http.MethodPost, "/menu", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.Add(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	if len(repo.items) != 1 {
		t.Errorf("repo has %d items, want 1", len(repo.items))
	}
}

func TestMenuHandler_Add_MissingFields(t *testing.T) {
	h := NewMenuHandler(service.NewMenuService(newFakeMenuItemRepo(), fakeMenuNotifier{}))

	body, _ := json.Marshal(addMenuItemRequest{Name: "", Price: ""})
	req := httptest.NewRequest(http.MethodPost, "/menu", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.Add(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestMenuHandler_Update(t *testing.T) {
	repo := newFakeMenuItemRepo()
	id := uuid.New()
	repo.items[id] = domain.MenuItem{ID: id, Name: "Паста", Price: "390.00", IsAvailable: true}
	h := NewMenuHandler(service.NewMenuService(repo, fakeMenuNotifier{}))

	newPrice := "420.00"
	body, _ := json.Marshal(updateMenuItemRequest{Price: &newPrice})
	req := httptest.NewRequest(http.MethodPatch, "/menu/"+id.String(), bytes.NewReader(body))
	req.SetPathValue("itemId", id.String())
	rec := httptest.NewRecorder()
	h.Update(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var resp menuItemResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Price != "420.00" || resp.Name != "Паста" {
		t.Errorf("response = %+v, want price=420.00 name unchanged", resp)
	}
}

func TestMenuHandler_Update_NotFound(t *testing.T) {
	h := NewMenuHandler(service.NewMenuService(newFakeMenuItemRepo(), fakeMenuNotifier{}))

	name := "x"
	body, _ := json.Marshal(updateMenuItemRequest{Name: &name})
	missingID := uuid.New()
	req := httptest.NewRequest(http.MethodPatch, "/menu/"+missingID.String(), bytes.NewReader(body))
	req.SetPathValue("itemId", missingID.String())
	rec := httptest.NewRecorder()
	h.Update(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestMenuHandler_Remove(t *testing.T) {
	repo := newFakeMenuItemRepo()
	id := uuid.New()
	repo.items[id] = domain.MenuItem{ID: id, Name: "Тирамису", Price: "280.00", IsAvailable: true}
	h := NewMenuHandler(service.NewMenuService(repo, fakeMenuNotifier{}))

	req := httptest.NewRequest(http.MethodDelete, "/menu/"+id.String(), nil)
	req.SetPathValue("itemId", id.String())
	rec := httptest.NewRecorder()
	h.Remove(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if _, ok := repo.items[id]; ok {
		t.Errorf("item still present after Remove()")
	}
}

func TestMenuHandler_Remove_NotFound(t *testing.T) {
	h := NewMenuHandler(service.NewMenuService(newFakeMenuItemRepo(), fakeMenuNotifier{}))

	missingID := uuid.New()
	req := httptest.NewRequest(http.MethodDelete, "/menu/"+missingID.String(), nil)
	req.SetPathValue("itemId", missingID.String())
	rec := httptest.NewRecorder()
	h.Remove(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
