package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"kitchen-api/internal/domain"
)

type fakeOrderRepo struct {
	orders map[uuid.UUID]domain.Order
}

func newFakeOrderRepo() *fakeOrderRepo {
	return &fakeOrderRepo{orders: make(map[uuid.UUID]domain.Order)}
}

func (f *fakeOrderRepo) Create(_ context.Context, order domain.Order) error {
	f.orders[order.ID] = order
	return nil
}

func (f *fakeOrderRepo) GetByID(_ context.Context, id uuid.UUID) (domain.Order, error) {
	o, ok := f.orders[id]
	if !ok {
		return domain.Order{}, domain.ErrOrderNotFound
	}
	return o, nil
}

func (f *fakeOrderRepo) UpdateStatus(_ context.Context, id uuid.UUID, status domain.OrderStatus) error {
	o, ok := f.orders[id]
	if !ok {
		return domain.ErrOrderNotFound
	}
	o.Status = status
	f.orders[id] = o
	return nil
}

type fakeAvailabilityChecker struct {
	available map[uuid.UUID]domain.MenuItem
}

func (f *fakeAvailabilityChecker) GetAvailableByIDs(_ context.Context, _ uuid.UUID, itemIDs []uuid.UUID) ([]domain.MenuItem, error) {
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

func (f *fakeNotifier) NotifyVenue(_ context.Context, _ domain.Order) error {
	f.calls++
	return nil
}

func TestOrderService_Create_AllAvailable(t *testing.T) {
	itemID := uuid.New()
	venueID := uuid.New()
	repo := newFakeOrderRepo()
	menu := &fakeAvailabilityChecker{available: map[uuid.UUID]domain.MenuItem{
		itemID: {ID: itemID, Name: "Пицца", Price: "450.00", IsAvailable: true},
	}}
	notifier := &fakeNotifier{}
	svc := NewOrderService(repo, menu, notifier)

	order, err := svc.Create(context.Background(), CreateOrderInput{
		VenueID:         venueID,
		Items:           []CreateOrderItem{{MenuItemID: itemID, Quantity: 2}},
		CustomerName:    "Тест",
		CustomerPhone:   "+70000000000",
		DeliveryAddress: "адрес",
	})
	if err != nil {
		t.Fatalf("Create() error = %v, want nil", err)
	}
	if order.Status != domain.OrderStatusAccepted {
		t.Errorf("Status = %q, want %q", order.Status, domain.OrderStatusAccepted)
	}
	if len(order.Items) != 1 || order.Items[0].NameSnapshot != "Пицца" || order.Items[0].PriceSnapshot != "450.00" {
		t.Errorf("unexpected order items: %+v", order.Items)
	}
	if notifier.calls != 1 {
		t.Errorf("notifier called %d times, want 1", notifier.calls)
	}
	if len(repo.orders) != 1 {
		t.Errorf("repo has %d orders, want 1", len(repo.orders))
	}
}

func TestOrderService_Create_UnavailableItem(t *testing.T) {
	availableID := uuid.New()
	unavailableID := uuid.New()
	venueID := uuid.New()
	repo := newFakeOrderRepo()
	menu := &fakeAvailabilityChecker{available: map[uuid.UUID]domain.MenuItem{
		availableID: {ID: availableID, Name: "Паста", Price: "390.00", IsAvailable: true},
	}}
	notifier := &fakeNotifier{}
	svc := NewOrderService(repo, menu, notifier)

	_, err := svc.Create(context.Background(), CreateOrderInput{
		VenueID: venueID,
		Items: []CreateOrderItem{
			{MenuItemID: availableID, Quantity: 1},
			{MenuItemID: unavailableID, Quantity: 1},
		},
		CustomerName:    "Тест",
		CustomerPhone:   "+70000000000",
		DeliveryAddress: "адрес",
	})

	var unavailableErr *UnavailableItemsError
	if !errors.As(err, &unavailableErr) {
		t.Fatalf("Create() error = %v, want *UnavailableItemsError", err)
	}
	if len(unavailableErr.ItemIDs) != 1 || unavailableErr.ItemIDs[0] != unavailableID {
		t.Errorf("unexpected unavailable IDs: %v", unavailableErr.ItemIDs)
	}
	if len(repo.orders) != 0 {
		t.Errorf("order was created despite unavailable item, repo has %d orders", len(repo.orders))
	}
	if notifier.calls != 0 {
		t.Errorf("notifier should not be called when order creation fails, got %d calls", notifier.calls)
	}
}

func TestOrderService_Cancel(t *testing.T) {
	tests := []struct {
		name    string
		status  domain.OrderStatus
		wantErr error
	}{
		{"created can be cancelled", domain.OrderStatusCreated, nil},
		{"accepted can be cancelled", domain.OrderStatusAccepted, nil},
		{"cooking cannot be cancelled", domain.OrderStatusCooking, domain.ErrOrderCancelNotAllowed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := uuid.New()
			repo := newFakeOrderRepo()
			repo.orders[id] = domain.Order{ID: id, Status: tt.status}
			svc := NewOrderService(repo, &fakeAvailabilityChecker{}, &fakeNotifier{})

			_, err := svc.Cancel(context.Background(), id)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Cancel() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestOrderService_UpdateStatusFromVenue(t *testing.T) {
	id := uuid.New()
	repo := newFakeOrderRepo()
	repo.orders[id] = domain.Order{ID: id, Status: domain.OrderStatusCreated}
	svc := NewOrderService(repo, &fakeAvailabilityChecker{}, &fakeNotifier{})

	order, err := svc.UpdateStatusFromVenue(context.Background(), id, domain.OrderStatusCooking)
	if err != nil {
		t.Fatalf("UpdateStatusFromVenue() error = %v", err)
	}
	if order.Status != domain.OrderStatusCooking {
		t.Errorf("Status = %q, want %q", order.Status, domain.OrderStatusCooking)
	}

	if _, err := svc.UpdateStatusFromVenue(context.Background(), id, domain.OrderStatusDelivered); !errors.Is(err, domain.ErrInvalidStatusTransition) {
		t.Errorf("expected ErrInvalidStatusTransition, got %v", err)
	}
}

func TestOrderService_AdvanceDelivery(t *testing.T) {
	id := uuid.New()
	repo := newFakeOrderRepo()
	repo.orders[id] = domain.Order{ID: id, Status: domain.OrderStatusReady}
	svc := NewOrderService(repo, &fakeAvailabilityChecker{}, &fakeNotifier{})

	order, err := svc.AdvanceDelivery(context.Background(), id, domain.OrderStatusDelivering)
	if err != nil {
		t.Fatalf("AdvanceDelivery() error = %v", err)
	}
	if order.Status != domain.OrderStatusDelivering {
		t.Errorf("Status = %q, want %q", order.Status, domain.OrderStatusDelivering)
	}

	if _, err := svc.AdvanceDelivery(context.Background(), id, domain.OrderStatusDelivered); err != nil {
		t.Fatalf("second AdvanceDelivery() error = %v", err)
	}
	if _, err := svc.AdvanceDelivery(context.Background(), id, domain.OrderStatusDelivering); !errors.Is(err, domain.ErrInvalidStatusTransition) {
		t.Errorf("expected ErrInvalidStatusTransition after delivered, got %v", err)
	}
}
