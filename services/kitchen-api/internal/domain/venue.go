package domain

import (
	"errors"

	"github.com/google/uuid"
)

// ErrVenueNotFound — заведение не найдено или неактивно.
var ErrVenueNotFound = errors.New("venue not found")

// Venue — заведение (ресторан/кафе/магазин), допущенное на площадку.
type Venue struct {
	ID         uuid.UUID
	Name       string
	Categories []Category
}
