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

type fakeCategoryRepo struct {
	categories []domain.Category
}

func (f *fakeCategoryRepo) List(_ context.Context) ([]domain.Category, error) {
	return f.categories, nil
}

func TestCategoryHandler_List(t *testing.T) {
	repo := &fakeCategoryRepo{categories: []domain.Category{{ID: uuid.New(), Name: "Итальянская кухня"}}}
	h := NewCategoryHandler(service.NewCategoryService(repo))

	req := httptest.NewRequest(http.MethodGet, "/categories", nil)
	rec := httptest.NewRecorder()
	h.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var resp []categoryResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp) != 1 || resp[0].Name != "Итальянская кухня" {
		t.Errorf("response = %+v", resp)
	}
}
