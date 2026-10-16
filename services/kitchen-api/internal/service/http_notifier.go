package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"kitchen-api/internal/domain"
)

// httpOrderNotifier уведомляет venue-service о заказах прямыми HTTP-запросами
// к его API (см. api/venue-openapi.yaml).
type httpOrderNotifier struct {
	baseURL string
	client  *http.Client
}

// NewHTTPOrderNotifier создаёт OrderNotifier поверх HTTP-клиента с адресом venue-service.
func NewHTTPOrderNotifier(baseURL string) OrderNotifier {
	return &httpOrderNotifier{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 5 * time.Second},
	}
}

type venueOrderItemRequest struct {
	ItemID   string `json:"itemId"`
	Quantity int    `json:"quantity"`
}

type venueReceiveOrderRequest struct {
	KitchenOrderID string                  `json:"kitchenOrderId"`
	Items          []venueOrderItemRequest `json:"items"`
	Comment        *string                 `json:"comment"`
}

type venueUpdateStatusRequest struct {
	Status string `json:"status"`
}

// NotifyVenue сообщает venue-service о заказе: новый заказ — POST /orders,
// отмена пользователем — PATCH /orders/{kitchenOrderId}/status.
func (n *httpOrderNotifier) NotifyVenue(ctx context.Context, order domain.Order) error {
	if order.Status == domain.OrderStatusCancelledByUser {
		return n.updateStatus(ctx, order)
	}
	return n.receiveOrder(ctx, order)
}

func (n *httpOrderNotifier) receiveOrder(ctx context.Context, order domain.Order) error {
	items := make([]venueOrderItemRequest, len(order.Items))
	for i, it := range order.Items {
		items[i] = venueOrderItemRequest{ItemID: it.MenuItemID.String(), Quantity: it.Quantity}
	}
	body, err := json.Marshal(venueReceiveOrderRequest{
		KitchenOrderID: order.ID.String(),
		Items:          items,
		Comment:        order.Comment,
	})
	if err != nil {
		return err
	}
	return n.send(ctx, http.MethodPost, n.baseURL+"/orders", body)
}

func (n *httpOrderNotifier) updateStatus(ctx context.Context, order domain.Order) error {
	body, err := json.Marshal(venueUpdateStatusRequest{Status: string(order.Status)})
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s/orders/%s/status", n.baseURL, order.ID.String())
	return n.send(ctx, http.MethodPatch, url, body)
}

func (n *httpOrderNotifier) send(ctx context.Context, method, url string, body []byte) error {
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
		return fmt.Errorf("venue-service responded with status %d", resp.StatusCode)
	}
	return nil
}
