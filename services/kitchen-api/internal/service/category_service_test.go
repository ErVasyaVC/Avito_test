package service

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"kitchen-api/internal/domain"
)

type fakeCategoryRepo struct {
	categories []domain.Category
}

func (f *fakeCategoryRepo) List(_ context.Context) ([]domain.Category, error) {
	return f.categories, nil
}

func TestCategoryService_List(t *testing.T) {
	want := []domain.Category{{ID: uuid.New(), Name: "Итальянская кухня"}}
	svc := NewCategoryService(&fakeCategoryRepo{categories: want})

	got, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(got) != 1 || got[0].Name != "Итальянская кухня" {
		t.Errorf("List() = %+v, want %+v", got, want)
	}
}
