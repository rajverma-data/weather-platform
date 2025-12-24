package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"weather-platform/pkg/logging"
	"weather-platform/pkg/metrics"
)

// Config holds database connection configuration
type Config struct {
	Host            string
	Port            int
	User            string
	Password        string
	Database        string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// PostgresDB wraps sqlx.DB with monitoring and metrics
type PostgresDB struct {
	db      *sqlx.DB
	logger  *logging.StructuredLogger
	metrics *metrics.Collector
	config  *Config
}

// NewPostgresDB creates a new PostgreSQL database connection
func NewPostgresDB(cfg *Config, logger *logging.StructuredLogger, metricsCollector *metrics.Collector) (*PostgresDB, error) {
	// Build connection string
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host,
		cfg.Port,
		cfg.User,
		cfg.Password,
		cfg.Database,
		cfg.SSLMode,
	)

	// Open database connection
	db, err := sqlx.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info(context.Background(), "[DB_INIT] PostgreSQL connection established", logging.Fields{
		"host":              cfg.Host,
		"port":              cfg.Port,
		"database":          cfg.Database,
		"max_open_conns":    cfg.MaxOpenConns,
		"max_idle_conns":    cfg.MaxIdleConns,
		"conn_max_lifetime": cfg.ConnMaxLifetime.String(),
	})

	pgDB := &PostgresDB{
		db:      db,
		logger:  logger,
		metrics: metricsCollector,
		config:  cfg,
	}

	// Start monitoring connection pool
	go pgDB.monitorConnectionPool()

	return pgDB, nil
}

// Close closes the database connection
func (p *PostgresDB) Close() error {
	p.logger.Info(context.Background(), "[DB_CLOSE] Closing database connection", logging.Fields{
		"database": p.config.Database,
	})
	return p.db.Close()
}

// DB returns the underlying sqlx.DB instance
func (p *PostgresDB) DB() *sqlx.DB {
	return p.db
}

// QueryContext executes a query with context and metrics
func (p *PostgresDB) QueryContext(ctx context.Context, queryType, query string, args ...interface{}) (*sqlx.Rows, error) {
	timer := time.Now()
	defer func() {
		duration := time.Since(timer)
		p.metrics.DBQueryDuration.WithLabelValues(queryType).Observe(duration.Seconds())

		p.logger.Debug(ctx, "[DB_QUERY] Query executed", logging.Fields{
			"query_type":       queryType,
			"duration_ms":      duration.Milliseconds(),
			"query":            query,
		})
	}()

	rows, err := p.db.QueryxContext(ctx, query, args...)
	if err != nil {
		p.metrics.RecordDBError("query_error")
		p.logger.Error(ctx, "[DB_QUERY_ERROR] Query failed", logging.Fields{
			"query_type": queryType,
			"query":      query,
		}, err)
		return nil, err
	}

	return rows, nil
}

// ExecContext executes a command with context and metrics
func (p *PostgresDB) ExecContext(ctx context.Context, queryType, query string, args ...interface{}) (sql.Result, error) {
	timer := time.Now()
	defer func() {
		duration := time.Since(timer)
		p.metrics.DBQueryDuration.WithLabelValues(queryType).Observe(duration.Seconds())

		p.logger.Debug(ctx, "[DB_EXEC] Command executed", logging.Fields{
			"query_type":  queryType,
			"duration_ms": duration.Milliseconds(),
		})
	}()

	result, err := p.db.ExecContext(ctx, query, args...)
	if err != nil {
		p.metrics.RecordDBError("exec_error")
		p.logger.Error(ctx, "[DB_EXEC_ERROR] Command failed", logging.Fields{
			"query_type": queryType,
		}, err)
		return nil, err
	}

	return result, nil
}

// GetContext executes a query that returns a single row
func (p *PostgresDB) GetContext(ctx context.Context, queryType string, dest interface{}, query string, args ...interface{}) error {
	timer := time.Now()
	defer func() {
		duration := time.Since(timer)
		p.metrics.DBQueryDuration.WithLabelValues(queryType).Observe(duration.Seconds())
	}()

	err := p.db.GetContext(ctx, dest, query, args...)
	if err != nil && err != sql.ErrNoRows {
		p.metrics.RecordDBError("get_error")
		p.logger.Error(ctx, "[DB_GET_ERROR] Get query failed", logging.Fields{
			"query_type": queryType,
		}, err)
	}

	return err
}

// SelectContext executes a query that returns multiple rows
func (p *PostgresDB) SelectContext(ctx context.Context, queryType string, dest interface{}, query string, args ...interface{}) error {
	timer := time.Now()
	defer func() {
		duration := time.Since(timer)
		p.metrics.DBQueryDuration.WithLabelValues(queryType).Observe(duration.Seconds())
	}()

	err := p.db.SelectContext(ctx, dest, query, args...)
	if err != nil {
		p.metrics.RecordDBError("select_error")
		p.logger.Error(ctx, "[DB_SELECT_ERROR] Select query failed", logging.Fields{
			"query_type": queryType,
		}, err)
		return err
	}

	return nil
}

// BeginTx begins a new transaction
func (p *PostgresDB) BeginTx(ctx context.Context) (*sqlx.Tx, error) {
	tx, err := p.db.BeginTxx(ctx, &sql.TxOptions{
		Isolation: sql.LevelSerializable,
	})
	if err != nil {
		p.metrics.RecordDBError("transaction_begin_error")
		p.logger.Error(ctx, "[DB_TX_ERROR] Failed to begin transaction", logging.Fields{}, err)
		return nil, err
	}

	return tx, nil
}

// monitorConnectionPool periodically updates connection pool metrics
func (p *PostgresDB) monitorConnectionPool() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		stats := p.db.Stats()

		p.metrics.UpdateDBConnectionPool(
			stats.InUse,
			stats.Idle,
			stats.OpenConnections,
		)

		// Log warning if connection pool is near capacity
		utilization := float64(stats.InUse) / float64(p.config.MaxOpenConns)
		if utilization > 0.8 {
			p.logger.Warn(context.Background(), "[DB_POOL_WARNING] Connection pool utilization high", logging.Fields{
				"in_use":       stats.InUse,
				"idle":         stats.Idle,
				"total":        stats.OpenConnections,
				"max_open":     p.config.MaxOpenConns,
				"utilization":  fmt.Sprintf("%.2f%%", utilization*100),
			})
		}
	}
}

// HealthCheck performs a database health check
func (p *PostgresDB) HealthCheck(ctx context.Context) error {
	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := p.db.PingContext(pingCtx); err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}

	return nil
}
