package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"venue-service/internal/domain"
)

// OrderRepository читает и пишет заказы и их позиции.
type OrderRepository struct {
	pool *pgxpool.Pool
}

// NewOrderRepository создаёт OrderRepository поверх готового пула соединений.
func NewOrderRepository(pool *pgxpool.Pool) *OrderRepository {
	return &OrderRepository{pool: pool}
}

// Create — заказ + его позиции одной транзакцией.
func (r *OrderRepository) Create(ctx context.Context, order domain.Order) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx, `
		INSERT INTO orders (id, kitchen_order_id, status, created_at, updated_at)
		VALUES ($1, $2, $3, now(), now())
	`, order.ID, order.KitchenOrderID, order.Status)
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

	return tx.Commit(ctx)
}

// GetByKitchenOrderID возвращает заказ вместе с его позициями. kitchenOrderID —
// единственный ключ поиска, отдельного публичного ID у venue-service нет.
func (r *OrderRepository) GetByKitchenOrderID(ctx context.Context, kitchenOrderID uuid.UUID) (domain.Order, error) {
	var o domain.Order
	err := r.pool.QueryRow(ctx, `
		SELECT id, kitchen_order_id, status, created_at, updated_at
		FROM orders
		WHERE kitchen_order_id = $1
	`, kitchenOrderID).Scan(&o.ID, &o.KitchenOrderID, &o.Status, &o.CreatedAt, &o.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Order{}, domain.ErrOrderNotFound
	}
	if err != nil {
		return domain.Order{}, err
	}

	items, err := r.itemsForOrder(ctx, o.ID)
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

// UpdateStatus меняет статус заказа по kitchen_order_id.
func (r *OrderRepository) UpdateStatus(ctx context.Context, kitchenOrderID uuid.UUID, status domain.OrderStatus) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE orders SET status = $1, updated_at = now() WHERE kitchen_order_id = $2
	`, status, kitchenOrderID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrOrderNotFound
	}
	return nil
}
