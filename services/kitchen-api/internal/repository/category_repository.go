// Package repository — единственный слой с SQL-запросами к Postgres.
package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"kitchen-api/internal/domain"
)

// CategoryRepository читает категории из таблицы categories.
type CategoryRepository struct {
	pool *pgxpool.Pool
}

// NewCategoryRepository создаёт CategoryRepository поверх готового пула соединений.
func NewCategoryRepository(pool *pgxpool.Pool) *CategoryRepository {
	return &CategoryRepository{pool: pool}
}

// List возвращает все категории, отсортированные по имени.
func (r *CategoryRepository) List(ctx context.Context) ([]domain.Category, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, name FROM categories ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []domain.Category
	for rows.Next() {
		var c domain.Category
		if err := rows.Scan(&c.ID, &c.Name); err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}
	return categories, rows.Err()
}
