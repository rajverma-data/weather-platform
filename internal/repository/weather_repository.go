package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"weather-platform/internal/models"
	"weather-platform/pkg/database"
	"weather-platform/pkg/logging"
	"weather-platform/pkg/metrics"
)

// WeatherRepository provides data access for weather data
type WeatherRepository interface {
	// Station operations
	CreateStation(ctx context.Context, station *models.WeatherStation) error
	GetStation(ctx context.Context, stationID string) (*models.WeatherStation, error)
	ListStations(ctx context.Context, limit, offset int) ([]*models.WeatherStation, error)

	// Observation operations
	CreateObservation(ctx context.Context, obs *models.WeatherObservation) error
	CreateObservationsBatch(ctx context.Context, observations []*models.WeatherObservation) error
	GetObservations(ctx context.Context, filter ObservationFilter) ([]*models.WeatherObservation, int, error)
	GetObservationByStationDate(ctx context.Context, stationID string, date time.Time) (*models.WeatherObservation, error)

	// Statistics operations
	CreateStatistics(ctx context.Context, stats *models.WeatherStatistics) error
	UpsertStatistics(ctx context.Context, stats *models.WeatherStatistics) error
	GetStatistics(ctx context.Context, filter StatisticsFilter) ([]*models.WeatherStatistics, int, error)
	CalculateYearlyStatistics(ctx context.Context, stationID string, year int) (*models.WeatherStatistics, error)

	// Utility operations
	HealthCheck(ctx context.Context) error
}

// ObservationFilter defines filters for querying observations
type ObservationFilter struct {
	StationID  *string
	StartDate  *time.Time
	EndDate    *time.Time
	Limit      int
	Offset     int
}

// StatisticsFilter defines filters for querying statistics
type StatisticsFilter struct {
	StationID *string
	Year      *int
	Limit     int
	Offset    int
}

// weatherRepository implements WeatherRepository
type weatherRepository struct {
	db      *database.PostgresDB
	logger  *logging.StructuredLogger
	metrics *metrics.Collector
}

// NewWeatherRepository creates a new weather repository
func NewWeatherRepository(db *database.PostgresDB, logger *logging.StructuredLogger, metricsCollector *metrics.Collector) WeatherRepository {
	return &weatherRepository{
		db:      db,
		logger:  logger,
		metrics: metricsCollector,
	}
}

// CreateStation creates a new weather station
func (r *weatherRepository) CreateStation(ctx context.Context, station *models.WeatherStation) error {
	query := `
		INSERT INTO weather_stations (station_id, state, created_at, updated_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (station_id) DO NOTHING
	`

	_, err := r.db.ExecContext(ctx, "insert_station", query,
		station.StationID,
		station.State,
		station.CreatedAt,
		station.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create station: %w", err)
	}

	r.logger.Debug(ctx, "[REPO_CREATE_STATION] Station created", logging.Fields{
		"station_id": station.StationID,
		"state":      station.State,
	})

	return nil
}

