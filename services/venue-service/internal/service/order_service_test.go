package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"venue-service/internal/domain"
)

type fakeOrderRepo struct {
	orders map[uuid.UUID]domain.Order // keyed by KitchenOrderID
}

func newFakeOrderRepo() *fakeOrderRepo {
	return &fakeOrderRepo{orders: make(map[uuid.UUID]domain.Order)}
}

func (f *fakeOrderRepo) Create(_ context.Context, order domain.Order) error {
	f.orders[order.KitchenOrderID] = order
	return nil
}

func (f *fakeOrderRepo) GetByKitchenOrderID(_ context.Context, kitchenOrderID uuid.UUID) (domain.Order, error) {
	o, ok := f.orders[kitchenOrderID]
	if !ok {
		return domain.Order{}, domain.ErrOrderNotFound
	}
	return o, nil
}

func (f *fakeOrderRepo) UpdateStatus(_ context.Context, kitchenOrderID uuid.UUID, status domain.OrderStatus) error {
	o, ok := f.orders[kitchenOrderID]
	if !ok {
		return domain.ErrOrderNotFound
	}
	o.Status = status
	f.orders[kitchenOrderID] = o
	return nil
}

type fakeAvailabilityChecker struct {
	available map[uuid.UUID]domain.MenuItem
}

func (f *fakeAvailabilityChecker) GetAvailableByIDs(_ context.Context, itemIDs []uuid.UUID) ([]domain.MenuItem, error) {
	var result []domain.MenuItem
	for _, id := range itemIDs {
		if m, ok := f.available[id]; ok {
			result = append(result, m)
		}
	}
	return result, nil
}

type fakeNotifier struct {
	calls int
}

func (f *fakeNotifier) NotifyStatusChanged(_ context.Context, _ domain.Order) error {
	f.calls++
	return nil
}

func TestOrderService_Receive_AllAvailable(t *testing.T) {
	itemID := uuid.New()
	kitchenOrderID := uuid.New()
	repo := newFakeOrderRepo()
	menu := &fakeAvailabilityChecker{available: map[uuid.UUID]domain.MenuItem{
		itemID: {ID: itemID, Name: "Паста Карбонара", Price: "390.00", IsAvailable: true},
	}}
	svc := NewOrderService(repo, menu, &fakeNotifier{})

	order, err := svc.Receive(context.Background(), kitchenOrderID, []ReceiveOrderItem{{MenuItemID: itemID, Quantity: 1}})
	if err != nil {
		t.Fatalf("Receive() error = %v, want nil", err)
	}
	if order.Status != domain.OrderStatusAccepted {
		t.Errorf("Status = %q, want %q", order.Status, domain.OrderStatusAccepted)
	}
	if len(order.Items) != 1 || order.Items[0].NameSnapshot != "Паста Карбонара" {
		t.Errorf("unexpected items: %+v", order.Items)
	}
	if len(repo.orders) != 1 {
		t.Errorf("repo has %d orders, want 1", len(repo.orders))
	}
}

func TestOrderService_Receive_UnavailableItem(t *testing.T) {
	availableID := uuid.New()
	unavailableID := uuid.New()
	kitchenOrderID := uuid.New()
	repo := newFakeOrderRepo()
	menu := &fakeAvailabilityChecker{available: map[uuid.UUID]domain.MenuItem{
		availableID: {ID: availableID, Name: "Гёдза", Price: "320.00", IsAvailable: true},
	}}
	svc := NewOrderService(repo, menu, &fakeNotifier{})

	_, err := svc.Receive(context.Background(), kitchenOrderID, []ReceiveOrderItem{
		{MenuItemID: availableID, Quantity: 1},
		{MenuItemID: unavailableID, Quantity: 1},
	})

	var unavailableErr *UnavailableItemsError
	if !errors.As(err, &unavailableErr) {
		t.Fatalf("Receive() error = %v, want *UnavailableItemsError", err)
	}
	if len(unavailableErr.ItemIDs) != 1 || unavailableErr.ItemIDs[0] != unavailableID {
		t.Errorf("unexpected unavailable IDs: %v", unavailableErr.ItemIDs)
	}
	if len(repo.orders) != 0 {
		t.Errorf("order was created despite unavailable item, repo has %d orders", len(repo.orders))
	}
}

func TestOrderService_UpdateStatus(t *testing.T) {
	kitchenOrderID := uuid.New()
	repo := newFakeOrderRepo()
	repo.orders[kitchenOrderID] = domain.Order{KitchenOrderID: kitchenOrderID, Status: domain.OrderStatusAccepted}
	notifier := &fakeNotifier{}
	svc := NewOrderService(repo, &fakeAvailabilityChecker{}, notifier)

	order, err := svc.UpdateStatus(context.Background(), kitchenOrderID, domain.OrderStatusCooking)
	if err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}
	if order.Status != domain.OrderStatusCooking {
		t.Errorf("Status = %q, want %q", order.Status, domain.OrderStatusCooking)
	}
	if notifier.calls != 1 {
		t.Errorf("notifier called %d times, want 1", notifier.calls)
	}

	if _, err := svc.UpdateStatus(context.Background(), kitchenOrderID, domain.OrderStatusCancelledByUser); !errors.Is(err, domain.ErrInvalidStatusTransition) {
		t.Errorf("expected ErrInvalidStatusTransition (too late to cancel while cooking), got %v", err)
	}
}

func TestOrderService_UpdateStatus_CancelledByUserDoesNotNotify(t *testing.T) {
	kitchenOrderID := uuid.New()
	repo := newFakeOrderRepo()
	repo.orders[kitchenOrderID] = domain.Order{KitchenOrderID: kitchenOrderID, Status: domain.OrderStatusAccepted}
	notifier := &fakeNotifier{}
	svc := NewOrderService(repo, &fakeAvailabilityChecker{}, notifier)

	order, err := svc.UpdateStatus(context.Background(), kitchenOrderID, domain.OrderStatusCancelledByUser)
	if err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}
	if order.Status != domain.OrderStatusCancelledByUser {
		t.Errorf("Status = %q, want %q", order.Status, domain.OrderStatusCancelledByUser)
	}
	if notifier.calls != 0 {
		t.Errorf("notifier called %d times, want 0 (cancelled_by_user comes from kitchen-api itself, no need to echo back)", notifier.calls)
	}
}
