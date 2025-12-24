package services

import (
	"fmt"
	"time"

	"radio-backend/internal/domain"
	"radio-backend/internal/infrastructure/logger"
	"radio-backend/internal/repositories/postgres"
)

// StationService handles station business logic with cache-aside pattern
type StationService struct {
	cacheRepo        domain.StationCacheRepository
	externalRepo     domain.StationRepository
	searchCache      domain.SearchCacheRepository
	analyticsService *AnalyticsService
	seoService       *SEOService // NUEVO: Servicio SEO
	stationMaxAge    time.Duration
	searchCacheTTL   time.Duration
}

// NewStationService creates a new station service with cache support
func NewStationService(
	cacheRepo domain.StationCacheRepository,
	externalRepo domain.StationRepository,
	searchCache domain.SearchCacheRepository,
	analyticsService *AnalyticsService,
	seoService *SEOService, // NUEVO: Inyección de SEOService
	stationMaxAge time.Duration,
	searchCacheTTL time.Duration,
) *StationService {
	return &StationService{
		cacheRepo:        cacheRepo,
		externalRepo:     externalRepo,
		searchCache:      searchCache,
		analyticsService: analyticsService,
		seoService:       seoService, // NUEVO
		stationMaxAge:    stationMaxAge,
		searchCacheTTL:   searchCacheTTL,
	}
}

// GetByID retrieves a station by ID using cache-aside pattern
func (s *StationService) GetByID(id string, userType domain.UserType) (*domain.Station, error) {
	logger.Info("fetching station by ID", "id", id, "user_type", userType)

	// 1. Try cache first
	cachedStation, err := s.cacheRepo.Get(id)
	if err == nil && cachedStation != nil {
		logger.Info("cache hit for station", "id", id)
		if !cachedStation.IsAccessibleBy(userType) {
			return nil, domain.ErrUnauthorized
		}
		// NUEVO: Enriquecer con SEO antes de retornar
		if s.seoService != nil {
			s.seoService.EnrichStationWithSEO(cachedStation)
		}
		return cachedStation, nil
	}

	logger.Info("cache miss for station, fetching from external API", "id", id)

	// 2. Fallback to external API
	station, err := s.externalRepo.FindByID(id)
	if err != nil {
		return nil, err
	}

	// 3. Save to cache asynchronously
	go s.cacheStations([]domain.Station{*station})

	// 4. Check user access
	if !station.IsAccessibleBy(userType) {
		return nil, domain.ErrUnauthorized
	}

	// NUEVO: Enriquecer con SEO antes de retornar
	if s.seoService != nil {
		s.seoService.EnrichStationWithSEO(station)
	}

	return station, nil
}

// ListPopular returns popular stations using cache-aside pattern
func (s *StationService) ListPopular(limit int, country string, userType domain.UserType) ([]domain.Station, error) {
	// 1. Try cache first
	logger.Info("fetching popular stations", "limit", limit, "country", country, "user_type", userType)

	cachedStations, err := s.cacheRepo.FindPopular(limit, country)
	if err == nil && len(cachedStations) >= limit {
		logger.Info("cache hit for popular stations", "count", len(cachedStations))
		filtered := s.filterByUserType(cachedStations, userType)
		// NUEVO: Enriquecer con SEO
		if s.seoService != nil {
			s.seoService.EnrichStationsWithSEO(filtered)
		}
		return filtered, nil
	}

	logger.Info("cache miss for popular stations, fetching from external API")

	// 2. Fallback to external API
	stations, err := s.externalRepo.FindPopular(limit, country)
	if err != nil {
		// If external fails, return whatever we have in cache (even if less than limit)
		if len(cachedStations) > 0 {
			logger.Warn("external API failed, using partial cache", "cached_count", len(cachedStations), "error", err)
			filtered := s.filterByUserType(cachedStations, userType)
			// NUEVO: Enriquecer con SEO
			if s.seoService != nil {
				s.seoService.EnrichStationsWithSEO(filtered)
			}
			return filtered, nil
		}
		return nil, fmt.Errorf("failed to fetch stations: %w", err)
	}

	// 3. Save to cache asynchronously
	go s.cacheStations(stations)

	filtered := s.filterByUserType(stations, userType)
	// NUEVO: Enriquecer con SEO
	if s.seoService != nil {
		s.seoService.EnrichStationsWithSEO(filtered)
	}
	return filtered, nil
}

