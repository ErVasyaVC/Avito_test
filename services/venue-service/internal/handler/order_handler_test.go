package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"venue-service/internal/domain"
	"venue-service/internal/service"
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

type fakeOrderStatusNotifier struct{}

func (fakeOrderStatusNotifier) NotifyStatusChanged(_ context.Context, _ domain.Order) error {
	return nil
}

func newTestOrderHandler(orders *fakeOrderRepo, menu *fakeAvailabilityChecker) *OrderHandler {
	return NewOrderHandler(service.NewOrderService(orders, menu, fakeOrderStatusNotifier{}))
}

func TestOrderHandler_Receive(t *testing.T) {
	itemID := uuid.New()
	menu := &fakeAvailabilityChecker{available: map[uuid.UUID]domain.MenuItem{
		itemID: {ID: itemID, Name: "Рамен", Price: "420.00", IsAvailable: true},
	}}
	h := newTestOrderHandler(newFakeOrderRepo(), menu)

	kitchenOrderID := uuid.New()
	body, _ := json.Marshal(receiveOrderRequest{
		KitchenOrderID: kitchenOrderID.String(),
		Items:          []orderItemInput{{ItemID: itemID.String(), Quantity: 2}},
	})

	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.Receive(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	var resp venueOrderResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Status != string(domain.OrderStatusAccepted) {
		t.Errorf("Status = %q, want %q", resp.Status, domain.OrderStatusAccepted)
	}
}

func TestOrderHandler_Receive_UnavailableItem(t *testing.T) {
	h := newTestOrderHandler(newFakeOrderRepo(), &fakeAvailabilityChecker{available: map[uuid.UUID]domain.MenuItem{}})

	body, _ := json.Marshal(receiveOrderRequest{
		KitchenOrderID: uuid.New().String(),
		Items:          []orderItemInput{{ItemID: uuid.New().String(), Quantity: 1}},
	})

	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.Receive(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusConflict, rec.Body.String())
	}
}

func TestOrderHandler_Receive_EmptyItems(t *testing.T) {
	h := newTestOrderHandler(newFakeOrderRepo(), &fakeAvailabilityChecker{})

	body, _ := json.Marshal(receiveOrderRequest{KitchenOrderID: uuid.New().String()})
	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.Receive(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestOrderHandler_UpdateStatus(t *testing.T) {
	orders := newFakeOrderRepo()
	kitchenOrderID := uuid.New()
	orders.orders[kitchenOrderID] = domain.Order{KitchenOrderID: kitchenOrderID, Status: domain.OrderStatusAccepted}
	h := newTestOrderHandler(orders, &fakeAvailabilityChecker{})

	body, _ := json.Marshal(updateOrderStatusRequest{Status: string(domain.OrderStatusCooking)})
	req := httptest.NewRequest(http.MethodPatch, "/orders/"+kitchenOrderID.String()+"/status", bytes.NewReader(body))
	req.SetPathValue("kitchenOrderId", kitchenOrderID.String())
	rec := httptest.NewRecorder()
	h.UpdateStatus(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestOrderHandler_UpdateStatus_NotFound(t *testing.T) {
	h := newTestOrderHandler(newFakeOrderRepo(), &fakeAvailabilityChecker{})

	missingID := uuid.New()
	body, _ := json.Marshal(updateOrderStatusRequest{Status: string(domain.OrderStatusCooking)})
	req := httptest.NewRequest(http.MethodPatch, "/orders/"+missingID.String()+"/status", bytes.NewReader(body))
	req.SetPathValue("kitchenOrderId", missingID.String())
	rec := httptest.NewRecorder()
	h.UpdateStatus(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestOrderHandler_UpdateStatus_InvalidTransition(t *testing.T) {
	orders := newFakeOrderRepo()
	kitchenOrderID := uuid.New()
	orders.orders[kitchenOrderID] = domain.Order{KitchenOrderID: kitchenOrderID, Status: domain.OrderStatusRejected}
	h := newTestOrderHandler(orders, &fakeAvailabilityChecker{})

	body, _ := json.Marshal(updateOrderStatusRequest{Status: string(domain.OrderStatusCooking)})
	req := httptest.NewRequest(http.MethodPatch, "/orders/"+kitchenOrderID.String()+"/status", bytes.NewReader(body))
	req.SetPathValue("kitchenOrderId", kitchenOrderID.String())
	rec := httptest.NewRecorder()
	h.UpdateStatus(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusConflict)
	}
}
