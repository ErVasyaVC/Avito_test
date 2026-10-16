package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"venue-service/internal/domain"
)

// HTTPKitchenNotifier уведомляет kitchen-api об изменениях меню и статуса заказа
// прямыми HTTP-запросами к его вебхукам (см. api/kitchen-openapi.yaml). Реализует
// одновременно MenuSyncNotifier и OrderStatusNotifier — обоим нужен один и тот же
// HTTP-клиент до kitchen-api.
type HTTPKitchenNotifier struct {
	baseURL string
	venueID string
	client  *http.Client
}

// NewHTTPKitchenNotifier создаёт уведомитель поверх HTTP-клиента с адресом kitchen-api
// и ID этого заведения в системе kitchen-api (venueID — своего ID у venue-service нет).
func NewHTTPKitchenNotifier(baseURL, venueID string) *HTTPKitchenNotifier {
	return &HTTPKitchenNotifier{
		baseURL: baseURL,
		venueID: venueID,
		client:  &http.Client{Timeout: 5 * time.Second},
	}
}

type syncMenuItemRequest struct {
	Name        string     `json:"name"`
	Description *string    `json:"description"`
	Price       string     `json:"price"`
	IsAvailable bool       `json:"isAvailable"`
	RemovedAt   *time.Time `json:"removedAt"`
}

// NotifyMenuItemChanged — PUT /internal/venues/{venueId}/menu/{itemId} на kitchen-api.
func (n *HTTPKitchenNotifier) NotifyMenuItemChanged(ctx context.Context, item domain.MenuItem) error {
	body, err := json.Marshal(syncMenuItemRequest{
		Name:        item.Name,
		Description: item.Description,
		Price:       item.Price,
		IsAvailable: item.IsAvailable,
		RemovedAt:   item.RemovedAt,
	})
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s/internal/venues/%s/menu/%s", n.baseURL, n.venueID, item.ID.String())
	return n.send(ctx, http.MethodPut, url, body)
}

type updateOrderStatusRequest struct {
	Status string `json:"status"`
}

// NotifyStatusChanged — PATCH /internal/orders/{orderId}/status на kitchen-api.
func (n *HTTPKitchenNotifier) NotifyStatusChanged(ctx context.Context, order domain.Order) error {
	body, err := json.Marshal(updateOrderStatusRequest{Status: string(order.Status)})
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s/internal/orders/%s/status", n.baseURL, order.KitchenOrderID.String())
	return n.send(ctx, http.MethodPatch, url, body)
}

func (n *HTTPKitchenNotifier) send(ctx context.Context, method, url string, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("kitchen-api responded with status %d", resp.StatusCode)
	}
	return nil
}
