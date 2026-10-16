package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"venue-service/internal/domain"
)

// OrderRepository — то, что нужно OrderService от хранилища заказов.
type OrderRepository interface {
	Create(ctx context.Context, order domain.Order) error
	GetByKitchenOrderID(ctx context.Context, kitchenOrderID uuid.UUID) (domain.Order, error)
	UpdateStatus(ctx context.Context, kitchenOrderID uuid.UUID, status domain.OrderStatus) error
}

// AvailabilityChecker — узкий интерфейс, нужен OrderService только для проверки
// доступности товаров при приёме заказа (не весь MenuItemRepository).
type AvailabilityChecker interface {
	GetAvailableByIDs(ctx context.Context, ids []uuid.UUID) ([]domain.MenuItem, error)
}

// ReceiveOrderItem — одна позиция в запросе на приём заказа.
type ReceiveOrderItem struct {
	MenuItemID uuid.UUID
	Quantity   int
}

// UnavailableItemsError — часть или все позиции заказа недоступны на момент приёма.
type UnavailableItemsError struct {
	ItemIDs []uuid.UUID
}

// Error реализует интерфейс error.
func (e *UnavailableItemsError) Error() string {
	return fmt.Sprintf("%d item(s) unavailable", len(e.ItemIDs))
}

// OrderService — бизнес-логика заказов на стороне заведения.
type OrderService struct {
	orders   OrderRepository
	menu     AvailabilityChecker
	notifier OrderStatusNotifier
}

// NewOrderService создаёт OrderService поверх готовых зависимостей.
func NewOrderService(orders OrderRepository, menu AvailabilityChecker, notifier OrderStatusNotifier) *OrderService {
	return &OrderService{orders: orders, menu: menu, notifier: notifier}
}

// Receive проверяет доступность каждой позиции и либо принимает заказ (status=accepted),
// либо отклоняет — *UnavailableItemsError, не сохраняя ничего.
func (s *OrderService) Receive(ctx context.Context, kitchenOrderID uuid.UUID, items []ReceiveOrderItem) (domain.Order, error) {
	ids := make([]uuid.UUID, len(items))
	for i, it := range items {
		ids[i] = it.MenuItemID
	}

	available, err := s.menu.GetAvailableByIDs(ctx, ids)
	if err != nil {
		return domain.Order{}, err
	}
	byID := make(map[uuid.UUID]domain.MenuItem, len(available))
	for _, m := range available {
		byID[m.ID] = m
	}

	var unavailable []uuid.UUID
	orderItems := make([]domain.OrderItem, 0, len(items))
	for _, it := range items {
		m, ok := byID[it.MenuItemID]
		if !ok {
			unavailable = append(unavailable, it.MenuItemID)
			continue
		}
		orderItems = append(orderItems, domain.OrderItem{
			MenuItemID:    m.ID,
			NameSnapshot:  m.Name,
			PriceSnapshot: m.Price,
			Quantity:      it.Quantity,
		})
	}

	if len(unavailable) > 0 {
		return domain.Order{}, &UnavailableItemsError{ItemIDs: unavailable}
	}

	now := time.Now()
	order := domain.Order{
		ID:             uuid.New(),
		KitchenOrderID: kitchenOrderID,
		Status:         domain.OrderStatusAccepted,
		Items:          orderItems,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.orders.Create(ctx, order); err != nil {
		return domain.Order{}, err
	}
	return order, nil
}

// UpdateStatus меняет статус заказа. Уведомляет kitchen-api, только если статус
// сменился по инициативе заведения (cooking/ready/cancelled_by_venue) — cancelled_by_user
// приходит от самого kitchen-api (пользователь отменил заказ), эхом отвечать не нужно.
func (s *OrderService) UpdateStatus(ctx context.Context, kitchenOrderID uuid.UUID, status domain.OrderStatus) (domain.Order, error) {
	order, err := s.orders.GetByKitchenOrderID(ctx, kitchenOrderID)
	if err != nil {
		return domain.Order{}, err
	}
	if !order.CanTransitionTo(status) {
		return domain.Order{}, domain.ErrInvalidStatusTransition
	}
	if err := s.orders.UpdateStatus(ctx, kitchenOrderID, status); err != nil {
		return domain.Order{}, err
	}
	order.Status = status
	order.UpdatedAt = time.Now()

	if status == domain.OrderStatusCancelledByUser {
		return order, nil
	}

	if err := s.notifier.NotifyStatusChanged(ctx, order); err != nil {
		return domain.Order{}, err
	}
	return order, nil
}
