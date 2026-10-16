// Package domain содержит бизнес-сущности и правила, не зависящие от HTTP или БД.
package domain

import "github.com/google/uuid"

// Category — категория кухни (например, "Итальянская кухня").
type Category struct {
	ID   uuid.UUID
	Name string
}
