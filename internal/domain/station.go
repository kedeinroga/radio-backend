package domain

// Station represents a radio station entity
type Station struct {
	ID            string
	Name          string
	StreamURL     string
	ImageURL      string
	Tags          []string
	Country       string
	Votes         int
	IsPremiumOnly bool
}

// IsAccessibleBy checks if a station is accessible by a given user type
func (s *Station) IsAccessibleBy(userType UserType) bool {
	if s.IsPremiumOnly {
		return userType == UserTypePremium
	}
	return true
}

// StationRepository defines the interface for station data access
type StationRepository interface {
	FindPopular(limit int, country string) ([]Station, error)
	Search(query string, limit int) ([]Station, error)
}
