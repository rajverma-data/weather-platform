package services

import (
	"context"
	"fmt"
	"time"

	"weather-platform/internal/models"
	"weather-platform/internal/repository"
	"weather-platform/pkg/logging"
	"weather-platform/pkg/metrics"
)

// StatisticsService handles weather statistics calculations
type StatisticsService struct {
	repo    repository.WeatherRepository
	logger  *logging.StructuredLogger
	metrics *metrics.Collector
}

// NewStatisticsService creates a new statistics service
func NewStatisticsService(repo repository.WeatherRepository, logger *logging.StructuredLogger, metricsCollector *metrics.Collector) *StatisticsService {
	return &StatisticsService{
		repo:    repo,
		logger:  logger,
		metrics: metricsCollector,
	}
}

// CalculateAllStatistics calculates statistics for all stations and years
func (s *StatisticsService) CalculateAllStatistics(ctx context.Context) error {
	startTime := time.Now()

	s.logger.Info(ctx, "[STATS_CALC_START] Starting statistics calculation", logging.Fields{
		"stage": "INITIALIZATION",
	})

	// Get all stations
	stations, err := s.repo.ListStations(ctx, 10000, 0)
	if err != nil {
		return fmt.Errorf("failed to list stations: %w", err)
	}

	totalStats := 0
	for _, station := range stations {
		// Calculate for years 1985-2014
		for year := 1985; year <= 2014; year++ {
			stats, err := s.repo.CalculateYearlyStatistics(ctx, station.StationID, year)
			if err != nil {
				s.logger.Error(ctx, "[STATS_CALC_ERROR] Failed to calculate statistics", logging.Fields{
					"station_id": station.StationID,
					"year":       year,
				}, err)
				continue
			}

			// Only save if there are observations
			if stats.ObservationCount > 0 {
				if err := s.repo.UpsertStatistics(ctx, stats); err != nil {
					s.logger.Error(ctx, "[STATS_SAVE_ERROR] Failed to save statistics", logging.Fields{
						"station_id": station.StationID,
						"year":       year,
					}, err)
					continue
				}
				totalStats++
			}
		}

		s.logger.Info(ctx, "[STATS_STATION_COMPLETE] Station statistics calculated", logging.Fields{
			"station_id": station.StationID,
		})
	}

	duration := time.Since(startTime)

	s.logger.Info(ctx, "[STATS_CALC_COMPLETE] Statistics calculation completed", logging.Fields{
		"total_stations":  len(stations),
		"total_statistics": totalStats,
		"duration_seconds": duration.Seconds(),
		"stage":           "COMPLETE",
	})

	return nil
}

// GetStatistics retrieves statistics with filtering
func (s *StatisticsService) GetStatistics(ctx context.Context, filter repository.StatisticsFilter) ([]*models.WeatherStatistics, int, error) {
	return s.repo.GetStatistics(ctx, filter)
}
