package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"

	"kitchen-api/internal/domain"
	"kitchen-api/internal/service"
)

// OrderHandler обслуживает ручки заказов.
type OrderHandler struct {
	service *service.OrderService
}

// NewOrderHandler создаёт OrderHandler поверх готового OrderService.
func NewOrderHandler(s *service.OrderService) *OrderHandler {
	return &OrderHandler{service: s}
}

type orderItemRequest struct {
	MenuItemID string `json:"menuItemId"`
	Quantity   int    `json:"quantity"`
}

type createOrderRequest struct {
	VenueID         string             `json:"venueId"`
	Items           []orderItemRequest `json:"items"`
	CustomerName    string             `json:"customerName"`
	CustomerPhone   string             `json:"customerPhone"`
	DeliveryAddress string             `json:"deliveryAddress"`
	Comment         *string            `json:"comment"`
}

type orderItemResponse struct {
	MenuItemID string `json:"menuItemId"`
	Name       string `json:"name"`
	Price      string `json:"price"`
	Quantity   int    `json:"quantity"`
}

type orderResponse struct {
	ID              string              `json:"id"`
	VenueID         string              `json:"venueId"`
	Status          string              `json:"status"`
	Items           []orderItemResponse `json:"items"`
	CustomerName    string              `json:"customerName"`
	CustomerPhone   string              `json:"customerPhone"`
	DeliveryAddress string              `json:"deliveryAddress"`
	Comment         *string             `json:"comment"`
	CreatedAt       time.Time           `json:"createdAt"`
	UpdatedAt       time.Time           `json:"updatedAt"`
}

type unavailableItemsErrorResponse struct {
	Message            string   `json:"message"`
	UnavailableItemIDs []string `json:"unavailableItemIds"`
}

func toOrderResponse(o domain.Order) orderResponse {
	items := make([]orderItemResponse, 0, len(o.Items))
	for _, it := range o.Items {
		items = append(items, orderItemResponse{
			MenuItemID: it.MenuItemID.String(),
			Name:       it.NameSnapshot,
			Price:      it.PriceSnapshot,
			Quantity:   it.Quantity,
		})
	}
	return orderResponse{
		ID:              o.ID.String(),
		VenueID:         o.VenueID.String(),
		Status:          string(o.Status),
		Items:           items,
		CustomerName:    o.CustomerName,
		CustomerPhone:   o.CustomerPhone,
		DeliveryAddress: o.DeliveryAddress,
		Comment:         o.Comment,
		CreatedAt:       o.CreatedAt,
		UpdatedAt:       o.UpdatedAt,
	}
}

// Create — POST /orders.
func (h *OrderHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	venueID, err := uuid.Parse(req.VenueID)
	if err != nil {
		http.Error(w, "invalid venueId", http.StatusBadRequest)
		return
	}
	if len(req.Items) == 0 {
		http.Error(w, "items must not be empty", http.StatusBadRequest)
		return
	}

	items := make([]service.CreateOrderItem, 0, len(req.Items))
	for _, it := range req.Items {
		itemID, err := uuid.Parse(it.MenuItemID)
		if err != nil {
			http.Error(w, "invalid menuItemId", http.StatusBadRequest)
			return
		}
		if it.Quantity < 1 {
			http.Error(w, "quantity must be positive", http.StatusBadRequest)
			return
		}
		items = append(items, service.CreateOrderItem{MenuItemID: itemID, Quantity: it.Quantity})
	}

	order, err := h.service.Create(r.Context(), service.CreateOrderInput{
		VenueID:         venueID,
		Items:           items,
		CustomerName:    req.CustomerName,
		CustomerPhone:   req.CustomerPhone,
		DeliveryAddress: req.DeliveryAddress,
		Comment:         req.Comment,
	})

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
	if err := json.NewEncoder(w).Encode(toOrderResponse(order)); err != nil {
		log.Printf("encode response: %v", err)
	}
}

// GetByID — GET /orders/{orderId}.
func (h *OrderHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("orderId"))
	if err != nil {
		http.Error(w, "invalid order id", http.StatusBadRequest)
		return
	}

	order, err := h.service.GetByID(r.Context(), id)
	switch {
	case errors.Is(err, domain.ErrOrderNotFound):
		http.Error(w, "order not found", http.StatusNotFound)
		return
	case err != nil:
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(toOrderResponse(order)); err != nil {
		log.Printf("encode response: %v", err)
	}
}

// Cancel — POST /orders/{orderId}/cancel.
func (h *OrderHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("orderId"))
	if err != nil {
		http.Error(w, "invalid order id", http.StatusBadRequest)
		return
	}

	order, err := h.service.Cancel(r.Context(), id)
	switch {
	case errors.Is(err, domain.ErrOrderNotFound):
		http.Error(w, "order not found", http.StatusNotFound)
		return
	case errors.Is(err, domain.ErrOrderCancelNotAllowed):
		http.Error(w, "order cannot be cancelled in its current status", http.StatusConflict)
		return
	case err != nil:
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(toOrderResponse(order)); err != nil {
		log.Printf("encode response: %v", err)
	}
}

type updateOrderStatusRequest struct {
	Status string `json:"status"`
}

// UpdateStatusFromVenue — PATCH /internal/orders/{orderId}/status, вебхук от venue-service.
func (h *OrderHandler) UpdateStatusFromVenue(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("orderId"))
	if err != nil {
		http.Error(w, "invalid order id", http.StatusBadRequest)
		return
	}

	var req updateOrderStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	order, err := h.service.UpdateStatusFromVenue(r.Context(), id, domain.OrderStatus(req.Status))
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
	if err := json.NewEncoder(w).Encode(toOrderResponse(order)); err != nil {
		log.Printf("encode response: %v", err)
	}
}

// AdvanceDelivery — PATCH /orders/{orderId}/status: ready -> delivering -> delivered.
func (h *OrderHandler) AdvanceDelivery(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("orderId"))
	if err != nil {
		http.Error(w, "invalid order id", http.StatusBadRequest)
		return
	}

	var req updateOrderStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	order, err := h.service.AdvanceDelivery(r.Context(), id, domain.OrderStatus(req.Status))
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
	if err := json.NewEncoder(w).Encode(toOrderResponse(order)); err != nil {
		log.Printf("encode response: %v", err)
	}
}