// Search searches for stations using cache-aside pattern
func (s *StationService) Search(query string, limit int, userType domain.UserType) ([]domain.Station, error) {
	if query == "" {
		return nil, domain.ErrInvalidQuery
	}

	logger.Info("searching stations", "query", query, "limit", limit, "user_type", userType)

	// 1. Check search cache
	queryHash := s.hashQuery(query, limit)
	logger.Info("checking search cache", "query", query, "query_hash", queryHash)

	cachedEntry, err := s.searchCache.Get(queryHash)
	if err != nil {
		logger.Warn("search cache get error", "query", query, "error", err)
	}

	// Database already filters expired entries (WHERE expires_at > NOW())
	if err == nil && cachedEntry != nil {
		logger.Info("search cache hit", "query", query, "station_count", len(cachedEntry.StationIDs))

		// Get stations from cache by IDs
		stations, err := s.cacheRepo.GetMany(cachedEntry.StationIDs)
		if err == nil && len(stations) > 0 {
			filtered := s.filterByUserType(stations, userType)
			// NUEVO: Enriquecer con SEO
			if s.seoService != nil {
				s.seoService.EnrichStationsWithSEO(filtered)
			}
			return filtered, nil
		}
		logger.Warn("failed to get stations from cache", "error", err, "station_count", len(stations))
	}

	// 2. Try local database search
	logger.Info("search cache miss, trying local database", "query", query)
	cachedStations, err := s.cacheRepo.FindByName(query, limit)
	if err == nil && len(cachedStations) >= limit {
		logger.Info("found in local database", "count", len(cachedStations))
		go s.saveSearchCache(queryHash, query, limit, cachedStations)
		filtered := s.filterByUserType(cachedStations, userType)
		// NUEVO: Enriquecer con SEO
		if s.seoService != nil {
			s.seoService.EnrichStationsWithSEO(filtered)
		}
		return filtered, nil
	}

	// 3. Fallback to external API
	logger.Info("local database miss, fetching from external API", "query", query)
	stations, err := s.externalRepo.Search(query, limit)
	if err != nil {
		logger.Error("external API search failed", "query", query, "limit", limit, "error", err)
		// If external fails, return whatever we have in cache
		if len(cachedStations) > 0 {
			logger.Warn("external API failed, using partial cache", "cached_count", len(cachedStations), "error", err)
			filtered := s.filterByUserType(cachedStations, userType)
			// NUEVO: Enriquecer con SEO
			if s.seoService != nil {
				s.seoService.EnrichStationsWithSEO(filtered)
			}
			return filtered, nil
		}
		return nil, fmt.Errorf("failed to search stations: %w", err)
	}

	// 4. Cache results asynchronously
	go s.cacheStations(stations)
	go s.saveSearchCache(queryHash, query, limit, stations)

	filtered := s.filterByUserType(stations, userType)
	// NUEVO: Enriquecer con SEO
	if s.seoService != nil {
		s.seoService.EnrichStationsWithSEO(filtered)
	}
	return filtered, nil
}

// filterByUserType filters stations based on user type
func (s *StationService) filterByUserType(stations []domain.Station, userType domain.UserType) []domain.Station {
	filtered := make([]domain.Station, 0, len(stations))
	for _, station := range stations {
		if station.IsAccessibleBy(userType) {
			filtered = append(filtered, station)
		}
	}
	return filtered
}

// cacheStations saves stations to cache asynchronously
func (s *StationService) cacheStations(stations []domain.Station) {
	for _, station := range stations {
		station.Source = "radio_browser"
		station.IsActive = true
		if err := s.cacheRepo.Save(&station); err != nil {
			logger.Error("failed to cache station", "station_id", station.ID, "error", err)
		}
	}
	logger.Info("cached stations", "count", len(stations))
}

// saveSearchCache saves search results to cache
func (s *StationService) saveSearchCache(queryHash, query string, limit int, stations []domain.Station) {
	stationIDs := make([]string, len(stations))
	for i, station := range stations {
		stationIDs[i] = station.ID
	}

	entry := &domain.SearchCacheEntry{
		QueryHash: queryHash,
		QueryParams: map[string]interface{}{
			"query": query,
			"limit": limit,
		},
		StationIDs:  stationIDs,
		ResultCount: len(stations),
		ExpiresAt:   time.Now().Add(s.searchCacheTTL),
		CreatedAt:   time.Now(),
	}

	if err := s.searchCache.Save(entry); err != nil {
		logger.Error("failed to save search cache", "query", query, "error", err)
	} else {
		logger.Info("saved search cache", "query", query, "count", len(stations))
	}
}

// hashQuery creates a hash for query parameters
func (s *StationService) hashQuery(query string, limit int) string {
	params := map[string]interface{}{
		"query": query,
		"limit": limit,
	}
	return postgres.HashQuery(params)
}
