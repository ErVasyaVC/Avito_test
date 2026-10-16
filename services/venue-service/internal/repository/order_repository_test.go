package repository

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"venue-service/internal/domain"
)

// testPool подключается к тестовой БД из TEST_DATABASE_URL. Если переменная не
// задана, тест пропускается — так можно гонять `go test ./...` без поднятой БД.
func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping integration test")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect to test db: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func truncateAll(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	_, err := pool.Exec(context.Background(), "TRUNCATE order_items, orders, menu_items CASCADE")
	if err != nil {
		t.Fatalf("truncate: %v", err)
	}
}

func TestMenuItemRepository_ListAndGetAvailableByIDs(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	truncateAll(t, pool)

	availableID := uuid.New()
	unavailableID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO menu_items (id, name, price, is_available) VALUES ($1, $2, $3, true), ($4, $5, $6, false)`,
		availableID, "Пицца Маргарита", "450.00", unavailableID, "Тирамису", "280.00")
	if err != nil {
		t.Fatalf("insert menu items: %v", err)
	}

	repo := NewMenuItemRepository(pool)

	all, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(all) != 2 {
		t.Errorf("List() returned %d items, want 2", len(all))
	}

	available, err := repo.GetAvailableByIDs(ctx, []uuid.UUID{availableID, unavailableID})
	if err != nil {
		t.Fatalf("GetAvailableByIDs() error = %v", err)
	}
	if len(available) != 1 || available[0].ID != availableID {
		t.Errorf("GetAvailableByIDs() = %+v, want only %v", available, availableID)
	}
}

func TestMenuItemRepository_SoftDelete(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	truncateAll(t, pool)

	id := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO menu_items (id, name, price, is_available) VALUES ($1, $2, $3, true)`, id, "Рамен", "420.00")
	if err != nil {
		t.Fatalf("insert menu item: %v", err)
	}

	repo := NewMenuItemRepository(pool)
	if err := repo.SoftDelete(ctx, id); err != nil {
		t.Fatalf("SoftDelete() error = %v", err)
	}

	items, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(items) != 0 {
		t.Errorf("List() after SoftDelete() = %+v, want empty", items)
	}

	if err := repo.SoftDelete(ctx, uuid.New()); !errors.Is(err, domain.ErrMenuItemNotFound) {
		t.Errorf("SoftDelete() for missing item error = %v, want ErrMenuItemNotFound", err)
	}
}

func TestOrderRepository_CreateAndGetByKitchenOrderID(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	truncateAll(t, pool)

	menuItemID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO menu_items (id, name, price, is_available) VALUES ($1, $2, $3, true)`,
		menuItemID, "Мисо-суп", "180.00")
	if err != nil {
		t.Fatalf("insert menu item: %v", err)
	}

	repo := NewOrderRepository(pool)
	kitchenOrderID := uuid.New()
	order := domain.Order{
		ID:             uuid.New(),
		KitchenOrderID: kitchenOrderID,
		Status:         domain.OrderStatusAccepted,
		Items: []domain.OrderItem{
			{MenuItemID: menuItemID, NameSnapshot: "Мисо-суп", PriceSnapshot: "180.00", Quantity: 3},
		},
	}

	if err := repo.Create(ctx, order); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := repo.GetByKitchenOrderID(ctx, kitchenOrderID)
	if err != nil {
		t.Fatalf("GetByKitchenOrderID() error = %v", err)
	}
	if got.Status != domain.OrderStatusAccepted {
		t.Errorf("Status = %q, want %q", got.Status, domain.OrderStatusAccepted)
	}
	if len(got.Items) != 1 || got.Items[0].Quantity != 3 {
		t.Errorf("unexpected items: %+v", got.Items)
	}

	if err := repo.UpdateStatus(ctx, kitchenOrderID, domain.OrderStatusCooking); err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}
	got, err = repo.GetByKitchenOrderID(ctx, kitchenOrderID)
	if err != nil {
		t.Fatalf("GetByKitchenOrderID() after update error = %v", err)
	}
	if got.Status != domain.OrderStatusCooking {
		t.Errorf("Status after update = %q, want %q", got.Status, domain.OrderStatusCooking)
	}

	if _, err := repo.GetByKitchenOrderID(ctx, uuid.New()); !errors.Is(err, domain.ErrOrderNotFound) {
		t.Errorf("GetByKitchenOrderID() for missing order error = %v, want ErrOrderNotFound", err)
	}
}
