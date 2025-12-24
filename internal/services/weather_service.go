package services

import (
	"context"

	"weather-platform/internal/models"
	"weather-platform/internal/repository"
	"weather-platform/pkg/logging"
	"weather-platform/pkg/metrics"
)

// WeatherService handles weather data operations
type WeatherService struct {
	repo    repository.WeatherRepository
	logger  *logging.StructuredLogger
	metrics *metrics.Collector
}

// NewWeatherService creates a new weather service
func NewWeatherService(repo repository.WeatherRepository, logger *logging.StructuredLogger, metricsCollector *metrics.Collector) *WeatherService {
	return &WeatherService{
		repo:    repo,
		logger:  logger,
		metrics: metricsCollector,
	}
}

// GetObservations retrieves weather observations with filtering
func (s *WeatherService) GetObservations(ctx context.Context, filter repository.ObservationFilter) ([]*models.WeatherObservation, int, error) {
	return s.repo.GetObservations(ctx, filter)
}

// GetStations retrieves all weather stations
func (s *WeatherService) GetStations(ctx context.Context, limit, offset int) ([]*models.WeatherStation, error) {
	return s.repo.ListStations(ctx, limit, offset)
}
