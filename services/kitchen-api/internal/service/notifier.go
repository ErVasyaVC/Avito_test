package service

import (
	"context"

	"kitchen-api/internal/domain"
)

// OrderNotifier abstracts venue notification so the delivery mechanism
// can change without touching order business logic.
type OrderNotifier interface {
	NotifyVenue(ctx context.Context, order domain.Order) error
}
