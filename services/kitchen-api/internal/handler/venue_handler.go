package handler

import (
	"encoding/json"
	"errors"
	"kitchen-api/internal/domain"
	"log"
	"net/http"

	"github.com/google/uuid"

	"kitchen-api/internal/service"
)

// VenueHandler обслуживает ручки заведений.
type VenueHandler struct {
	service *service.VenueService
}

// NewVenueHandler создаёт VenueHandler поверх готового VenueService.
func NewVenueHandler(s *service.VenueService) *VenueHandler {
	return &VenueHandler{service: s}
}

type venueResponse struct {
	ID         string             `json:"id"`
	Name       string             `json:"name"`
	Categories []categoryResponse `json:"categories"`
}

// List — GET /venues, опционально ?category_id=.
func (h *VenueHandler) List(w http.ResponseWriter, r *http.Request) {
	var categoryID *uuid.UUID
	if raw := r.URL.Query().Get("category_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			http.Error(w, "invalid category_id", http.StatusBadRequest)
			return
		}
		categoryID = &id
	}

	venues, err := h.service.List(r.Context(), categoryID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	resp := make([]venueResponse, 0, len(venues))
	for _, v := range venues {
		categories := make([]categoryResponse, 0, len(v.Categories))
		for _, c := range v.Categories {
			categories = append(categories, categoryResponse{ID: c.ID.String(), Name: c.Name})
		}
		resp = append(resp, venueResponse{ID: v.ID.String(), Name: v.Name, Categories: categories})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("encode response: %v", err)
	}
}

// GetByID — GET /venues/{venueId}.
func (h *VenueHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("venueId"))
	if err != nil {
		http.Error(w, "invalid venue id", http.StatusBadRequest)
		return
	}

	v, err := h.service.GetByID(r.Context(), id)
	switch {
	case errors.Is(err, domain.ErrVenueNotFound):
		http.Error(w, "venue not found", http.StatusNotFound)
		return
	case err != nil:
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	categories := make([]categoryResponse, 0, len(v.Categories))
	for _, c := range v.Categories {
		categories = append(categories, categoryResponse{ID: c.ID.String(), Name: c.Name})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(venueResponse{ID: v.ID.String(), Name: v.Name, Categories: categories}); err != nil {
		log.Printf("encode response: %v", err)
	}
}
