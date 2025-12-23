package domain

import "time"

// Favorite represents a user's favorite station
type Favorite struct {
	UserID    string
	StationID string
	CreatedAt time.Time
}

// FavoriteRepository defines the interface for favorite data access
type FavoriteRepository interface {
	// GetUserFavorites returns all favorite stations for a user
	GetUserFavorites(userID string) ([]Favorite, error)

	// AddFavorite adds a station to user's favorites
	AddFavorite(userID, stationID string) error

	// RemoveFavorite removes a station from user's favorites
	RemoveFavorite(userID, stationID string) error

	// IsFavorite checks if a station is in user's favorites
	IsFavorite(userID, stationID string) (bool, error)
}
