package services

import (
	"errors"
	"testing"
	"time"

	"radio-backend/internal/domain"
)

// Mock SecurityRepository
type mockSecurityRepository struct {
	getMetricsFunc               func(period string) (*domain.SecurityMetrics, error)
	getLogsFunc                  func(filter *domain.SecurityLogFilter) (*domain.SecurityLogResult, error)
	getSuspiciousSourceStatsFunc func(period string) (*domain.SuspiciousSourceStats, error)
}

func (m *mockSecurityRepository) GetMetrics(period string) (*domain.SecurityMetrics, error) {
	if m.getMetricsFunc != nil {
		return m.getMetricsFunc(period)
	}
	return nil, nil
}

func (m *mockSecurityRepository) GetLogs(filter *domain.SecurityLogFilter) (*domain.SecurityLogResult, error) {
	if m.getLogsFunc != nil {
		return m.getLogsFunc(filter)
	}
	return nil, nil
}

func (m *mockSecurityRepository) LogSecurityEvent(event *domain.SecurityEvent) error {
	return nil
}

func (m *mockSecurityRepository) GetSuspiciousSourceStats(period string) (*domain.SuspiciousSourceStats, error) {
	if m.getSuspiciousSourceStatsFunc != nil {
		return m.getSuspiciousSourceStatsFunc(period)
	}
	return &domain.SuspiciousSourceStats{Period: period}, nil
}

