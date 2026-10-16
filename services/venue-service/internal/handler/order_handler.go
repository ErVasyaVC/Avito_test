package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"

	"venue-service/internal/domain"
	"venue-service/internal/service"
)

// OrderHandler обслуживает ручки заказов.
type OrderHandler struct {
	service *service.OrderService
}

// NewOrderHandler создаёт OrderHandler поверх готового OrderService.
func NewOrderHandler(s *service.OrderService) *OrderHandler {
	return &OrderHandler{service: s}
}

type orderItemInput struct {
	ItemID   string `json:"itemId"`
	Quantity int    `json:"quantity"`
}

type receiveOrderRequest struct {
	KitchenOrderID string           `json:"kitchenOrderId"`
	Items          []orderItemInput `json:"items"`
	Comment        *string          `json:"comment"`
}

type venueOrderResponse struct {
	KitchenOrderID string    `json:"kitchenOrderId"`
	Status         string    `json:"status"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

func toVenueOrderResponse(o domain.Order) venueOrderResponse {
	return venueOrderResponse{KitchenOrderID: o.KitchenOrderID.String(), Status: string(o.Status), UpdatedAt: o.UpdatedAt}
}

type unavailableItemsErrorResponse struct {
	Message            string   `json:"message"`
	UnavailableItemIDs []string `json:"unavailableItemIds"`
}

// Receive — POST /orders.
func (h *OrderHandler) Receive(w http.ResponseWriter, r *http.Request) {
	var req receiveOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	kitchenOrderID, err := uuid.Parse(req.KitchenOrderID)
	if err != nil {
		http.Error(w, "invalid kitchenOrderId", http.StatusBadRequest)
		return
	}
	if len(req.Items) == 0 {
		http.Error(w, "items must not be empty", http.StatusBadRequest)
		return
	}

	items := make([]service.ReceiveOrderItem, 0, len(req.Items))
	for _, it := range req.Items {
		itemID, err := uuid.Parse(it.ItemID)
		if err != nil {
			http.Error(w, "invalid itemId", http.StatusBadRequest)
			return
		}
		if it.Quantity < 1 {
			http.Error(w, "quantity must be positive", http.StatusBadRequest)
			return
		}
		items = append(items, service.ReceiveOrderItem{MenuItemID: itemID, Quantity: it.Quantity})
	}

	order, err := h.service.Receive(r.Context(), kitchenOrderID, items)

	var unavailableErr *service.UnavailableItemsError
	switch {
	case errors.As(err, &unavailableErr):
		ids := make([]string, len(unavailableErr.ItemIDs))
		for i, id := range unavailableErr.ItemIDs {
			ids[i] = id.String()
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		if err := json.NewEncoder(w).Encode(unavailableItemsErrorResponse{
			Message:            "some items are unavailable",
			UnavailableItemIDs: ids,
		}); err != nil {
			log.Printf("encode response: %v", err)
		}
		return
	case err != nil:
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(toVenueOrderResponse(order)); err != nil {
		log.Printf("encode response: %v", err)
	}
}

type updateOrderStatusRequest struct {
	Status string `json:"status"`
}

// UpdateStatus — PATCH /orders/{kitchenOrderId}/status.
func (h *OrderHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	kitchenOrderID, err := uuid.Parse(r.PathValue("kitchenOrderId"))
	if err != nil {
		http.Error(w, "invalid kitchen order id", http.StatusBadRequest)
		return
	}

	var req updateOrderStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	order, err := h.service.UpdateStatus(r.Context(), kitchenOrderID, domain.OrderStatus(req.Status))
	switch {
	case errors.Is(err, domain.ErrOrderNotFound):
		http.Error(w, "order not found", http.StatusNotFound)
		return
	case errors.Is(err, domain.ErrInvalidStatusTransition):
		http.Error(w, "invalid status transition", http.StatusConflict)
		return
	case err != nil:
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(toVenueOrderResponse(order)); err != nil {
		log.Printf("encode response: %v", err)
	}
}
