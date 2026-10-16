package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"kitchen-api/internal/domain"
)

// OrderRepository читает и пишет заказы, их позиции и историю статусов.
type OrderRepository struct {
	pool *pgxpool.Pool
}

// NewOrderRepository создаёт OrderRepository поверх готового пула соединений.
func NewOrderRepository(pool *pgxpool.Pool) *OrderRepository {
	return &OrderRepository{pool: pool}
}

// Create — заказ + его позиции + первая запись в истории статусов одной транзакцией.
func (r *OrderRepository) Create(ctx context.Context, order domain.Order) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx, `
		INSERT INTO orders (id, venue_id, status, customer_name, customer_phone, delivery_address, comment, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, now(), now())
	`, order.ID, order.VenueID, order.Status, order.CustomerName, order.CustomerPhone, order.DeliveryAddress, order.Comment)
	if err != nil {
		return err
	}

	for _, item := range order.Items {
		_, err = tx.Exec(ctx, `
			INSERT INTO order_items (id, order_id, menu_item_id, name_snapshot, price_snapshot, quantity)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, uuid.New(), order.ID, item.MenuItemID, item.NameSnapshot, item.PriceSnapshot, item.Quantity)
		if err != nil {
			return err
		}
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO order_status_history (id, order_id, status, changed_at)
		VALUES ($1, $2, $3, now())
	`, uuid.New(), order.ID, order.Status)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// GetByID возвращает заказ вместе с его позициями.
func (r *OrderRepository) GetByID(ctx context.Context, id uuid.UUID) (domain.Order, error) {
	var o domain.Order
	err := r.pool.QueryRow(ctx, `
		SELECT id, venue_id, status, customer_name, customer_phone, delivery_address, comment, created_at, updated_at
		FROM orders
		WHERE id = $1
	`, id).Scan(
		&o.ID, &o.VenueID, &o.Status, &o.CustomerName, &o.CustomerPhone,
		&o.DeliveryAddress, &o.Comment, &o.CreatedAt, &o.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Order{}, domain.ErrOrderNotFound
	}
	if err != nil {
		return domain.Order{}, err
	}

	items, err := r.itemsForOrder(ctx, id)
	if err != nil {
		return domain.Order{}, err
	}
	o.Items = items

	return o, nil
}

func (r *OrderRepository) itemsForOrder(ctx context.Context, orderID uuid.UUID) ([]domain.OrderItem, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT menu_item_id, name_snapshot, price_snapshot, quantity
		FROM order_items
		WHERE order_id = $1
	`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.OrderItem
	for rows.Next() {
		var it domain.OrderItem
		if err := rows.Scan(&it.MenuItemID, &it.NameSnapshot, &it.PriceSnapshot, &it.Quantity); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

// UpdateStatus — меняет статус и дописывает историю одной транзакцией.
func (r *OrderRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.OrderStatus) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `UPDATE orders SET status = $1, updated_at = now() WHERE id = $2`, status, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrOrderNotFound
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO order_status_history (id, order_id, status, changed_at)
		VALUES ($1, $2, $3, now())
	`, uuid.New(), id, status)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}
