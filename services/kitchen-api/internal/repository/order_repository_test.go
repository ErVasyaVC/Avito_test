package repository

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"kitchen-api/internal/domain"
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
	_, err := pool.Exec(context.Background(),
		"TRUNCATE order_status_history, order_items, orders, menu_items, venue_categories, venues, categories CASCADE")
	if err != nil {
		t.Fatalf("truncate: %v", err)
	}
}

func TestCategoryRepository_List(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	truncateAll(t, pool)

	_, err := pool.Exec(ctx, "INSERT INTO categories (id, name) VALUES ($1, $2)", uuid.New(), "Тестовая кухня")
	if err != nil {
		t.Fatalf("insert category: %v", err)
	}

	repo := NewCategoryRepository(pool)
	categories, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(categories) != 1 || categories[0].Name != "Тестовая кухня" {
		t.Errorf("unexpected categories: %+v", categories)
	}
}

func TestOrderRepository_CreateAndGetByID(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	truncateAll(t, pool)

	venueID := uuid.New()
	_, err := pool.Exec(ctx, "INSERT INTO venues (id, name, api_base_url, is_active) VALUES ($1, $2, $3, true)",
		venueID, "Тестовое заведение", "http://venue-service:8081")
	if err != nil {
		t.Fatalf("insert venue: %v", err)
	}

	menuItemID := uuid.New()
	_, err = pool.Exec(ctx, "INSERT INTO menu_items (id, venue_id, external_id, name, price, is_available) VALUES ($1, $2, $3, $4, $5, true)",
		menuItemID, venueID, "ext-1", "Тестовое блюдо", "100.00")
	if err != nil {
		t.Fatalf("insert menu item: %v", err)
	}

	repo := NewOrderRepository(pool)
	order := domain.Order{
		ID:              uuid.New(),
		VenueID:         venueID,
		Status:          domain.OrderStatusCreated,
		CustomerName:    "Тест",
		CustomerPhone:   "+70000000000",
		DeliveryAddress: "адрес",
		Items: []domain.OrderItem{
			{MenuItemID: menuItemID, NameSnapshot: "Тестовое блюдо", PriceSnapshot: "100.00", Quantity: 2},
		},
	}

	if err := repo.Create(ctx, order); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := repo.GetByID(ctx, order.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got.Status != domain.OrderStatusCreated {
		t.Errorf("Status = %q, want %q", got.Status, domain.OrderStatusCreated)
	}
	if len(got.Items) != 1 || got.Items[0].Quantity != 2 {
		t.Errorf("unexpected items: %+v", got.Items)
	}

	if err := repo.UpdateStatus(ctx, order.ID, domain.OrderStatusCooking); err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}
	got, err = repo.GetByID(ctx, order.ID)
	if err != nil {
		t.Fatalf("GetByID() after update error = %v", err)
	}
	if got.Status != domain.OrderStatusCooking {
		t.Errorf("Status after update = %q, want %q", got.Status, domain.OrderStatusCooking)
	}

	if _, err := repo.GetByID(ctx, uuid.New()); !errors.Is(err, domain.ErrOrderNotFound) {
		t.Errorf("GetByID() for missing order error = %v, want ErrOrderNotFound", err)
	}
}
