package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"kitchen-api/internal/domain"
)

// OrderRepository — то, что нужно OrderService от хранилища заказов.
type OrderRepository interface {
	Create(ctx context.Context, order domain.Order) error
	GetByID(ctx context.Context, id uuid.UUID) (domain.Order, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.OrderStatus) error
}

// AvailabilityChecker — узкий интерфейс, нужен OrderService только для проверки
// доступности товаров при оформлении заказа (не весь MenuItemRepository).
type AvailabilityChecker interface {
	GetAvailableByIDs(ctx context.Context, venueID uuid.UUID, itemIDs []uuid.UUID) ([]domain.MenuItem, error)
}

// CreateOrderItem — одна позиция в запросе на создание заказа.
type CreateOrderItem struct {
	MenuItemID uuid.UUID
	Quantity   int
}

// CreateOrderInput — вход для OrderService.Create.
type CreateOrderInput struct {
	VenueID         uuid.UUID
	Items           []CreateOrderItem
	CustomerName    string
	CustomerPhone   string
	DeliveryAddress string
	Comment         *string
}

// UnavailableItemsError — часть или все позиции заказа стали недоступны
// между тем, как пользователь положил их в корзину, и оформлением заказа.
type UnavailableItemsError struct {
	ItemIDs []uuid.UUID
}

// Error реализует интерфейс error.
func (e *UnavailableItemsError) Error() string {
	return fmt.Sprintf("%d item(s) unavailable", len(e.ItemIDs))
}

// OrderService — бизнес-логика заказов: создание, отмена, смена статуса.
type OrderService struct {
	orders   OrderRepository
	menu     AvailabilityChecker
	notifier OrderNotifier
}

// NewOrderService создаёт OrderService поверх готовых зависимостей.
func NewOrderService(orders OrderRepository, menu AvailabilityChecker, notifier OrderNotifier) *OrderService {
	return &OrderService{orders: orders, menu: menu, notifier: notifier}
}

// Create проверяет доступность каждой позиции и создаёт заказ; если хоть одна
// позиция недоступна — не создаёт ничего, возвращает *UnavailableItemsError.
func (s *OrderService) Create(ctx context.Context, input CreateOrderInput) (domain.Order, error) {
	itemIDs := make([]uuid.UUID, len(input.Items))
	for i, it := range input.Items {
		itemIDs[i] = it.MenuItemID
	}

	available, err := s.menu.GetAvailableByIDs(ctx, input.VenueID, itemIDs)
	if err != nil {
		return domain.Order{}, err
	}

	availableByID := make(map[uuid.UUID]domain.MenuItem, len(available))
	for _, m := range available {
		availableByID[m.ID] = m
	}

	var unavailable []uuid.UUID
	orderItems := make([]domain.OrderItem, 0, len(input.Items))
	for _, it := range input.Items {
		m, ok := availableByID[it.MenuItemID]
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
		ID:              uuid.New(),
		VenueID:         input.VenueID,
		Status:          domain.OrderStatusCreated,
		CustomerName:    input.CustomerName,
		CustomerPhone:   input.CustomerPhone,
		DeliveryAddress: input.DeliveryAddress,
		Comment:         input.Comment,
		Items:           orderItems,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.orders.Create(ctx, order); err != nil {
		return domain.Order{}, err
	}

	// Успешный NotifyVenue означает, что venue-service принял заказ (Receive
	// либо создаёт его сразу в статусе accepted, либо возвращает 409 — тогда
	// notifier вернёт ошибку и до этой строки выполнение не дойдёт).
	if err := s.notifier.NotifyVenue(ctx, order); err != nil {
		return domain.Order{}, err
	}

	if err := s.orders.UpdateStatus(ctx, order.ID, domain.OrderStatusAccepted); err != nil {
		return domain.Order{}, err
	}
	order.Status = domain.OrderStatusAccepted
	order.UpdatedAt = time.Now()

	return order, nil
}

// GetByID возвращает заказ по ID.
func (s *OrderService) GetByID(ctx context.Context, id uuid.UUID) (domain.Order, error) {
	return s.orders.GetByID(ctx, id)
}

// Cancel отменяет заказ пользователем; допустимо только пока заведение не начало готовить.
func (s *OrderService) Cancel(ctx context.Context, id uuid.UUID) (domain.Order, error) {
	order, err := s.orders.GetByID(ctx, id)
	if err != nil {
		return domain.Order{}, err
	}
	if !order.CanBeCancelledByUser() {
		return domain.Order{}, domain.ErrOrderCancelNotAllowed
	}

	if err := s.orders.UpdateStatus(ctx, id, domain.OrderStatusCancelledByUser); err != nil {
		return domain.Order{}, err
	}
	order.Status = domain.OrderStatusCancelledByUser
	order.UpdatedAt = time.Now()

	if err := s.notifier.NotifyVenue(ctx, order); err != nil {
		return domain.Order{}, err
	}

	return order, nil
}

// UpdateStatusFromVenue — вебхук от venue-service (cooking/ready/cancelled_by_venue).
func (s *OrderService) UpdateStatusFromVenue(ctx context.Context, id uuid.UUID, status domain.OrderStatus) (domain.Order, error) {
	order, err := s.orders.GetByID(ctx, id)
	if err != nil {
		return domain.Order{}, err
	}
	if !order.CanTransitionTo(status) {
		return domain.Order{}, domain.ErrInvalidStatusTransition
	}

	if err := s.orders.UpdateStatus(ctx, id, status); err != nil {
		return domain.Order{}, err
	}
	order.Status = status
	order.UpdatedAt = time.Now()

	return order, nil
}

// AdvanceDelivery — ready -> delivering -> delivered.
func (s *OrderService) AdvanceDelivery(ctx context.Context, id uuid.UUID, status domain.OrderStatus) (domain.Order, error) {
	order, err := s.orders.GetByID(ctx, id)
	if err != nil {
		return domain.Order{}, err
	}
	if !order.CanAdvanceDelivery(status) {
		return domain.Order{}, domain.ErrInvalidStatusTransition
	}

	if err := s.orders.UpdateStatus(ctx, id, status); err != nil {
		return domain.Order{}, err
	}
	order.Status = status
	order.UpdatedAt = time.Now()

	return order, nil
}
