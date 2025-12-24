package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"weather-platform/internal/config"
	"weather-platform/internal/repository"
	"weather-platform/internal/services"
	"weather-platform/pkg/database"
	"weather-platform/pkg/logging"
	"weather-platform/pkg/metrics"
)

func main() {
	// Parse command-line flags
	dataDir := flag.String("data-dir", "./wx_data", "Directory containing weather data files")
	batchSize := flag.Int("batch-size", 1000, "Number of records to process in each batch")
	calculateStats := flag.Bool("calculate-stats", false, "Calculate statistics after ingestion")
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logLevel := logging.InfoLevel
	if cfg.Logging.Level == "debug" {
		logLevel = logging.DebugLevel
	}

	logger := logging.NewStructuredLogger("weather-ingester", "1.0.0", logLevel)

	ctx := context.Background()
	logger.Info(ctx, "[INGESTER_START] Starting weather data ingestion", logging.Fields{
		"version":          "1.0.0",
		"data_dir":         *dataDir,
		"batch_size":       *batchSize,
		"calculate_stats":  *calculateStats,
	})

	// Initialize metrics collector
	metricsCollector := metrics.NewCollector("weather_ingester")

	// Initialize database
	dbConfig := &database.Config{
		Host:            cfg.Database.Host,
		Port:            cfg.Database.Port,
		User:            cfg.Database.User,
		Password:        cfg.Database.Password,
		Database:        cfg.Database.Database,
		SSLMode:         cfg.Database.SSLMode,
		MaxOpenConns:    cfg.Database.MaxOpenConns,
		MaxIdleConns:    cfg.Database.MaxIdleConns,
		ConnMaxLifetime: cfg.Database.ConnMaxLifetime,
		ConnMaxIdleTime: cfg.Database.ConnMaxIdleTime,
	}

	db, err := database.NewPostgresDB(dbConfig, logger, metricsCollector)
	if err != nil {
		logger.Fatal(ctx, "[INGESTER_ERROR] Failed to connect to database", logging.Fields{}, err)
	}
	defer db.Close()

	// Initialize repository
	weatherRepo := repository.NewWeatherRepository(db, logger, metricsCollector)

	// Initialize services
	ingestionService := services.NewIngestionService(weatherRepo, logger, metricsCollector)
	statsService := services.NewStatisticsService(weatherRepo, logger, metricsCollector)

	// Ingest data
	result, err := ingestionService.IngestDirectory(ctx, *dataDir, *batchSize)
	if err != nil {
		logger.Fatal(ctx, "[INGESTION_ERROR] Ingestion failed", logging.Fields{
			"error": err.Error(),
		}, err)
	}

	// Print results
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("INGESTION COMPLETE")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("Total Files:        %d\n", result.TotalFiles)
	fmt.Printf("Total Records:      %d\n", result.TotalRecords)
	fmt.Printf("Successful Records: %d\n", result.SuccessfulRecords)
	fmt.Printf("Failed Records:     %d\n", result.FailedRecords)
	fmt.Printf("Duration:           %v\n", result.Duration)
	fmt.Printf("Records/Second:     %.2f\n", float64(result.SuccessfulRecords)/result.Duration.Seconds())

	if len(result.Errors) > 0 {
		fmt.Printf("\nErrors (%d):\n", len(result.Errors))
		for i, errMsg := range result.Errors {
			if i < 10 {
				fmt.Printf("  - %s\n", errMsg)
			}
		}
		if len(result.Errors) > 10 {
			fmt.Printf("  ... and %d more errors\n", len(result.Errors)-10)
		}
	}

	// Calculate statistics if requested
	if *calculateStats {
		fmt.Println("\n" + strings.Repeat("=", 80))
		fmt.Println("CALCULATING STATISTICS")
		fmt.Println(strings.Repeat("=", 80))

		if err := statsService.CalculateAllStatistics(ctx); err != nil {
			logger.Error(ctx, "[STATS_ERROR] Statistics calculation failed", logging.Fields{}, err)
			fmt.Printf("Statistics calculation failed: %v\n", err)
		} else {
			fmt.Println("Statistics calculation completed successfully")
		}
	}

	logger.Info(ctx, "[INGESTER_COMPLETE] Ingestion completed successfully", logging.Fields{
		"total_records":      result.TotalRecords,
		"successful_records": result.SuccessfulRecords,
		"failed_records":     result.FailedRecords,
		"duration_seconds":   result.Duration.Seconds(),
	})
}
