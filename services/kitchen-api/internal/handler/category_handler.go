// Package handler — HTTP-слой: разбор запроса, вызов service, сериализация ответа.
package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"kitchen-api/internal/service"
)

// CategoryHandler обслуживает ручки категорий.
type CategoryHandler struct {
	service *service.CategoryService
}

// NewCategoryHandler создаёт CategoryHandler поверх готового CategoryService.
func NewCategoryHandler(s *service.CategoryService) *CategoryHandler {
	return &CategoryHandler{service: s}
}

type categoryResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// List — GET /categories.
func (h *CategoryHandler) List(w http.ResponseWriter, r *http.Request) {
	categories, err := h.service.List(r.Context())
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	resp := make([]categoryResponse, 0, len(categories))
	for _, c := range categories {
		resp = append(resp, categoryResponse{ID: c.ID.String(), Name: c.Name})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("encode response: %v", err)
	}
}