func TestSecurityService_GetMetrics(t *testing.T) {
	tests := []struct {
		name    string
		period  string
		setup   func(*mockSecurityRepository)
		wantErr bool
	}{
		{
			name:   "Valid 7d period",
			period: "7d",
			setup: func(repo *mockSecurityRepository) {
				repo.getMetricsFunc = func(period string) (*domain.SecurityMetrics, error) {
					return &domain.SecurityMetrics{
						TotalLoginsToday:    100,
						TotalLoginsWeek:     500,
						FailedAttemptsToday: 10,
						FailedAttemptsWeek:  50,
						ActiveSessions:      25,
						UniqueLocationsWeek: 15,
						Trends: domain.SecurityTrends{
							LoginsTrend:         12.5,
							FailedAttemptsTrend: -8.3,
						},
					}, nil
				}
			},
			wantErr: false,
		},
		{
			name:   "Valid 30d period",
			period: "30d",
			setup: func(repo *mockSecurityRepository) {
				repo.getMetricsFunc = func(period string) (*domain.SecurityMetrics, error) {
					return &domain.SecurityMetrics{
						TotalLoginsToday:    50,
						TotalLoginsWeek:     1200,
						FailedAttemptsToday: 5,
						FailedAttemptsWeek:  120,
						ActiveSessions:      30,
						UniqueLocationsWeek: 25,
					}, nil
				}
			},
			wantErr: false,
		},
		{
			name:    "Invalid period",
			period:  "90d",
			setup:   func(repo *mockSecurityRepository) {},
			wantErr: true,
		},
		{
			name:   "Repository error",
			period: "7d",
			setup: func(repo *mockSecurityRepository) {
				repo.getMetricsFunc = func(period string) (*domain.SecurityMetrics, error) {
					return nil, errors.New("database error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockSecurityRepository{}
			tt.setup(repo)

			service := NewSecurityService(repo)
			metrics, err := service.GetMetrics(tt.period)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetMetrics() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && metrics == nil {
				t.Error("GetMetrics() returned nil metrics when no error expected")
			}
		})
	}
}

func TestSecurityService_GetLogs(t *testing.T) {
	now := time.Now()
	testEmail := "test@example.com"
	testIP := "192.168.1.1"

	tests := []struct {
		name    string
		filter  *domain.SecurityLogFilter
		setup   func(*mockSecurityRepository)
		wantErr bool
	}{
		{
			name: "Valid filter with pagination",
			filter: &domain.SecurityLogFilter{
				Page:  1,
				Limit: 50,
			},
			setup: func(repo *mockSecurityRepository) {
				repo.getLogsFunc = func(filter *domain.SecurityLogFilter) (*domain.SecurityLogResult, error) {
					return &domain.SecurityLogResult{
						Logs: []*domain.SecurityLog{
							{
								ID:        "log-1",
								Timestamp: now,
								EventType: "login_success",
								Email:     &testEmail,
								IPAddress: &testIP,
							},
						},
						Total: 1,
						Page:  1,
						Limit: 50,
					}, nil
				}
			},
			wantErr: false,
		},
		{
			name: "Filter by event type",
			filter: &domain.SecurityLogFilter{
				Page:      1,
				Limit:     50,
				EventType: "login_failed",
			},
			setup: func(repo *mockSecurityRepository) {
				repo.getLogsFunc = func(filter *domain.SecurityLogFilter) (*domain.SecurityLogResult, error) {
					return &domain.SecurityLogResult{
						Logs:  []*domain.SecurityLog{},
						Total: 0,
						Page:  1,
						Limit: 50,
					}, nil
				}
			},
			wantErr: false,
		},
		{
			name: "Invalid event type",
			filter: &domain.SecurityLogFilter{
				Page:      1,
				Limit:     50,
				EventType: "invalid_type",
			},
			setup:   func(repo *mockSecurityRepository) {},
			wantErr: true,
		},
		{
			name: "Invalid page number (negative)",
			filter: &domain.SecurityLogFilter{
				Page:  -1,
				Limit: 50,
			},
			setup: func(repo *mockSecurityRepository) {
				repo.getLogsFunc = func(filter *domain.SecurityLogFilter) (*domain.SecurityLogResult, error) {
					if filter.Page < 1 {
						t.Error("Service should normalize page to 1")
					}
					return &domain.SecurityLogResult{
						Logs:  []*domain.SecurityLog{},
						Total: 0,
						Page:  filter.Page,
						Limit: filter.Limit,
					}, nil
				}
			},
			wantErr: false,
		},
		{
			name: "Limit exceeds maximum",
			filter: &domain.SecurityLogFilter{
				Page:  1,
				Limit: 200,
			},
			setup: func(repo *mockSecurityRepository) {
				repo.getLogsFunc = func(filter *domain.SecurityLogFilter) (*domain.SecurityLogResult, error) {
					if filter.Limit > 100 {
						t.Error("Service should limit to max 100")
					}
					return &domain.SecurityLogResult{
						Logs:  []*domain.SecurityLog{},
						Total: 0,
						Page:  filter.Page,
						Limit: filter.Limit,
					}, nil
				}
			},
			wantErr: false,
		},
		{
			name: "Repository error",
			filter: &domain.SecurityLogFilter{
				Page:  1,
				Limit: 50,
			},
			setup: func(repo *mockSecurityRepository) {
				repo.getLogsFunc = func(filter *domain.SecurityLogFilter) (*domain.SecurityLogResult, error) {
					return nil, errors.New("database error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockSecurityRepository{}
			tt.setup(repo)

			service := NewSecurityService(repo)
			result, err := service.GetLogs(tt.filter)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetLogs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if result == nil {
					t.Error("GetLogs() returned nil result when no error expected")
				}
				// Verify pagination normalization
				if tt.filter.Page < 1 && result.Page < 1 {
					t.Error("Page should be normalized to 1")
				}
				if tt.filter.Limit > 100 && result.Limit > 100 {
					t.Error("Limit should be capped at 100")
				}
			}
		})
	}
}

func TestSecurityService_GetSuspiciousSourceStats(t *testing.T) {
	tests := []struct {
		name    string
		period  string
		setup   func(*mockSecurityRepository)
		wantErr bool
	}{
		{
			name:   "Valid 24h period",
			period: "24h",
			setup: func(repo *mockSecurityRepository) {
				repo.getSuspiciousSourceStatsFunc = func(period string) (*domain.SuspiciousSourceStats, error) {
					return &domain.SuspiciousSourceStats{
						Period:     period,
						TotalCount: 10,
						BySouce:    []domain.SourceCount{{Source: "postman", Count: 7}, {Source: "curl", Count: 3}},
						TopIPs:     []domain.IPCount{{IP: "1.2.3.4", Count: 5, LastSeen: "2026-04-10T00:00:00Z"}},
						TopPaths:   []domain.PathCount{{Path: "/api/v1/auth/login", Count: 8}},
					}, nil
				}
			},
			wantErr: false,
		},
		{
			name:   "Valid 7d period",
			period: "7d",
			setup: func(repo *mockSecurityRepository) {
				repo.getSuspiciousSourceStatsFunc = func(period string) (*domain.SuspiciousSourceStats, error) {
					return &domain.SuspiciousSourceStats{Period: period, TotalCount: 0}, nil
				}
			},
			wantErr: false,
		},
		{
			name:   "Valid 30d period",
			period: "30d",
			setup: func(repo *mockSecurityRepository) {
				repo.getSuspiciousSourceStatsFunc = func(period string) (*domain.SuspiciousSourceStats, error) {
					return &domain.SuspiciousSourceStats{Period: period, TotalCount: 0}, nil
				}
			},
			wantErr: false,
		},
		{
			name:    "Invalid period",
			period:  "90d",
			setup:   func(repo *mockSecurityRepository) {},
			wantErr: true,
		},
		{
			name:    "Empty period",
			period:  "",
			setup:   func(repo *mockSecurityRepository) {},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockSecurityRepository{}
			tt.setup(repo)

			service := NewSecurityService(repo)
			stats, err := service.GetSuspiciousSourceStats(tt.period)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetSuspiciousSourceStats() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && stats == nil {
				t.Error("GetSuspiciousSourceStats() returned nil stats when no error expected")
			}
		})
	}
}
