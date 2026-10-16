package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"kitchen-api/internal/domain"
	"kitchen-api/internal/service"
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

type fakeOrderNotifier struct{}

func (fakeOrderNotifier) NotifyVenue(_ context.Context, _ domain.Order) error { return nil }

func newTestOrderHandler(orders *fakeOrderRepo, menu *fakeAvailabilityChecker) *OrderHandler {
	return NewOrderHandler(service.NewOrderService(orders, menu, fakeOrderNotifier{}))
}

func TestOrderHandler_Create(t *testing.T) {
	venueID := uuid.New()
	itemID := uuid.New()
	menu := &fakeAvailabilityChecker{available: map[uuid.UUID]domain.MenuItem{
		itemID: {ID: itemID, Name: "Паста", Price: "390.00", IsAvailable: true},
	}}
	h := newTestOrderHandler(newFakeOrderRepo(), menu)

	body, _ := json.Marshal(createOrderRequest{
		VenueID:         venueID.String(),
		Items:           []orderItemRequest{{MenuItemID: itemID.String(), Quantity: 2}},
		CustomerName:    "Тест",
		CustomerPhone:   "+70000000000",
		DeliveryAddress: "адрес",
	})

	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	var resp orderResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Status != string(domain.OrderStatusAccepted) {
		t.Errorf("Status = %q, want %q", resp.Status, domain.OrderStatusAccepted)
	}
}

func TestOrderHandler_Create_UnavailableItem(t *testing.T) {
	venueID := uuid.New()
	itemID := uuid.New()
	h := newTestOrderHandler(newFakeOrderRepo(), &fakeAvailabilityChecker{available: map[uuid.UUID]domain.MenuItem{}})

	body, _ := json.Marshal(createOrderRequest{
		VenueID:         venueID.String(),
		Items:           []orderItemRequest{{MenuItemID: itemID.String(), Quantity: 1}},
		CustomerName:    "Тест",
		CustomerPhone:   "+70000000000",
		DeliveryAddress: "адрес",
	})

	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.Create(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusConflict, rec.Body.String())
	}
}

func TestOrderHandler_Create_EmptyItems(t *testing.T) {
	h := newTestOrderHandler(newFakeOrderRepo(), &fakeAvailabilityChecker{})

	body, _ := json.Marshal(createOrderRequest{
		VenueID: uuid.New().String(), CustomerName: "Тест", CustomerPhone: "+7", DeliveryAddress: "адрес",
	})
	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestOrderHandler_Create_InvalidBody(t *testing.T) {
	h := newTestOrderHandler(newFakeOrderRepo(), &fakeAvailabilityChecker{})

	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader([]byte("not json")))
	rec := httptest.NewRecorder()
	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestOrderHandler_GetByID(t *testing.T) {
	orders := newFakeOrderRepo()
	orderID := uuid.New()
	orders.orders[orderID] = domain.Order{ID: orderID, Status: domain.OrderStatusCreated}
	h := newTestOrderHandler(orders, &fakeAvailabilityChecker{})

	req := httptest.NewRequest(http.MethodGet, "/orders/"+orderID.String(), nil)
	req.SetPathValue("orderId", orderID.String())
	rec := httptest.NewRecorder()
	h.GetByID(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestOrderHandler_GetByID_NotFound(t *testing.T) {
	h := newTestOrderHandler(newFakeOrderRepo(), &fakeAvailabilityChecker{})

	missingID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/orders/"+missingID.String(), nil)
	req.SetPathValue("orderId", missingID.String())
	rec := httptest.NewRecorder()
	h.GetByID(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestOrderHandler_Cancel(t *testing.T) {
	orders := newFakeOrderRepo()
	orderID := uuid.New()
	orders.orders[orderID] = domain.Order{ID: orderID, Status: domain.OrderStatusCreated}
	h := newTestOrderHandler(orders, &fakeAvailabilityChecker{})

	req := httptest.NewRequest(http.MethodPost, "/orders/"+orderID.String()+"/cancel", nil)
	req.SetPathValue("orderId", orderID.String())
	rec := httptest.NewRecorder()
	h.Cancel(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestOrderHandler_Cancel_Conflict(t *testing.T) {
	orders := newFakeOrderRepo()
	orderID := uuid.New()
	orders.orders[orderID] = domain.Order{ID: orderID, Status: domain.OrderStatusCooking}
	h := newTestOrderHandler(orders, &fakeAvailabilityChecker{})

	req := httptest.NewRequest(http.MethodPost, "/orders/"+orderID.String()+"/cancel", nil)
	req.SetPathValue("orderId", orderID.String())
	rec := httptest.NewRecorder()
	h.Cancel(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusConflict)
	}
}

func TestOrderHandler_UpdateStatusFromVenue(t *testing.T) {
	orders := newFakeOrderRepo()
	orderID := uuid.New()
	orders.orders[orderID] = domain.Order{ID: orderID, Status: domain.OrderStatusAccepted}
	h := newTestOrderHandler(orders, &fakeAvailabilityChecker{})

	body, _ := json.Marshal(updateOrderStatusRequest{Status: string(domain.OrderStatusCooking)})
	req := httptest.NewRequest(http.MethodPatch, "/internal/orders/"+orderID.String()+"/status", bytes.NewReader(body))
	req.SetPathValue("orderId", orderID.String())
	rec := httptest.NewRecorder()
	h.UpdateStatusFromVenue(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestOrderHandler_UpdateStatusFromVenue_InvalidTransition(t *testing.T) {
	orders := newFakeOrderRepo()
	orderID := uuid.New()
	orders.orders[orderID] = domain.Order{ID: orderID, Status: domain.OrderStatusDelivered}
	h := newTestOrderHandler(orders, &fakeAvailabilityChecker{})

	body, _ := json.Marshal(updateOrderStatusRequest{Status: string(domain.OrderStatusCooking)})
	req := httptest.NewRequest(http.MethodPatch, "/internal/orders/"+orderID.String()+"/status", bytes.NewReader(body))
	req.SetPathValue("orderId", orderID.String())
	rec := httptest.NewRecorder()
	h.UpdateStatusFromVenue(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusConflict)
	}
}

func TestOrderHandler_AdvanceDelivery(t *testing.T) {
	orders := newFakeOrderRepo()
	orderID := uuid.New()
	orders.orders[orderID] = domain.Order{ID: orderID, Status: domain.OrderStatusReady}
	h := newTestOrderHandler(orders, &fakeAvailabilityChecker{})

	body, _ := json.Marshal(updateOrderStatusRequest{Status: string(domain.OrderStatusDelivering)})
	req := httptest.NewRequest(http.MethodPatch, "/orders/"+orderID.String()+"/status", bytes.NewReader(body))
	req.SetPathValue("orderId", orderID.String())
	rec := httptest.NewRecorder()
	h.AdvanceDelivery(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestOrderHandler_AdvanceDelivery_InvalidTransition(t *testing.T) {
	orders := newFakeOrderRepo()
	orderID := uuid.New()
	orders.orders[orderID] = domain.Order{ID: orderID, Status: domain.OrderStatusCreated}
	h := newTestOrderHandler(orders, &fakeAvailabilityChecker{})

	body, _ := json.Marshal(updateOrderStatusRequest{Status: string(domain.OrderStatusDelivering)})
	req := httptest.NewRequest(http.MethodPatch, "/orders/"+orderID.String()+"/status", bytes.NewReader(body))
	req.SetPathValue("orderId", orderID.String())
	rec := httptest.NewRecorder()
	h.AdvanceDelivery(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusConflict)
	}
}
