package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/google/uuid"

	"kitchen-api/internal/service"
)

// MenuHandler обслуживает ручку меню заведения.
type MenuHandler struct {
	venues *service.VenueService
	menu   *service.MenuService
}

// NewMenuHandler создаёт MenuHandler поверх готовых VenueService и MenuService.
func NewMenuHandler(venues *service.VenueService, menu *service.MenuService) *MenuHandler {
	return &MenuHandler{venues: venues, menu: menu}
}

type menuItemResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Price       string `json:"price"`
	IsAvailable bool   `json:"isAvailable"`
}

// GetByVenue — GET /venues/{venueId}/menu.
func (h *MenuHandler) GetByVenue(w http.ResponseWriter, r *http.Request) {
	venueID, err := uuid.Parse(r.PathValue("venueId"))
	if err != nil {
		http.Error(w, "invalid venue id", http.StatusBadRequest)
		return
	}

	// Переиспользуем VenueService.GetByID только для проверки "заведение существует" (404).
	if _, err := h.venues.GetByID(r.Context(), venueID); err != nil {
		http.Error(w, "venue not found", http.StatusNotFound)
		return
	}

	items, err := h.menu.ListByVenue(r.Context(), venueID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	resp := make([]menuItemResponse, 0, len(items))
	for _, m := range items {
		desc := ""
		if m.Description != nil {
			desc = *m.Description
		}
		resp = append(resp, menuItemResponse{
			ID:          m.ID.String(),
			Name:        m.Name,
			Description: desc,
			Price:       m.Price,
			IsAvailable: m.IsAvailable,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("encode response: %v", err)
	}
}
