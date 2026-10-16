// Package service содержит бизнес-логику: use case'ы поверх repository, ничего не знает про HTTP/SQL.
package service

import (
	"context"

	"venue-service/internal/domain"
)

// MenuSyncNotifier сообщает kitchen-api об изменении товара
// (PUT /internal/venues/{venueId}/menu/{itemId}) — тот же приём, что OrderNotifier в kitchen-api.
type MenuSyncNotifier interface {
	NotifyMenuItemChanged(ctx context.Context, item domain.MenuItem) error
}

// OrderStatusNotifier сообщает kitchen-api о смене статуса заказа
// (PATCH /internal/orders/{orderId}/status).
type OrderStatusNotifier interface {
	NotifyStatusChanged(ctx context.Context, order domain.Order) error
}
