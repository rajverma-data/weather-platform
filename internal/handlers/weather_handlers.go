package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	"weather-platform/internal/repository"
	"weather-platform/internal/services"
	"weather-platform/pkg/logging"
	"weather-platform/pkg/metrics"
)

// WeatherHandler handles weather API endpoints
type WeatherHandler struct {
	weatherService *services.WeatherService
	statsService   *services.StatisticsService
	logger         *logging.StructuredLogger
	metrics        *metrics.Collector
}

// NewWeatherHandler creates a new weather handler
func NewWeatherHandler(
	weatherService *services.WeatherService,
	statsService *services.StatisticsService,
	logger *logging.StructuredLogger,
	metricsCollector *metrics.Collector,
) *WeatherHandler {
	return &WeatherHandler{
		weatherService: weatherService,
		statsService:   statsService,
		logger:         logger,
		metrics:        metricsCollector,
	}
}

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// PaginatedResponse represents a paginated API response
type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Total      int         `json:"total"`
	Page       int         `json:"page"`
	Limit      int         `json:"limit"`
	TotalPages int         `json:"total_pages"`
}

// GetObservations handles GET /api/weather
func (h *WeatherHandler) GetObservations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	startTime := time.Now()

	defer func() {
		duration := time.Since(startTime)
		h.metrics.APIRequestDuration.WithLabelValues("/api/weather").Observe(duration.Seconds())
	}()

	// Parse query parameters
	stationID := r.URL.Query().Get("station_id")
	startDateStr := r.URL.Query().Get("start_date")
	endDateStr := r.URL.Query().Get("end_date")
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	// Default pagination
	page := 1
	limit := 100

	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	offset := (page - 1) * limit

	// Build filter
	filter := repository.ObservationFilter{
		Limit:  limit,
		Offset: offset,
	}

	if stationID != "" {
		filter.StationID = &stationID
	}

	if startDateStr != "" {
		startDate, err := time.Parse("2006-01-02", startDateStr)
		if err != nil {
			h.sendError(w, r, "invalid start_date format, expected YYYY-MM-DD", http.StatusBadRequest)
			return
		}
		filter.StartDate = &startDate
	}

	if endDateStr != "" {
		endDate, err := time.Parse("2006-01-02", endDateStr)
		if err != nil {
			h.sendError(w, r, "invalid end_date format, expected YYYY-MM-DD", http.StatusBadRequest)
			return
		}
		filter.EndDate = &endDate
	}

	// Get observations
	observations, total, err := h.weatherService.GetObservations(ctx, filter)
	if err != nil {
		h.logger.Error(ctx, "[API_GET_OBSERVATIONS_ERROR] Failed to get observations", logging.Fields{
			"filter": filter,
		}, err)
		h.metrics.RecordAPIError("internal_error", "/api/weather")
		h.sendError(w, r, "failed to retrieve observations", http.StatusInternalServerError)
		return
	}

	totalPages := (total + limit - 1) / limit

	response := PaginatedResponse{
		Data:       observations,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}

	h.metrics.RecordAPIRequest("/api/weather", "GET", "200")
	h.sendJSON(w, response, http.StatusOK)
}

// GetStatistics handles GET /api/weather/stats
func (h *WeatherHandler) GetStatistics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	startTime := time.Now()

	defer func() {
		duration := time.Since(startTime)
		h.metrics.APIRequestDuration.WithLabelValues("/api/weather/stats").Observe(duration.Seconds())
	}()

	// Parse query parameters
	stationID := r.URL.Query().Get("station_id")
	yearStr := r.URL.Query().Get("year")
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	// Default pagination
	page := 1
	limit := 100

	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	offset := (page - 1) * limit

	// Build filter
	filter := repository.StatisticsFilter{
		Limit:  limit,
		Offset: offset,
	}

	if stationID != "" {
		filter.StationID = &stationID
	}

	if yearStr != "" {
		year, err := strconv.Atoi(yearStr)
		if err != nil || year < 1985 || year > 2014 {
			h.sendError(w, r, "invalid year, expected integer between 1985 and 2014", http.StatusBadRequest)
			return
		}
		filter.Year = &year
	}

	// Get statistics
	statistics, total, err := h.statsService.GetStatistics(ctx, filter)
	if err != nil {
		h.logger.Error(ctx, "[API_GET_STATISTICS_ERROR] Failed to get statistics", logging.Fields{
			"filter": filter,
		}, err)
		h.metrics.RecordAPIError("internal_error", "/api/weather/stats")
		h.sendError(w, r, "failed to retrieve statistics", http.StatusInternalServerError)
		return
	}

	totalPages := (total + limit - 1) / limit

	response := PaginatedResponse{
		Data:       statistics,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}

	h.metrics.RecordAPIRequest("/api/weather/stats", "GET", "200")
	h.sendJSON(w, response, http.StatusOK)
}

// HealthCheck handles GET /health
func (h *WeatherHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	status := map[string]string{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	h.logger.Debug(ctx, "[HEALTH_CHECK] Health check requested", logging.Fields{})
	h.sendJSON(w, status, http.StatusOK)
}

// sendJSON sends a JSON response
func (h *WeatherHandler) sendJSON(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// sendError sends an error response
func (h *WeatherHandler) sendError(w http.ResponseWriter, r *http.Request, message string, statusCode int) {
	h.metrics.RecordAPIRequest(r.URL.Path, r.Method, strconv.Itoa(statusCode))

	response := ErrorResponse{
		Error:   http.StatusText(statusCode),
		Message: message,
		Code:    statusCode,
	}

	h.sendJSON(w, response, statusCode)
}

// RegisterRoutes registers all weather API routes
func (h *WeatherHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/weather", h.GetObservations).Methods("GET")
	router.HandleFunc("/api/weather/stats", h.GetStatistics).Methods("GET")
	router.HandleFunc("/health", h.HealthCheck).Methods("GET")
}
