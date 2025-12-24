package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Collector provides application metrics collection
type Collector struct {
	// API Metrics
	APIRequestsTotal    *prometheus.CounterVec
	APIRequestDuration  *prometheus.HistogramVec
	APIErrorsTotal      *prometheus.CounterVec

	// Ingestion Metrics
	IngestionRecordsTotal    prometheus.Counter
	IngestionDuration        prometheus.Histogram
	IngestionErrorsTotal     *prometheus.CounterVec
	IngestionBatchSize       prometheus.Histogram

	// Database Metrics
	DBQueryDuration     *prometheus.HistogramVec
	DBConnectionPool    *prometheus.GaugeVec
	DBErrorsTotal       *prometheus.CounterVec

	// Statistics Metrics
	StatsCacheHitRatio  prometheus.Gauge
	StatsCalculationDuration prometheus.Histogram

	// System Metrics
	ProcessingTimeMS    *prometheus.HistogramVec
	ActiveConnections   prometheus.Gauge
}

// NewCollector creates a new metrics collector
func NewCollector(namespace string) *Collector {
	return &Collector{
		APIRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "api_requests_total",
				Help:      "Total number of API requests by endpoint, method, and status",
			},
			[]string{"endpoint", "method", "status"},
		),

		APIRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "api_request_duration_seconds",
				Help:      "API request duration in seconds",
				Buckets:   []float64{0.001, 0.005, 0.01, 0.02, 0.05, 0.1, 0.2, 0.5, 1.0, 2.0, 5.0},
			},
			[]string{"endpoint"},
		),

		APIErrorsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "api_errors_total",
				Help:      "Total number of API errors by type",
			},
			[]string{"error_type", "endpoint"},
		),

		IngestionRecordsTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "ingestion_records_processed_total",
				Help:      "Total number of weather records ingested",
			},
		),

		IngestionDuration: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "ingestion_duration_seconds",
				Help:      "Duration of ingestion operations in seconds",
				Buckets:   []float64{1, 5, 10, 30, 60, 120, 300, 600},
			},
		),

		IngestionErrorsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "ingestion_errors_total",
				Help:      "Total number of ingestion errors by type",
			},
			[]string{"error_type"},
		),

		IngestionBatchSize: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "ingestion_batch_size",
				Help:      "Number of records per batch during ingestion",
				Buckets:   []float64{10, 50, 100, 500, 1000, 5000, 10000},
			},
		),

		DBQueryDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "db_query_duration_seconds",
				Help:      "Database query duration in seconds by query type",
				Buckets:   []float64{0.001, 0.002, 0.005, 0.01, 0.02, 0.05, 0.1, 0.2, 0.5},
			},
			[]string{"query_type"},
		),

		DBConnectionPool: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "db_connection_pool",
				Help:      "Database connection pool statistics",
			},
			[]string{"state"}, // "in_use", "idle", "total"
		),

		DBErrorsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "db_errors_total",
				Help:      "Total number of database errors by type",
			},
			[]string{"error_type"},
		),

		StatsCacheHitRatio: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "stats_cache_hit_ratio",
				Help:      "Cache hit ratio for statistics queries",
			},
		),

		StatsCalculationDuration: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "stats_calculation_duration_seconds",
				Help:      "Duration of statistics calculation in seconds",
				Buckets:   []float64{0.005, 0.01, 0.02, 0.05, 0.1, 0.2, 0.5, 1.0},
			},
		),

		ProcessingTimeMS: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "processing_time_milliseconds",
				Help:      "Processing time in milliseconds by operation",
				Buckets:   []float64{1, 5, 10, 20, 50, 100, 200, 500, 1000, 2000, 5000},
			},
			[]string{"operation"},
		),

		ActiveConnections: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "active_connections",
				Help:      "Number of active client connections",
			},
		),
	}
}

// Timer provides timing functionality for operations
type Timer struct {
	start    time.Time
	observer prometheus.Observer
}

// NewTimer creates a new timer
func (c *Collector) NewTimer(histogram prometheus.Observer) *Timer {
	return &Timer{
		start:    time.Now(),
		observer: histogram,
	}
}

// ObserveDuration records the elapsed time since timer creation
func (t *Timer) ObserveDuration() time.Duration {
	duration := time.Since(t.start)
	if t.observer != nil {
		t.observer.Observe(duration.Seconds())
	}
	return duration
}

// RecordAPIRequest increments API request counter
func (c *Collector) RecordAPIRequest(endpoint, method, status string) {
	c.APIRequestsTotal.WithLabelValues(endpoint, method, status).Inc()
}

// RecordAPIError increments API error counter
func (c *Collector) RecordAPIError(errorType, endpoint string) {
	c.APIErrorsTotal.WithLabelValues(errorType, endpoint).Inc()
}

// RecordIngestionError increments ingestion error counter
func (c *Collector) RecordIngestionError(errorType string) {
	c.IngestionErrorsTotal.WithLabelValues(errorType).Inc()
}

// RecordDBError increments database error counter
func (c *Collector) RecordDBError(errorType string) {
	c.DBErrorsTotal.WithLabelValues(errorType).Inc()
}

// UpdateDBConnectionPool updates database connection pool metrics
func (c *Collector) UpdateDBConnectionPool(inUse, idle, total int) {
	c.DBConnectionPool.WithLabelValues("in_use").Set(float64(inUse))
	c.DBConnectionPool.WithLabelValues("idle").Set(float64(idle))
	c.DBConnectionPool.WithLabelValues("total").Set(float64(total))
}
