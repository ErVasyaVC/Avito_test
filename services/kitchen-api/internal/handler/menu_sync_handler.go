package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"

	"kitchen-api/internal/domain"
	"kitchen-api/internal/service"
)

// MenuSyncHandler обслуживает вебхук синхронизации меню от venue-service.
type MenuSyncHandler struct {
	menu *service.MenuService
}

// NewMenuSyncHandler создаёт MenuSyncHandler поверх готового MenuService.
func NewMenuSyncHandler(menu *service.MenuService) *MenuSyncHandler {
	return &MenuSyncHandler{menu: menu}
}

type syncMenuItemRequest struct {
	Name        string     `json:"name"`
	Description *string    `json:"description"`
	Price       string     `json:"price"`
	IsAvailable bool       `json:"isAvailable"`
	RemovedAt   *time.Time `json:"removedAt"`
}

// Sync — PUT /internal/venues/{venueId}/menu/{itemId}, вебхук от venue-service.
func (h *MenuSyncHandler) Sync(w http.ResponseWriter, r *http.Request) {
	venueID, err := uuid.Parse(r.PathValue("venueId"))
	if err != nil {
		http.Error(w, "invalid venue id", http.StatusBadRequest)
		return
	}
	itemID, err := uuid.Parse(r.PathValue("itemId"))
	if err != nil {
		http.Error(w, "invalid item id", http.StatusBadRequest)
		return
	}

	var req syncMenuItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	item := domain.MenuItem{
		ID:          itemID,
		VenueID:     venueID,
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		IsAvailable: req.IsAvailable,
		RemovedAt:   req.RemovedAt,
	}

	if err := h.menu.Sync(r.Context(), item); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
