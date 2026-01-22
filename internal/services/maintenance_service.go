package services

import (
	"radio-backend/internal/domain"
)

// MaintenanceService maneja la lógica de mantenimiento
type MaintenanceService struct {
	maintenanceRepo domain.MaintenanceRepository
}

// NewMaintenanceService crea un nuevo servicio de mantenimiento
func NewMaintenanceService(maintenanceRepo domain.MaintenanceRepository) *MaintenanceService {
	return &MaintenanceService{
		maintenanceRepo: maintenanceRepo,
	}
}

// GetRecommendations obtiene las recomendaciones de mantenimiento
func (s *MaintenanceService) GetRecommendations() ([]domain.MaintenanceRecommendation, error) {
	return s.maintenanceRepo.GetMaintenanceRecommendations()
}

// RefreshAllViews refresca todas las vistas materializadas
func (s *MaintenanceService) RefreshAllViews() ([]domain.RefreshResult, error) {
	return s.maintenanceRepo.RefreshAllViews()
}

// RefreshSEOViews refresca solo las vistas de SEO
func (s *MaintenanceService) RefreshSEOViews() ([]domain.RefreshResult, error) {
	return s.maintenanceRepo.RefreshSEOViews()
}

// RefreshAnalyticsViews refresca solo las vistas de analytics
func (s *MaintenanceService) RefreshAnalyticsViews() ([]domain.RefreshResult, error) {
	return s.maintenanceRepo.RefreshAnalyticsViews()
}

// GetRefreshStatistics obtiene las estadísticas de refresh
func (s *MaintenanceService) GetRefreshStatistics(daysBack int) ([]domain.RefreshStatistics, error) {
	if daysBack <= 0 {
		daysBack = 7 // Default: últimos 7 días
	}
	return s.maintenanceRepo.GetRefreshStatistics(daysBack)
}

// CleanupOldPartitions limpia particiones antiguas
func (s *MaintenanceService) CleanupOldPartitions(retentionMonths int) ([]domain.PartitionCleanupResult, error) {
	if retentionMonths <= 0 {
		retentionMonths = 12 // Default: mantener 12 meses
	}
	return s.maintenanceRepo.CleanupOldPartitions(retentionMonths)
}

// CheckFuturePartitions verifica que existan particiones futuras
func (s *MaintenanceService) CheckFuturePartitions(monthsAhead int) ([]domain.PartitionCheckResult, error) {
	if monthsAhead <= 0 {
		monthsAhead = 3 // Default: verificar 3 meses adelante
	}
	return s.maintenanceRepo.CheckFuturePartitions(monthsAhead)
}

// GetPartitionStatus obtiene el estado de todas las particiones
func (s *MaintenanceService) GetPartitionStatus() ([]domain.PartitionStatusResult, error) {
	return s.maintenanceRepo.GetPartitionStatus()
}

// PerformFullMaintenance ejecuta un mantenimiento completo
func (s *MaintenanceService) PerformFullMaintenance() (map[string]interface{}, error) {
	result := make(map[string]interface{})

	// 1. Refresh de vistas
	refreshResults, err := s.RefreshAllViews()
	if err != nil {
		result["refresh_views_error"] = err.Error()
	} else {
		result["refresh_views"] = refreshResults
	}

	// 2. Verificar particiones futuras
	checkResults, err := s.CheckFuturePartitions(3)
	if err != nil {
		result["check_partitions_error"] = err.Error()
	} else {
		result["check_partitions"] = checkResults

		// Verificar si hay problemas
		hasMissing := false
		for _, r := range checkResults {
			if !r.PartitionsExist {
				hasMissing = true
				break
			}
		}
		result["partitions_missing"] = hasMissing
	}

	// 3. Estado de particiones
	statusResults, err := s.GetPartitionStatus()
	if err != nil {
		result["partition_status_error"] = err.Error()
	} else {
		result["partition_status"] = statusResults
	}

	return result, nil
}
