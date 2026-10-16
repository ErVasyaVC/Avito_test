// Package handler — HTTP-слой: разбор запроса, вызов service, сериализация ответа.
package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/google/uuid"

	"venue-service/internal/domain"
	"venue-service/internal/service"
)

// MenuHandler обслуживает ручки меню.
type MenuHandler struct {
	service *service.MenuService
}

// NewMenuHandler создаёт MenuHandler поверх готового MenuService.
func NewMenuHandler(s *service.MenuService) *MenuHandler {
	return &MenuHandler{service: s}
}

type menuItemResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Price       string `json:"price"`
	IsAvailable bool   `json:"isAvailable"`
}

func toMenuItemResponse(m domain.MenuItem) menuItemResponse {
	desc := ""
	if m.Description != nil {
		desc = *m.Description
	}
	return menuItemResponse{ID: m.ID.String(), Name: m.Name, Description: desc, Price: m.Price, IsAvailable: m.IsAvailable}
}

// List — GET /menu.
func (h *MenuHandler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.List(r.Context())
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	resp := make([]menuItemResponse, 0, len(items))
	for _, m := range items {
		resp = append(resp, toMenuItemResponse(m))
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("encode response: %v", err)
	}
}

type addMenuItemRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Price       string  `json:"price"`
}

// Add — POST /menu.
func (h *MenuHandler) Add(w http.ResponseWriter, r *http.Request) {
	var req addMenuItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Name == "" || req.Price == "" {
		http.Error(w, "name and price are required", http.StatusBadRequest)
		return
	}

	item, err := h.service.Create(r.Context(), service.CreateInput{
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
	})
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(toMenuItemResponse(item)); err != nil {
		log.Printf("encode response: %v", err)
	}
}

type updateMenuItemRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Price       *string `json:"price"`
	IsAvailable *bool   `json:"isAvailable"`
}

// Update — PATCH /menu/{itemId}.
func (h *MenuHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("itemId"))
	if err != nil {
		http.Error(w, "invalid item id", http.StatusBadRequest)
		return
	}

	var req updateMenuItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	item, err := h.service.Update(r.Context(), id, service.UpdateInput{
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		IsAvailable: req.IsAvailable,
	})
	switch {
	case errors.Is(err, domain.ErrMenuItemNotFound):
		http.Error(w, "menu item not found", http.StatusNotFound)
		return
	case err != nil:
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(toMenuItemResponse(item)); err != nil {
		log.Printf("encode response: %v", err)
	}
}

// Remove — DELETE /menu/{itemId}.
func (h *MenuHandler) Remove(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("itemId"))
	if err != nil {
		http.Error(w, "invalid item id", http.StatusBadRequest)
		return
	}

	err = h.service.Remove(r.Context(), id)
	switch {
	case errors.Is(err, domain.ErrMenuItemNotFound):
		http.Error(w, "menu item not found", http.StatusNotFound)
		return
	case err != nil:
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
