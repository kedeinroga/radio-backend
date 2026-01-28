package services

import (
	"context"
	"fmt"

	"radio-backend/internal/domain"
	"radio-backend/internal/i18n"
	"radio-backend/internal/infrastructure/logger"
)

// FavoriteService handles favorite business logic
type FavoriteService struct {
	favoriteRepo domain.FavoriteRepository
	stationRepo  domain.StationCacheRepository
	stationSvc   *StationService
}

// NewFavoriteService creates a new favorite service
func NewFavoriteService(
	favoriteRepo domain.FavoriteRepository,
	stationRepo domain.StationCacheRepository,
	stationSvc *StationService,
) *FavoriteService {
	return &FavoriteService{
		favoriteRepo: favoriteRepo,
		stationRepo:  stationRepo,
		stationSvc:   stationSvc,
	}
}

// GetUserFavorites returns all favorite stations for a user with full station details
func (s *FavoriteService) GetUserFavorites(ctx context.Context, userID string) ([]domain.Station, error) {
	logger.Info("getting user favorites", "user_id", userID)

	// Get favorite records
	favorites, err := s.favoriteRepo.GetUserFavorites(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get favorites: %w", err)
	}

	if len(favorites) == 0 {
		return []domain.Station{}, nil
	}

	// Extract station IDs
	stationIDs := make([]string, len(favorites))
	for i, fav := range favorites {
		stationIDs[i] = fav.StationID
	}

	// Get station details from cache
	stations, err := s.stationRepo.GetMany(ctx, stationIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get station details: %w", err)
	}

	// Create a map for quick lookup
	stationMap := make(map[string]domain.Station)
	for _, station := range stations {
		stationMap[station.ID] = station
	}

	// Build result preserving the order of favorites
	result := make([]domain.Station, 0, len(favorites))
	for _, fav := range favorites {
		if station, ok := stationMap[fav.StationID]; ok {
			result = append(result, station)
		} else {
			// Station not in cache, log warning but continue
			logger.Warn("favorite station not found in cache", "station_id", fav.StationID, "user_id", userID)
		}
	}

	logger.Info("retrieved user favorites", "user_id", userID, "count", len(result))
	return result, nil
}

// AddFavorite adds a station to user's favorites
func (s *FavoriteService) AddFavorite(ctx context.Context, userID, stationID string, userType domain.UserType, lang i18n.Language) error {
	logger.Info("adding favorite", "user_id", userID, "station_id", stationID, "language", lang)

	// Verify station exists and is accessible
	station, err := s.stationSvc.GetByID(ctx, stationID, userType, lang)
	if err != nil {
		if err == domain.ErrUnauthorized {
			return domain.ErrUnauthorized
		}
		return fmt.Errorf("station not found or inaccessible: %w", err)
	}

	// Ensure station is in cache for future lookups
	if err := s.stationRepo.Save(ctx, station); err != nil {
		logger.Warn("failed to cache station", "station_id", stationID, "error", err)
	}

	// Add to favorites
	if err := s.favoriteRepo.AddFavorite(ctx, userID, stationID); err != nil {
		return err
	}

	logger.Info("favorite added", "user_id", userID, "station_id", stationID)
	return nil
}

// RemoveFavorite removes a station from user's favorites
func (s *FavoriteService) RemoveFavorite(ctx context.Context, userID, stationID string) error {
	logger.Info("removing favorite", "user_id", userID, "station_id", stationID)

	if err := s.favoriteRepo.RemoveFavorite(ctx, userID, stationID); err != nil {
		return err
	}

	logger.Info("favorite removed", "user_id", userID, "station_id", stationID)
	return nil
}

// IsFavorite checks if a station is in user's favorites
func (s *FavoriteService) IsFavorite(ctx context.Context, userID, stationID string) (bool, error) {
	return s.favoriteRepo.IsFavorite(ctx, userID, stationID)
}
