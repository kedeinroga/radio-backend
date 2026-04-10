package services

import (
	"fmt"
	"radio-backend/internal/domain"
)

// SecurityService handles security-related operations
type SecurityService struct {
	securityRepo domain.SecurityRepository
}

// NewSecurityService creates a new security service
func NewSecurityService(securityRepo domain.SecurityRepository) *SecurityService {
	return &SecurityService{
		securityRepo: securityRepo,
	}
}

// GetMetrics retrieves security metrics for a given period
func (s *SecurityService) GetMetrics(period string) (*domain.SecurityMetrics, error) {
	// Validate period
	if period != "7d" && period != "30d" {
		return nil, fmt.Errorf("invalid period: %s (allowed: 7d, 30d)", period)
	}

	metrics, err := s.securityRepo.GetMetrics(period)
	if err != nil {
		return nil, fmt.Errorf("failed to get security metrics: %w", err)
	}

	return metrics, nil
}

// GetLogs retrieves security logs with pagination and filtering
func (s *SecurityService) GetLogs(filter *domain.SecurityLogFilter) (*domain.SecurityLogResult, error) {
	// Validate pagination parameters
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 50
	}
	if filter.Limit > 100 {
		filter.Limit = 100 // Max limit
	}

	// Validate event_type if provided
	validEventTypes := map[string]bool{
		"login_success":             true,
		"login_failed":              true,
		"token.issued":              true,
		"token.validated":           true,
		"token.revoked":             true,
		"session.created":           true,
		"session.revoked":           true,
		"session.suspicious":        true,
		"password.reset":            true,
		"password.changed":          true,
		"suspicious_request_source": true,
	}

	if filter.EventType != "" && !validEventTypes[filter.EventType] {
		return nil, fmt.Errorf("invalid event_type: %s", filter.EventType)
	}

	logs, err := s.securityRepo.GetLogs(filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get security logs: %w", err)
	}

	return logs, nil
}

// GetSuspiciousSourceStats returns aggregated anomaly stats for non-browser access.
func (s *SecurityService) GetSuspiciousSourceStats(period string) (*domain.SuspiciousSourceStats, error) {
	if period != "24h" && period != "7d" && period != "30d" {
		return nil, fmt.Errorf("invalid period: %s (allowed: 24h, 7d, 30d)", period)
	}

	stats, err := s.securityRepo.GetSuspiciousSourceStats(period)
	if err != nil {
		return nil, fmt.Errorf("failed to get suspicious source stats: %w", err)
	}

	return stats, nil
}
