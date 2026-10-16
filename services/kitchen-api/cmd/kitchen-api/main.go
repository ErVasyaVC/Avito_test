// Command kitchen-api собирает зависимости (domain -> repository -> service ->
// handler) и запускает HTTP-сервер.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"kitchen-api/internal/handler"
	"kitchen-api/internal/repository"
	"kitchen-api/internal/service"
)

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {

	ctx := context.Background()

	connString := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_HOST"),
		os.Getenv("POSTGRES_PORT"),
		os.Getenv("POSTGRES_DB"),
	)

	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		log.Fatalf("connect to db: %v", err)
	}
	defer pool.Close()

	menuItemRepo := repository.NewMenuItemRepository(pool)

	categoryHandler := handler.NewCategoryHandler(
		service.NewCategoryService(
			repository.NewCategoryRepository(pool),
		),
	)

	venueService := service.NewVenueService(repository.NewVenueRepository(pool))
	venueHandler := handler.NewVenueHandler(venueService)

	menuService := service.NewMenuService(menuItemRepo)
	menuHandler := handler.NewMenuHandler(venueService, menuService)
	menuSyncHandler := handler.NewMenuSyncHandler(menuService)

	orderService := service.NewOrderService(
		repository.NewOrderRepository(pool),
		menuItemRepo,
		service.NewHTTPOrderNotifier(os.Getenv("VENUE_SERVICE_URL")),
	)
	orderHandler := handler.NewOrderHandler(orderService)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /categories", categoryHandler.List)
	mux.HandleFunc("GET /venues", venueHandler.List)
	mux.HandleFunc("GET /venues/{venueId}", venueHandler.GetByID)
	mux.HandleFunc("GET /venues/{venueId}/menu", menuHandler.GetByVenue)
	mux.HandleFunc("POST /orders", orderHandler.Create)
	mux.HandleFunc("GET /orders/{orderId}", orderHandler.GetByID)
	mux.HandleFunc("POST /orders/{orderId}/cancel", orderHandler.Cancel)
	mux.HandleFunc("PATCH /internal/orders/{orderId}/status", orderHandler.UpdateStatusFromVenue)
	mux.HandleFunc("PATCH /orders/{orderId}/status", orderHandler.AdvanceDelivery)
	mux.HandleFunc("PUT /internal/venues/{venueId}/menu/{itemId}", menuSyncHandler.Sync)

	log.Println("kitchen-api listening on :8080")
	server := &http.Server{
		Addr:         ":8080",
		Handler:      withCORS(mux),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	log.Fatal(server.ListenAndServe())

}
