// Command venue-service собирает зависимости (domain -> repository -> service ->
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

	"venue-service/internal/handler"
	"venue-service/internal/repository"
	"venue-service/internal/service"
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

	kitchenNotifier := service.NewHTTPKitchenNotifier(os.Getenv("KITCHEN_API_URL"), os.Getenv("KITCHEN_VENUE_ID"))

	menuService := service.NewMenuService(menuItemRepo, kitchenNotifier)
	menuHandler := handler.NewMenuHandler(menuService)

	orderService := service.NewOrderService(
		repository.NewOrderRepository(pool),
		menuItemRepo,
		kitchenNotifier,
	)
	orderHandler := handler.NewOrderHandler(orderService)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /menu", menuHandler.List)
	mux.HandleFunc("POST /menu", menuHandler.Add)
	mux.HandleFunc("PATCH /menu/{itemId}", menuHandler.Update)
	mux.HandleFunc("DELETE /menu/{itemId}", menuHandler.Remove)
	mux.HandleFunc("POST /orders", orderHandler.Receive)
	mux.HandleFunc("PATCH /orders/{kitchenOrderId}/status", orderHandler.UpdateStatus)

	log.Println("venue-service listening on :8081")
	server := &http.Server{
		Addr:         ":8081",
		Handler:      withCORS(mux),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	log.Fatal(server.ListenAndServe())
}
