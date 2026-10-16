package service

import (
	"context"
	"kitchen-api/internal/domain"
)

// CategoryRepository — то, что нужно CategoryService от хранилища категорий.
type CategoryRepository interface {
	List(ctx context.Context) ([]domain.Category, error)
}

// CategoryService — бизнес-логика категорий (сейчас без правил, чистый проброс).
type CategoryService struct {
	repo CategoryRepository
}

// NewCategoryService создаёт CategoryService поверх готового CategoryRepository.
func NewCategoryService(repo CategoryRepository) *CategoryService {
	return &CategoryService{repo: repo}
}

// List возвращает все категории.
func (s *CategoryService) List(ctx context.Context) ([]domain.Category, error) {
	return s.repo.List(ctx)
}
