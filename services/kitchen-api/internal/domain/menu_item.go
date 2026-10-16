package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// ErrMenuItemNotFound — товар не найден или недоступен.
var ErrMenuItemNotFound = errors.New("menu item not found")

// MenuItem — позиция меню заведения (кэш kitchen-api, синхронизируется с venue-service).
type MenuItem struct {
	ID          uuid.UUID
	VenueID     uuid.UUID
	Name        string
	Description *string
	Price       string
	IsAvailable bool
	RemovedAt   *time.Time
}
