// Package repository — единственный слой с SQL-запросами к Postgres.
package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"venue-service/internal/domain"
)

// MenuItemRepository читает и пишет меню в таблице menu_items.
type MenuItemRepository struct {
	pool *pgxpool.Pool
}

// NewMenuItemRepository создаёт MenuItemRepository поверх готового пула соединений.
func NewMenuItemRepository(pool *pgxpool.Pool) *MenuItemRepository {
	return &MenuItemRepository{pool: pool}
}

// List возвращает меню (без убранных позиций).
func (r *MenuItemRepository) List(ctx context.Context) ([]domain.MenuItem, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, description, price, is_available
		FROM menu_items
		WHERE removed_at IS NULL
		ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.MenuItem
	for rows.Next() {
		var m domain.MenuItem
		if err := rows.Scan(&m.ID, &m.Name, &m.Description, &m.Price, &m.IsAvailable); err != nil {
			return nil, err
		}
		items = append(items, m)
	}
	return items, rows.Err()
}

// GetAvailableByIDs — только доступные позиции; отсутствие ID в результате значит "недоступен".
func (r *MenuItemRepository) GetAvailableByIDs(ctx context.Context, ids []uuid.UUID) ([]domain.MenuItem, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, description, price, is_available
		FROM menu_items
		WHERE id = ANY($1) AND removed_at IS NULL AND is_available = true
	`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.MenuItem
	for rows.Next() {
		var m domain.MenuItem
		if err := rows.Scan(&m.ID, &m.Name, &m.Description, &m.Price, &m.IsAvailable); err != nil {
			return nil, err
		}
		items = append(items, m)
	}
	return items, rows.Err()
}

// GetByID возвращает позицию меню по ID (в т.ч. недоступную, но не убранную).
func (r *MenuItemRepository) GetByID(ctx context.Context, id uuid.UUID) (domain.MenuItem, error) {
	var m domain.MenuItem
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, description, price, is_available
		FROM menu_items
		WHERE id = $1 AND removed_at IS NULL
	`, id).Scan(&m.ID, &m.Name, &m.Description, &m.Price, &m.IsAvailable)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.MenuItem{}, domain.ErrMenuItemNotFound
	}
	return m, err
}

// Create добавляет новую позицию меню.
func (r *MenuItemRepository) Create(ctx context.Context, item domain.MenuItem) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO menu_items (id, name, description, price, is_available, updated_at)
		VALUES ($1, $2, $3, $4, $5, now())
	`, item.ID, item.Name, item.Description, item.Price, item.IsAvailable)
	return err
}

// Update перезаписывает все изменяемые поля позиции (частичный merge делает service-слой).
func (r *MenuItemRepository) Update(ctx context.Context, item domain.MenuItem) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE menu_items
		SET name = $1, description = $2, price = $3, is_available = $4, updated_at = now()
		WHERE id = $5 AND removed_at IS NULL
	`, item.Name, item.Description, item.Price, item.IsAvailable, item.ID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrMenuItemNotFound
	}
	return nil
}

// SoftDelete помечает позицию убранной из меню (removed_at), не удаляя строку физически —
// иначе сломались бы ссылки на неё из уже принятых заказов (order_items).
func (r *MenuItemRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE menu_items SET removed_at = now(), updated_at = now()
		WHERE id = $1 AND removed_at IS NULL
	`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrMenuItemNotFound
	}
	return nil
}
