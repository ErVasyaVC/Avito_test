package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"kitchen-api/internal/domain"
)

// VenueRepository читает заведения и их категории.
type VenueRepository struct {
	pool *pgxpool.Pool
}

// NewVenueRepository создаёт VenueRepository поверх готового пула соединений.
func NewVenueRepository(pool *pgxpool.Pool) *VenueRepository {
	return &VenueRepository{pool: pool}
}

// List возвращает активные заведения, опционально отфильтрованные по категории.
func (r *VenueRepository) List(ctx context.Context, categoryID *uuid.UUID) ([]domain.Venue, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT v.id, v.name
		FROM venues v
		LEFT JOIN venue_categories vc ON vc.venue_id = v.id
		WHERE v.is_active = true
		  AND ($1::uuid IS NULL OR vc.category_id = $1)
		ORDER BY v.name
	`, categoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var venues []domain.Venue
	venueIDs := make([]uuid.UUID, 0)
	for rows.Next() {
		var v domain.Venue
		if err := rows.Scan(&v.ID, &v.Name); err != nil {
			return nil, err
		}
		venues = append(venues, v)
		venueIDs = append(venueIDs, v.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	categoriesByVenue, err := r.categoriesForVenues(ctx, venueIDs)
	if err != nil {
		return nil, err
	}
	for i := range venues {
		venues[i].Categories = categoriesByVenue[venues[i].ID]
	}

	return venues, nil
}

// GetByID возвращает активное заведение вместе с его категориями.
func (r *VenueRepository) GetByID(ctx context.Context, id uuid.UUID) (domain.Venue, error) {
	var v domain.Venue
	err := r.pool.QueryRow(ctx, `
		SELECT id, name FROM venues WHERE id = $1 AND is_active = true
	`, id).Scan(&v.ID, &v.Name)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Venue{}, domain.ErrVenueNotFound
	}
	if err != nil {
		return domain.Venue{}, err
	}

	categoriesByVenue, err := r.categoriesForVenues(ctx, []uuid.UUID{v.ID})
	if err != nil {
		return domain.Venue{}, err
	}
	v.Categories = categoriesByVenue[v.ID]

	return v, nil
}

func (r *VenueRepository) categoriesForVenues(ctx context.Context, venueIDs []uuid.UUID) (map[uuid.UUID][]domain.Category, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT vc.venue_id, c.id, c.name
		FROM venue_categories vc
		JOIN categories c ON c.id = vc.category_id
		WHERE vc.venue_id = ANY($1)
	`, venueIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[uuid.UUID][]domain.Category)
	for rows.Next() {
		var venueID uuid.UUID
		var c domain.Category
		if err := rows.Scan(&venueID, &c.ID, &c.Name); err != nil {
			return nil, err
		}
		result[venueID] = append(result[venueID], c)
	}
	return result, rows.Err()
}
