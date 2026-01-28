package domain

import (
	"context"
	"time"
)

// Favorite represents a user's favorite station
type Favorite struct {
	UserID    string
	StationID string
	CreatedAt time.Time
}

// FavoriteRepository defines the interface for favorite data access
type FavoriteRepository interface {
	// GetUserFavorites returns all favorite stations for a user
	GetUserFavorites(ctx context.Context, userID string) ([]Favorite, error)

	// AddFavorite adds a station to user's favorites
	AddFavorite(ctx context.Context, userID, stationID string) error

	// RemoveFavorite removes a station from user's favorites
	RemoveFavorite(ctx context.Context, userID, stationID string) error

	// IsFavorite checks if a station is in user's favorites
	IsFavorite(ctx context.Context, userID, stationID string) (bool, error)
}
