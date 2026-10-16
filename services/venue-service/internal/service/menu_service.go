package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"venue-service/internal/domain"
)

// MenuItemRepository — то, что нужно MenuService от хранилища меню.
type MenuItemRepository interface {
	List(ctx context.Context) ([]domain.MenuItem, error)
	GetByID(ctx context.Context, id uuid.UUID) (domain.MenuItem, error)
	Create(ctx context.Context, item domain.MenuItem) error
	Update(ctx context.Context, item domain.MenuItem) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
}

// MenuService — управление своим меню; после каждого изменения уведомляет kitchen-api.
type MenuService struct {
	repo     MenuItemRepository
	notifier MenuSyncNotifier
}

// NewMenuService создаёт MenuService поверх готовых зависимостей.
func NewMenuService(repo MenuItemRepository, notifier MenuSyncNotifier) *MenuService {
	return &MenuService{repo: repo, notifier: notifier}
}

// List возвращает меню.
func (s *MenuService) List(ctx context.Context) ([]domain.MenuItem, error) {
	return s.repo.List(ctx)
}

// CreateInput — вход для MenuService.Create.
type CreateInput struct {
	Name        string
	Description *string
	Price       string
}

// Create добавляет новую позицию меню и уведомляет kitchen-api.
func (s *MenuService) Create(ctx context.Context, input CreateInput) (domain.MenuItem, error) {
	item := domain.MenuItem{
		ID:          uuid.New(),
		Name:        input.Name,
		Description: input.Description,
		Price:       input.Price,
		IsAvailable: true,
	}
	if err := s.repo.Create(ctx, item); err != nil {
		return domain.MenuItem{}, err
	}
	if err := s.notifier.NotifyMenuItemChanged(ctx, item); err != nil {
		return domain.MenuItem{}, err
	}
	return item, nil
}

// UpdateInput — вход для MenuService.Update; nil-поля не меняются (частичное обновление).
type UpdateInput struct {
	Name        *string
	Description *string
	Price       *string
	IsAvailable *bool
}

// Update частично обновляет позицию меню и уведомляет kitchen-api.
func (s *MenuService) Update(ctx context.Context, id uuid.UUID, input UpdateInput) (domain.MenuItem, error) {
	item, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return domain.MenuItem{}, err
	}

	if input.Name != nil {
		item.Name = *input.Name
	}
	if input.Description != nil {
		item.Description = input.Description
	}
	if input.Price != nil {
		item.Price = *input.Price
	}
	if input.IsAvailable != nil {
		item.IsAvailable = *input.IsAvailable
	}

	if err := s.repo.Update(ctx, item); err != nil {
		return domain.MenuItem{}, err
	}
	if err := s.notifier.NotifyMenuItemChanged(ctx, item); err != nil {
		return domain.MenuItem{}, err
	}
	return item, nil
}

// Remove убирает позицию из меню (soft delete) и уведомляет kitchen-api.
func (s *MenuService) Remove(ctx context.Context, id uuid.UUID) error {
	item, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repo.SoftDelete(ctx, id); err != nil {
		return err
	}
	removedAt := time.Now()
	item.RemovedAt = &removedAt
	return s.notifier.NotifyMenuItemChanged(ctx, item)
}
