package services

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"weather-platform/internal/models"
	"weather-platform/internal/repository"
	"weather-platform/pkg/logging"
	"weather-platform/pkg/metrics"
)

// IngestionService handles weather data ingestion
type IngestionService struct {
	repo    repository.WeatherRepository
	logger  *logging.StructuredLogger
	metrics *metrics.Collector
}

// IngestionResult contains ingestion statistics
type IngestionResult struct {
	TotalFiles       int
	TotalRecords     int
	SuccessfulRecords int
	FailedRecords    int
	StationsCreated  int
	Duration         time.Duration
	Errors           []string
}

// NewIngestionService creates a new ingestion service
func NewIngestionService(repo repository.WeatherRepository, logger *logging.StructuredLogger, metricsCollector *metrics.Collector) *IngestionService {
	return &IngestionService{
		repo:    repo,
		logger:  logger,
		metrics: metricsCollector,
	}
}

// IngestDirectory ingests all weather data files from a directory
func (s *IngestionService) IngestDirectory(ctx context.Context, dataDir string, batchSize int) (*IngestionResult, error) {
	startTime := time.Now()

	s.logger.Info(ctx, "[INGEST_START] Starting data ingestion", logging.Fields{
		"data_dir":   dataDir,
		"batch_size": batchSize,
		"stage":      "INITIALIZATION",
	})

	result := &IngestionResult{
		Errors: make([]string, 0),
	}

	// Read directory
	files, err := filepath.Glob(filepath.Join(dataDir, "*.txt"))
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no data files found in %s", dataDir)
	}

	result.TotalFiles = len(files)

	s.logger.Info(ctx, "[INGEST_FILES] Found data files", logging.Fields{
		"file_count": len(files),
		"stage":      "FILE_DISCOVERY",
	})

	// Process each file
	for _, filePath := range files {
		fileResult, err := s.ingestFile(ctx, filePath, batchSize)
		if err != nil {
			errMsg := fmt.Sprintf("failed to ingest %s: %v", filePath, err)
			result.Errors = append(result.Errors, errMsg)
			s.logger.Error(ctx, "[INGEST_FILE_ERROR] File ingestion failed", logging.Fields{
				"file_path": filePath,
				"stage":     "FILE_PROCESSING",
			}, err)
			s.metrics.RecordIngestionError("file_error")
			continue
		}

		result.TotalRecords += fileResult.TotalRecords
		result.SuccessfulRecords += fileResult.SuccessfulRecords
		result.FailedRecords += fileResult.FailedRecords

		s.logger.Info(ctx, "[INGEST_FILE_SUCCESS] File ingested successfully", logging.Fields{
			"file_path":         filePath,
			"total_records":     fileResult.TotalRecords,
			"successful_records": fileResult.SuccessfulRecords,
			"failed_records":    fileResult.FailedRecords,
			"stage":             "FILE_COMPLETE",
		})
	}

	result.Duration = time.Since(startTime)
	s.metrics.IngestionDuration.Observe(result.Duration.Seconds())

	s.logger.Info(ctx, "[INGEST_COMPLETE] Data ingestion completed", logging.Fields{
		"total_files":        result.TotalFiles,
		"total_records":      result.TotalRecords,
		"successful_records": result.SuccessfulRecords,
		"failed_records":     result.FailedRecords,
		"duration_seconds":   result.Duration.Seconds(),
		"records_per_second": float64(result.SuccessfulRecords) / result.Duration.Seconds(),
		"error_count":        len(result.Errors),
		"stage":              "COMPLETE",
	})

	return result, nil
}

// FileIngestionResult contains per-file ingestion statistics
type FileIngestionResult struct {
	TotalRecords      int
	SuccessfulRecords int
	FailedRecords     int
}

// ingestFile ingests a single weather data file
func (s *IngestionService) ingestFile(ctx context.Context, filePath string, batchSize int) (*FileIngestionResult, error) {
	// Extract station ID from filename
	fileName := filepath.Base(filePath)
	stationID := strings.TrimSuffix(fileName, filepath.Ext(fileName))

	// Create station if not exists
	station := &models.WeatherStation{
		StationID: stationID,
		State:     extractStateFromStationID(stationID),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	err := s.repo.CreateStation(ctx, station)
	if err != nil {
		return nil, fmt.Errorf("failed to create station: %w", err)
	}

	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	result := &FileIngestionResult{}
	batch := make([]*models.WeatherObservation, 0, batchSize)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		result.TotalRecords++

		line := scanner.Text()
		record, err := s.parseLine(line)
		if err != nil {
			result.FailedRecords++
			s.metrics.RecordIngestionError("parse_error")
			continue
		}

		observation, err := record.ToObservation(stationID)
		if err != nil {
			result.FailedRecords++
			s.metrics.RecordIngestionError("conversion_error")
			continue
		}

		batch = append(batch, observation)

		// Process batch when full
		if len(batch) >= batchSize {
			if err := s.repo.CreateObservationsBatch(ctx, batch); err != nil {
				return nil, fmt.Errorf("failed to insert batch: %w", err)
			}
			result.SuccessfulRecords += len(batch)
			batch = batch[:0]
		}
	}

	// Process remaining records
	if len(batch) > 0 {
		if err := s.repo.CreateObservationsBatch(ctx, batch); err != nil {
			return nil, fmt.Errorf("failed to insert final batch: %w", err)
		}
		result.SuccessfulRecords += len(batch)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return result, nil
}

// parseLine parses a single line from weather data file
// Format: YYYYMMDD\tMAX_TEMP\tMIN_TEMP\tPRECIP
func (s *IngestionService) parseLine(line string) (*models.RawWeatherRecord, error) {
	parts := strings.Split(line, "\t")
	if len(parts) != 4 {
		return nil, fmt.Errorf("invalid line format: expected 4 fields, got %d", len(parts))
	}

	maxTemp, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return nil, fmt.Errorf("invalid max temperature: %w", err)
	}

	minTemp, err := strconv.Atoi(strings.TrimSpace(parts[2]))
	if err != nil {
		return nil, fmt.Errorf("invalid min temperature: %w", err)
	}

	precip, err := strconv.Atoi(strings.TrimSpace(parts[3]))
	if err != nil {
		return nil, fmt.Errorf("invalid precipitation: %w", err)
	}

	return &models.RawWeatherRecord{
		Date:                 strings.TrimSpace(parts[0]),
		MaxTemperatureTenths: maxTemp,
		MinTemperatureTenths: minTemp,
		PrecipitationTenths:  precip,
	}, nil
}

// extractStateFromStationID extracts state code from station ID
// Assumes format: USC00XXXXXX where first 2 chars after USC00 might indicate state
// For simplicity, using first 2 chars of station ID
func extractStateFromStationID(stationID string) string {
	if len(stationID) >= 2 {
		return stationID[:2]
	}
	return "XX"
}
