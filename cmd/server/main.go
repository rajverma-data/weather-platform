package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"weather-platform/internal/config"
	"weather-platform/internal/handlers"
	"weather-platform/internal/repository"
	"weather-platform/internal/services"
	"weather-platform/pkg/database"
	"weather-platform/pkg/logging"
	"weather-platform/pkg/metrics"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logLevel := logging.InfoLevel
	if cfg.Logging.Level == "debug" {
		logLevel = logging.DebugLevel
	}

	logger := logging.NewStructuredLogger("weather-api", "1.0.0", logLevel)

	ctx := context.Background()
	logger.Info(ctx, "[STARTUP] Starting weather platform API server", logging.Fields{
		"version":     "1.0.0",
		"server_host": cfg.Server.Host,
		"server_port": cfg.Server.Port,
		"db_host":     cfg.Database.Host,
		"db_name":     cfg.Database.Database,
	})

	// Initialize metrics collector
	metricsCollector := metrics.NewCollector("weather_platform")

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
		logger.Fatal(ctx, "[STARTUP_ERROR] Failed to connect to database", logging.Fields{}, err)
	}
	defer db.Close()

	// Initialize repository
	weatherRepo := repository.NewWeatherRepository(db, logger, metricsCollector)

	// Initialize services
	weatherService := services.NewWeatherService(weatherRepo, logger, metricsCollector)
	statsService := services.NewStatisticsService(weatherRepo, logger, metricsCollector)

	// Initialize handlers
	weatherHandler := handlers.NewWeatherHandler(weatherService, statsService, logger, metricsCollector)

	// Setup router
	router := mux.NewRouter()

	// Register routes
	weatherHandler.RegisterRoutes(router)

	// Prometheus metrics endpoint
	router.Handle("/metrics", promhttp.Handler())

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start server in goroutine
	go func() {
		logger.Info(ctx, "[SERVER_START] HTTP server listening", logging.Fields{
			"address": server.Addr,
		})

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal(ctx, "[SERVER_ERROR] Server failed", logging.Fields{}, err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info(ctx, "[SHUTDOWN] Shutting down server...", logging.Fields{})

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error(ctx, "[SHUTDOWN_ERROR] Server forced to shutdown", logging.Fields{}, err)
	}

	logger.Info(ctx, "[SHUTDOWN_COMPLETE] Server stopped", logging.Fields{})
}
