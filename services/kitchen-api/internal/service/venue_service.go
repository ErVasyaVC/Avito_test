// Package service содержит бизнес-логику: use case'ы поверх repository, ничего не знает про HTTP/SQL.
package service

import (
	"context"

	"github.com/google/uuid"

	"kitchen-api/internal/domain"
)

// VenueRepository — то, что нужно VenueService от хранилища заведений.
type VenueRepository interface {
	List(ctx context.Context, categoryID *uuid.UUID) ([]domain.Venue, error)
	GetByID(ctx context.Context, id uuid.UUID) (domain.Venue, error)
}

// VenueService — бизнес-логика заведений (сейчас без правил, чистый проброс).
type VenueService struct {
	repo VenueRepository
}

// NewVenueService создаёт VenueService поверх готового VenueRepository.
func NewVenueService(repo VenueRepository) *VenueService {
	return &VenueService{repo: repo}
}

// List возвращает активные заведения, опционально по категории.
func (s *VenueService) List(ctx context.Context, categoryID *uuid.UUID) ([]domain.Venue, error) {
	return s.repo.List(ctx, categoryID)
}

// GetByID возвращает заведение по ID.
func (s *VenueService) GetByID(ctx context.Context, id uuid.UUID) (domain.Venue, error) {
	return s.repo.GetByID(ctx, id)
}
