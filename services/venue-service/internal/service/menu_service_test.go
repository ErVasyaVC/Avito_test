package service

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"venue-service/internal/domain"
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

type fakeMenuNotifier struct {
	calls []domain.MenuItem
}

func (f *fakeMenuNotifier) NotifyMenuItemChanged(_ context.Context, item domain.MenuItem) error {
	f.calls = append(f.calls, item)
	return nil
}

func TestMenuService_Create(t *testing.T) {
	repo := newFakeMenuItemRepo()
	notifier := &fakeMenuNotifier{}
	svc := NewMenuService(repo, notifier)

	desc := "Слои теста, мясной соус, бешамель"
	item, err := svc.Create(context.Background(), CreateInput{Name: "Лазанья", Description: &desc, Price: "470.00"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if !item.IsAvailable {
		t.Errorf("new item should be available by default")
	}
	if len(repo.items) != 1 {
		t.Errorf("repo has %d items, want 1", len(repo.items))
	}
	if len(notifier.calls) != 1 {
		t.Errorf("notifier called %d times, want 1", len(notifier.calls))
	}
}

func TestMenuService_Update_PartialMerge(t *testing.T) {
	id := uuid.New()
	desc := "исходное описание"
	repo := newFakeMenuItemRepo()
	repo.items[id] = domain.MenuItem{ID: id, Name: "Паста", Description: &desc, Price: "390.00", IsAvailable: true}
	notifier := &fakeMenuNotifier{}
	svc := NewMenuService(repo, notifier)

	newPrice := "420.00"
	got, err := svc.Update(context.Background(), id, UpdateInput{Price: &newPrice})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if got.Name != "Паста" {
		t.Errorf("Name = %q, want unchanged 'Паста'", got.Name)
	}
	if got.Description == nil || *got.Description != "исходное описание" {
		t.Errorf("Description = %v, want unchanged", got.Description)
	}
	if got.Price != "420.00" {
		t.Errorf("Price = %q, want 420.00", got.Price)
	}
	if !got.IsAvailable {
		t.Errorf("IsAvailable = false, want unchanged true")
	}
}

func TestMenuService_Update_NotFound(t *testing.T) {
	repo := newFakeMenuItemRepo()
	svc := NewMenuService(repo, &fakeMenuNotifier{})

	name := "x"
	_, err := svc.Update(context.Background(), uuid.New(), UpdateInput{Name: &name})
	if err == nil {
		t.Fatal("Update() for missing item should return error")
	}
}

func TestMenuService_Remove(t *testing.T) {
	id := uuid.New()
	repo := newFakeMenuItemRepo()
	repo.items[id] = domain.MenuItem{ID: id, Name: "Тирамису", Price: "280.00", IsAvailable: true}
	notifier := &fakeMenuNotifier{}
	svc := NewMenuService(repo, notifier)

	if err := svc.Remove(context.Background(), id); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	if _, ok := repo.items[id]; ok {
		t.Errorf("item still present in fake repo after Remove()")
	}
	if len(notifier.calls) != 1 {
		t.Errorf("notifier called %d times, want 1", len(notifier.calls))
	}
	if notifier.calls[0].RemovedAt == nil {
		t.Errorf("notifier should receive item with RemovedAt set")
	}
}