// GetStation retrieves a weather station by ID
func (r *weatherRepository) GetStation(ctx context.Context, stationID string) (*models.WeatherStation, error) {
	query := `
		SELECT station_id, state, created_at, updated_at
		FROM weather_stations
		WHERE station_id = $1
	`

	var station models.WeatherStation
	err := r.db.GetContext(ctx, "get_station", &station, query, stationID)

	if err == sql.ErrNoRows {
		return nil, &NotFoundError{
			Resource: "weather_station",
			ID:       stationID,
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get station: %w", err)
	}

	return &station, nil
}

// ListStations retrieves all weather stations with pagination
func (r *weatherRepository) ListStations(ctx context.Context, limit, offset int) ([]*models.WeatherStation, error) {
	query := `
		SELECT station_id, state, created_at, updated_at
		FROM weather_stations
		ORDER BY station_id
		LIMIT $1 OFFSET $2
	`

	var stations []*models.WeatherStation
	err := r.db.SelectContext(ctx, "list_stations", &stations, query, limit, offset)

	if err != nil {
		return nil, fmt.Errorf("failed to list stations: %w", err)
	}

	return stations, nil
}

// CreateObservation creates a new weather observation
func (r *weatherRepository) CreateObservation(ctx context.Context, obs *models.WeatherObservation) error {
	query := `
		INSERT INTO weather_observations (
			station_id, observation_date,
			max_temperature_celsius, min_temperature_celsius, precipitation_cm,
			created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (station_id, observation_date) DO UPDATE SET
			max_temperature_celsius = EXCLUDED.max_temperature_celsius,
			min_temperature_celsius = EXCLUDED.min_temperature_celsius,
			precipitation_cm = EXCLUDED.precipitation_cm
		RETURNING id
	`

	err := r.db.DB().QueryRowContext(ctx, query,
		obs.StationID,
		obs.ObservationDate,
		obs.MaxTemperatureCelsius,
		obs.MinTemperatureCelsius,
		obs.PrecipitationCm,
		obs.CreatedAt,
	).Scan(&obs.ID)

	if err != nil {
		return fmt.Errorf("failed to create observation: %w", err)
	}

	return nil
}

// CreateObservationsBatch creates multiple observations in a single transaction
func (r *weatherRepository) CreateObservationsBatch(ctx context.Context, observations []*models.WeatherObservation) error {
	if len(observations) == 0 {
		return nil
	}

	timer := time.Now()
	defer func() {
		duration := time.Since(timer)
		r.metrics.IngestionBatchSize.Observe(float64(len(observations)))
		r.logger.Debug(ctx, "[REPO_BATCH_INSERT] Batch insert completed", logging.Fields{
			"count":       len(observations),
			"duration_ms": duration.Milliseconds(),
		})
	}()

	// Begin transaction
	tx, err := r.db.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Prepare statement
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO weather_observations (
			station_id, observation_date,
			max_temperature_celsius, min_temperature_celsius, precipitation_cm,
			created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (station_id, observation_date) DO UPDATE SET
			max_temperature_celsius = EXCLUDED.max_temperature_celsius,
			min_temperature_celsius = EXCLUDED.min_temperature_celsius,
			precipitation_cm = EXCLUDED.precipitation_cm
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	// Execute batch
	for _, obs := range observations {
		_, err := stmt.ExecContext(ctx,
			obs.StationID,
			obs.ObservationDate,
			obs.MaxTemperatureCelsius,
			obs.MinTemperatureCelsius,
			obs.PrecipitationCm,
			obs.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to insert observation: %w", err)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	r.metrics.IngestionRecordsTotal.Add(float64(len(observations)))

	return nil
}

// GetObservations retrieves weather observations with filtering and pagination
func (r *weatherRepository) GetObservations(ctx context.Context, filter ObservationFilter) ([]*models.WeatherObservation, int, error) {
	// Build query with filters
	query := `
		SELECT id, station_id, observation_date,
		       max_temperature_celsius, min_temperature_celsius, precipitation_cm,
		       created_at
		FROM weather_observations
		WHERE 1=1
	`
	args := []interface{}{}
	argNum := 1

	if filter.StationID != nil {
		query += fmt.Sprintf(" AND station_id = $%d", argNum)
		args = append(args, *filter.StationID)
		argNum++
	}

	if filter.StartDate != nil {
		query += fmt.Sprintf(" AND observation_date >= $%d", argNum)
		args = append(args, *filter.StartDate)
		argNum++
	}

	if filter.EndDate != nil {
		query += fmt.Sprintf(" AND observation_date <= $%d", argNum)
		args = append(args, *filter.EndDate)
		argNum++
	}

	// Get total count
	countQuery := "SELECT COUNT(*) FROM (" + query + ") AS count_query"
	var totalCount int
	err := r.db.GetContext(ctx, "count_observations", &totalCount, countQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count observations: %w", err)
	}

	// Add ordering and pagination
	query += " ORDER BY observation_date DESC, station_id"
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argNum, argNum+1)
	args = append(args, filter.Limit, filter.Offset)

	// Execute query
	var observations []*models.WeatherObservation
	err = r.db.SelectContext(ctx, "get_observations", &observations, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get observations: %w", err)
	}

	return observations, totalCount, nil
}

// GetObservationByStationDate retrieves a specific observation
func (r *weatherRepository) GetObservationByStationDate(ctx context.Context, stationID string, date time.Time) (*models.WeatherObservation, error) {
	query := `
		SELECT id, station_id, observation_date,
		       max_temperature_celsius, min_temperature_celsius, precipitation_cm,
		       created_at
		FROM weather_observations
		WHERE station_id = $1 AND observation_date = $2
	`

	var obs models.WeatherObservation
	err := r.db.GetContext(ctx, "get_observation_by_date", &obs, query, stationID, date)

	if err == sql.ErrNoRows {
		return nil, &NotFoundError{
			Resource: "weather_observation",
			ID:       fmt.Sprintf("%s:%s", stationID, date.Format("2006-01-02")),
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get observation: %w", err)
	}

	return &obs, nil
}

// CreateStatistics creates new weather statistics
func (r *weatherRepository) CreateStatistics(ctx context.Context, stats *models.WeatherStatistics) error {
	query := `
		INSERT INTO weather_statistics (
			station_id, year,
			avg_max_temperature_celsius, avg_min_temperature_celsius, total_precipitation_cm,
			observation_count, valid_max_temp_count, valid_min_temp_count, valid_precipitation_count,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id
	`

	err := r.db.DB().QueryRowContext(ctx, query,
		stats.StationID,
		stats.Year,
		stats.AvgMaxTemperatureCelsius,
		stats.AvgMinTemperatureCelsius,
		stats.TotalPrecipitationCm,
		stats.ObservationCount,
		stats.ValidMaxTempCount,
		stats.ValidMinTempCount,
		stats.ValidPrecipitationCount,
		stats.CreatedAt,
		stats.UpdatedAt,
	).Scan(&stats.ID)

	if err != nil {
		return fmt.Errorf("failed to create statistics: %w", err)
	}

	return nil
}

// UpsertStatistics creates or updates weather statistics
func (r *weatherRepository) UpsertStatistics(ctx context.Context, stats *models.WeatherStatistics) error {
	query := `
		INSERT INTO weather_statistics (
			station_id, year,
			avg_max_temperature_celsius, avg_min_temperature_celsius, total_precipitation_cm,
			observation_count, valid_max_temp_count, valid_min_temp_count, valid_precipitation_count,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (station_id, year) DO UPDATE SET
			avg_max_temperature_celsius = EXCLUDED.avg_max_temperature_celsius,
			avg_min_temperature_celsius = EXCLUDED.avg_min_temperature_celsius,
			total_precipitation_cm = EXCLUDED.total_precipitation_cm,
			observation_count = EXCLUDED.observation_count,
			valid_max_temp_count = EXCLUDED.valid_max_temp_count,
			valid_min_temp_count = EXCLUDED.valid_min_temp_count,
			valid_precipitation_count = EXCLUDED.valid_precipitation_count,
			updated_at = EXCLUDED.updated_at
		RETURNING id
	`

	err := r.db.DB().QueryRowContext(ctx, query,
		stats.StationID,
		stats.Year,
		stats.AvgMaxTemperatureCelsius,
		stats.AvgMinTemperatureCelsius,
		stats.TotalPrecipitationCm,
		stats.ObservationCount,
		stats.ValidMaxTempCount,
		stats.ValidMinTempCount,
		stats.ValidPrecipitationCount,
		stats.CreatedAt,
		stats.UpdatedAt,
	).Scan(&stats.ID)

	if err != nil {
		return fmt.Errorf("failed to upsert statistics: %w", err)
	}

	return nil
}

// GetStatistics retrieves weather statistics with filtering and pagination
func (r *weatherRepository) GetStatistics(ctx context.Context, filter StatisticsFilter) ([]*models.WeatherStatistics, int, error) {
	// Build query with filters
	query := `
		SELECT id, station_id, year,
		       avg_max_temperature_celsius, avg_min_temperature_celsius, total_precipitation_cm,
		       observation_count, valid_max_temp_count, valid_min_temp_count, valid_precipitation_count,
		       created_at, updated_at
		FROM weather_statistics
		WHERE 1=1
	`
	args := []interface{}{}
	argNum := 1

	if filter.StationID != nil {
		query += fmt.Sprintf(" AND station_id = $%d", argNum)
		args = append(args, *filter.StationID)
		argNum++
	}

	if filter.Year != nil {
		query += fmt.Sprintf(" AND year = $%d", argNum)
		args = append(args, *filter.Year)
		argNum++
	}

	// Get total count
	countQuery := "SELECT COUNT(*) FROM (" + query + ") AS count_query"
	var totalCount int
	err := r.db.GetContext(ctx, "count_statistics", &totalCount, countQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count statistics: %w", err)
	}

	// Add ordering and pagination
	query += " ORDER BY year DESC, station_id"
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argNum, argNum+1)
	args = append(args, filter.Limit, filter.Offset)

	// Execute query
	var statistics []*models.WeatherStatistics
	err = r.db.SelectContext(ctx, "get_statistics", &statistics, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get statistics: %w", err)
	}

	return statistics, totalCount, nil
}

// CalculateYearlyStatistics calculates statistics for a station and year
func (r *weatherRepository) CalculateYearlyStatistics(ctx context.Context, stationID string, year int) (*models.WeatherStatistics, error) {
	timer := time.Now()
	defer func() {
		duration := time.Since(timer)
		r.metrics.StatsCalculationDuration.Observe(duration.Seconds())
		r.logger.Debug(ctx, "[REPO_CALC_STATS] Statistics calculated", logging.Fields{
			"station_id":  stationID,
			"year":        year,
			"duration_ms": duration.Milliseconds(),
		})
	}()

	query := `
		SELECT
			COUNT(*) as observation_count,
			COUNT(max_temperature_celsius) as valid_max_temp_count,
			COUNT(min_temperature_celsius) as valid_min_temp_count,
			COUNT(precipitation_cm) as valid_precipitation_count,
			AVG(max_temperature_celsius) as avg_max_temperature_celsius,
			AVG(min_temperature_celsius) as avg_min_temperature_celsius,
			SUM(precipitation_cm) as total_precipitation_cm
		FROM weather_observations
		WHERE station_id = $1
		  AND EXTRACT(YEAR FROM observation_date) = $2
	`

	var result struct {
		ObservationCount        int
		ValidMaxTempCount       int
		ValidMinTempCount       int
		ValidPrecipitationCount int
		AvgMaxTemperatureCelsius  *float64
		AvgMinTemperatureCelsius  *float64
		TotalPrecipitationCm      *float64
	}

	err := r.db.GetContext(ctx, "calculate_statistics", &result, query, stationID, year)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate statistics: %w", err)
	}

	stats := &models.WeatherStatistics{
		StationID:               stationID,
		Year:                    year,
		ObservationCount:        result.ObservationCount,
		ValidMaxTempCount:       result.ValidMaxTempCount,
		ValidMinTempCount:       result.ValidMinTempCount,
		ValidPrecipitationCount: result.ValidPrecipitationCount,
		AvgMaxTemperatureCelsius:  result.AvgMaxTemperatureCelsius,
		AvgMinTemperatureCelsius:  result.AvgMinTemperatureCelsius,
		TotalPrecipitationCm:      result.TotalPrecipitationCm,
		CreatedAt:               time.Now().UTC(),
		UpdatedAt:               time.Now().UTC(),
	}

	return stats, nil
}

// HealthCheck performs a repository health check
func (r *weatherRepository) HealthCheck(ctx context.Context) error {
	return r.db.HealthCheck(ctx)
}

// NotFoundError represents a resource not found error
type NotFoundError struct {
	Resource string
	ID       string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s not found: %s", e.Resource, e.ID)
}

func (e *NotFoundError) IsTransient() bool {
	return false
}
