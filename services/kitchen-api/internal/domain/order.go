package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// OrderStatus — статус заказа, см. docs/order-status-lifecycle.puml.
type OrderStatus string

// Статусы заказа.
const (
	OrderStatusCreated          OrderStatus = "created"
	OrderStatusAccepted         OrderStatus = "accepted"
	OrderStatusRejected         OrderStatus = "rejected"
	OrderStatusCancelledByUser  OrderStatus = "cancelled_by_user"
	OrderStatusCooking          OrderStatus = "cooking"
	OrderStatusCancelledByVenue OrderStatus = "cancelled_by_venue"
	OrderStatusReady            OrderStatus = "ready"
	OrderStatusDelivering       OrderStatus = "delivering"
	OrderStatusDelivered        OrderStatus = "delivered"
)

// Ошибки заказа.
var (
	ErrOrderNotFound           = errors.New("order not found")
	ErrOrderCancelNotAllowed   = errors.New("order cannot be cancelled in its current status")
	ErrInvalidStatusTransition = errors.New("invalid order status transition")
)

// OrderItem — позиция заказа со снимком названия/цены на момент оформления.
type OrderItem struct {
	MenuItemID    uuid.UUID
	NameSnapshot  string
	PriceSnapshot string
	Quantity      int
}

// Order — заказ пользователя.
type Order struct {
	ID              uuid.UUID
	VenueID         uuid.UUID
	Status          OrderStatus
	CustomerName    string
	CustomerPhone   string
	DeliveryAddress string
	Comment         *string
	Items           []OrderItem
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// CanBeCancelledByUser — отменить можно, только пока заведение не начало готовить.
func (o Order) CanBeCancelledByUser() bool {
	switch o.Status {
	case OrderStatusCreated, OrderStatusAccepted:
		return true
	default:
		return false
	}
}

// CanTransitionTo — допустимые переходы статуса по вебхуку от venue-service.
func (o Order) CanTransitionTo(next OrderStatus) bool {
	switch o.Status {
	case OrderStatusCreated, OrderStatusAccepted:
		return next == OrderStatusCooking
	case OrderStatusCooking:
		return next == OrderStatusReady || next == OrderStatusCancelledByVenue
	default:
		return false
	}
}

// CanAdvanceDelivery — переходы, которые двигает сам kitchen-api (нет реального
// курьера, это его "заглушка" за доставку), а не заведение и не пользователь.
func (o Order) CanAdvanceDelivery(next OrderStatus) bool {
	switch o.Status {
	case OrderStatusReady:
		return next == OrderStatusDelivering
	case OrderStatusDelivering:
		return next == OrderStatusDelivered
	default:
		return false
	}
}
