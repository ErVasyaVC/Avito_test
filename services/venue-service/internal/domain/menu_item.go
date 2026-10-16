// Package domain содержит бизнес-сущности и правила, не зависящие от HTTP или БД.
package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// ErrMenuItemNotFound — товар не найден или уже убран из меню.
var ErrMenuItemNotFound = errors.New("menu item not found")

// MenuItem — позиция меню заведения; источник правды (kitchen-api хранит кэш).
type MenuItem struct {
	ID          uuid.UUID
	Name        string
	Description *string
	Price       string
	IsAvailable bool
	RemovedAt   *time.Time
}
