package service

import (
	"context"

	"github.com/google/uuid"

	"kitchen-api/internal/domain"
)

// MenuItemRepository — то, что нужно MenuService от хранилища меню.
type MenuItemRepository interface {
	ListByVenue(ctx context.Context, venueID uuid.UUID) ([]domain.MenuItem, error)
	Upsert(ctx context.Context, item domain.MenuItem) error
}

// MenuService — бизнес-логика меню (чтение + приём вебхука синхронизации).
type MenuService struct {
	repo MenuItemRepository
}

// NewMenuService создаёт MenuService поверх готового MenuItemRepository.
func NewMenuService(repo MenuItemRepository) *MenuService {
	return &MenuService{repo: repo}
}

// ListByVenue возвращает меню заведения.
func (s *MenuService) ListByVenue(ctx context.Context, venueID uuid.UUID) ([]domain.MenuItem, error) {
	return s.repo.ListByVenue(ctx, venueID)
}

// Sync — принимает состояние товара от вебхука venue-service и перезаписывает кэш-строку.
func (s *MenuService) Sync(ctx context.Context, item domain.MenuItem) error {
	return s.repo.Upsert(ctx, item)
}
