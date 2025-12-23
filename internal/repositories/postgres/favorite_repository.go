package postgres

import (
	"fmt"
	"time"

	"radio-backend/internal/domain"
	"radio-backend/internal/infrastructure/database"

	"github.com/lib/pq"
)

// FavoriteRepository implements domain.FavoriteRepository
type FavoriteRepository struct {
	db *database.Connection
}

// NewFavoriteRepository creates a new favorite repository
func NewFavoriteRepository(db *database.Connection) *FavoriteRepository {
	return &FavoriteRepository{db: db}
}

// GetUserFavorites returns all favorite stations for a user
func (r *FavoriteRepository) GetUserFavorites(userID string) ([]domain.Favorite, error) {
	query := `
		SELECT user_id, station_id, created_at
		FROM user_favorites
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.DB.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user favorites: %w", err)
	}
	defer rows.Close()

	favorites := make([]domain.Favorite, 0)
	for rows.Next() {
		var fav domain.Favorite
		err := rows.Scan(&fav.UserID, &fav.StationID, &fav.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan favorite: %w", err)
		}
		favorites = append(favorites, fav)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating favorites: %w", err)
	}

	return favorites, nil
}

// AddFavorite adds a station to user's favorites
func (r *FavoriteRepository) AddFavorite(userID, stationID string) error {
	query := `
		INSERT INTO user_favorites (user_id, station_id, created_at)
		VALUES ($1, $2, $3)
	`

	_, err := r.db.DB.Exec(query, userID, stationID, time.Now())
	if err != nil {
		// Check for unique constraint violation (duplicate favorite)
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return domain.ErrFavoriteAlreadyExists
		}
		return fmt.Errorf("failed to add favorite: %w", err)
	}

	return nil
}

// RemoveFavorite removes a station from user's favorites
func (r *FavoriteRepository) RemoveFavorite(userID, stationID string) error {
	query := `
		DELETE FROM user_favorites
		WHERE user_id = $1 AND station_id = $2
	`

	result, err := r.db.DB.Exec(query, userID, stationID)
	if err != nil {
		return fmt.Errorf("failed to remove favorite: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return domain.ErrFavoriteNotFound
	}

	return nil
}

// IsFavorite checks if a station is in user's favorites
func (r *FavoriteRepository) IsFavorite(userID, stationID string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM user_favorites
			WHERE user_id = $1 AND station_id = $2
		)
	`

	var exists bool
	err := r.db.DB.QueryRow(query, userID, stationID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check favorite: %w", err)
	}

	return exists, nil
}
