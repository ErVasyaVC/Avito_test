package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// OrderStatus — статус заказа на стороне заведения.
type OrderStatus string

// Статусы заказа.
const (
	OrderStatusAccepted         OrderStatus = "accepted"
	OrderStatusRejected         OrderStatus = "rejected"
	OrderStatusCooking          OrderStatus = "cooking"
	OrderStatusReady            OrderStatus = "ready"
	OrderStatusCancelledByVenue OrderStatus = "cancelled_by_venue"
	OrderStatusCancelledByUser  OrderStatus = "cancelled_by_user"
)

// Ошибки заказа.
var (
	ErrOrderNotFound           = errors.New("order not found")
	ErrInvalidStatusTransition = errors.New("invalid order status transition")
)

// OrderItem — позиция заказа со снимком названия/цены на момент приёма.
type OrderItem struct {
	MenuItemID    uuid.UUID
	NameSnapshot  string
	PriceSnapshot string
	Quantity      int
}

// Order — заказ, принятый от kitchen-api. KitchenOrderID — единственный ID заказа,
// своего публичного ID у venue-service нет (см. docs/db-schema/venue.puml).
type Order struct {
	ID             uuid.UUID
	KitchenOrderID uuid.UUID
	Status         OrderStatus
	Items          []OrderItem
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// CanTransitionTo — допустимые переходы статуса.
func (o Order) CanTransitionTo(next OrderStatus) bool {
	switch o.Status {
	case OrderStatusAccepted:
		return next == OrderStatusCooking || next == OrderStatusCancelledByUser
	case OrderStatusCooking:
		return next == OrderStatusReady || next == OrderStatusCancelledByVenue
	default:
		return false
	}
}
