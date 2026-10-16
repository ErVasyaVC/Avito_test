package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"kitchen-api/internal/domain"
)

// MenuItemRepository читает и пишет кэш меню в таблице menu_items.
type MenuItemRepository struct {
	pool *pgxpool.Pool
}

// NewMenuItemRepository создаёт MenuItemRepository поверх готового пула соединений.
func NewMenuItemRepository(pool *pgxpool.Pool) *MenuItemRepository {
	return &MenuItemRepository{pool: pool}
}

// ListByVenue возвращает меню заведения (включая недоступные, но не убранные позиции).
func (r *MenuItemRepository) ListByVenue(ctx context.Context, venueID uuid.UUID) ([]domain.MenuItem, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, venue_id, name, description, price, is_available
		FROM menu_items
		WHERE venue_id = $1 AND removed_at IS NULL
		ORDER BY name
	`, venueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.MenuItem
	for rows.Next() {
		var m domain.MenuItem
		if err := rows.Scan(&m.ID, &m.VenueID, &m.Name, &m.Description, &m.Price, &m.IsAvailable); err != nil {
			return nil, err
		}
		items = append(items, m)
	}
	return items, rows.Err()
}

// GetAvailableByIDs — только доступные позиции (removed_at IS NULL, is_available = true).
// Отсутствие ID в результате означает "недоступен" — этим пользуется OrderService при оформлении заказа.
func (r *MenuItemRepository) GetAvailableByIDs(ctx context.Context, venueID uuid.UUID, itemIDs []uuid.UUID) ([]domain.MenuItem, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, venue_id, name, description, price, is_available
		FROM menu_items
		WHERE venue_id = $1 AND id = ANY($2) AND removed_at IS NULL AND is_available = true
	`, venueID, itemIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.MenuItem
	for rows.Next() {
		var m domain.MenuItem
		if err := rows.Scan(&m.ID, &m.VenueID, &m.Name, &m.Description, &m.Price, &m.IsAvailable); err != nil {
			return nil, err
		}
		items = append(items, m)
	}
	return items, rows.Err()
}

// Upsert — вызывается вебхуком от venue-service при любом изменении товара
// (добавлен/изменён/выключен/убран). item.ID используется и как id, и как external_id кэша.
func (r *MenuItemRepository) Upsert(ctx context.Context, item domain.MenuItem) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO menu_items (id, venue_id, external_id, name, description, price, is_available, removed_at, updated_at)
		VALUES ($1, $2, $8, $3, $4, $5, $6, $7, now())
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			price = EXCLUDED.price,
			is_available = EXCLUDED.is_available,
			removed_at = EXCLUDED.removed_at,
			updated_at = now()
	`, item.ID, item.VenueID, item.Name, item.Description, item.Price, item.IsAvailable, item.RemovedAt, item.ID.String())
	return err
}
