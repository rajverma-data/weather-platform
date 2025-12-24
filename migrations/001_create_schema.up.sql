-- Weather Platform Database Schema
-- Migration: 001 - Create base schema with proper normalization and indexing

-- Enable required PostgreSQL extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_stat_statements";

-- Weather stations table with full normalization
CREATE TABLE weather_stations (
    station_id VARCHAR(50) PRIMARY KEY,
    state VARCHAR(2) NOT NULL CHECK (LENGTH(state) = 2),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT valid_station_id CHECK (LENGTH(station_id) > 0)
);

-- Index for state-based queries
CREATE INDEX idx_weather_stations_state ON weather_stations(state);

-- Raw weather observations with proper indexing
CREATE TABLE weather_observations (
    id BIGSERIAL PRIMARY KEY,
    station_id VARCHAR(50) NOT NULL REFERENCES weather_stations(station_id) ON DELETE CASCADE,
    observation_date DATE NOT NULL,
    max_temperature_celsius DECIMAL(5,2),  -- NULL for -9999 values
    min_temperature_celsius DECIMAL(5,2),  -- NULL for -9999 values
    precipitation_cm DECIMAL(8,4),         -- NULL for -9999 values
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Constraints for data quality
    CONSTRAINT unique_station_date UNIQUE(station_id, observation_date),
    CONSTRAINT valid_observation_date CHECK (observation_date >= '1900-01-01' AND observation_date <= CURRENT_DATE),
    CONSTRAINT valid_max_temp CHECK (max_temperature_celsius IS NULL OR (max_temperature_celsius >= -100 AND max_temperature_celsius <= 100)),
    CONSTRAINT valid_min_temp CHECK (min_temperature_celsius IS NULL OR (min_temperature_celsius >= -100 AND min_temperature_celsius <= 100)),
    CONSTRAINT valid_precipitation CHECK (precipitation_cm IS NULL OR precipitation_cm >= 0),
    CONSTRAINT valid_temp_range CHECK (
        max_temperature_celsius IS NULL OR
        min_temperature_celsius IS NULL OR
        max_temperature_celsius >= min_temperature_celsius
    )
);

-- Covering index for station-date range queries (<10ms target)
CREATE INDEX idx_weather_obs_station_date ON weather_observations(station_id, observation_date DESC);

-- Index for date range queries across all stations
CREATE INDEX idx_weather_obs_date_range ON weather_observations(observation_date DESC);

-- Composite index for aggregation queries
CREATE INDEX idx_weather_obs_station_date_temps ON weather_observations(station_id, observation_date)
    INCLUDE (max_temperature_celsius, min_temperature_celsius, precipitation_cm);

-- Pre-calculated statistics for performance
CREATE TABLE weather_statistics (
    id BIGSERIAL PRIMARY KEY,
    station_id VARCHAR(50) NOT NULL REFERENCES weather_stations(station_id) ON DELETE CASCADE,
    year INTEGER NOT NULL,
    avg_max_temperature_celsius DECIMAL(5,2),
    avg_min_temperature_celsius DECIMAL(5,2),
    total_precipitation_cm DECIMAL(8,4),
    observation_count INTEGER NOT NULL,
    valid_max_temp_count INTEGER NOT NULL DEFAULT 0,
    valid_min_temp_count INTEGER NOT NULL DEFAULT 0,
    valid_precipitation_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Constraints for data quality
    CONSTRAINT unique_station_year UNIQUE(station_id, year),
    CONSTRAINT valid_year CHECK (year >= 1900 AND year <= EXTRACT(YEAR FROM CURRENT_DATE)),
    CONSTRAINT valid_observation_count CHECK (observation_count >= 0),
    CONSTRAINT valid_counts CHECK (
        valid_max_temp_count >= 0 AND
        valid_min_temp_count >= 0 AND
        valid_precipitation_count >= 0 AND
        valid_max_temp_count <= observation_count AND
        valid_min_temp_count <= observation_count AND
        valid_precipitation_count <= observation_count
    )
);

-- Indexes for statistics queries
CREATE INDEX idx_weather_stats_station_year ON weather_statistics(station_id, year DESC);
CREATE INDEX idx_weather_stats_year ON weather_statistics(year DESC);

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Triggers for automatic timestamp updates
CREATE TRIGGER update_weather_stations_updated_at
    BEFORE UPDATE ON weather_stations
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_weather_statistics_updated_at
    BEFORE UPDATE ON weather_statistics
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Comments for documentation
COMMENT ON TABLE weather_stations IS 'Weather station metadata with state information';
COMMENT ON TABLE weather_observations IS 'Raw weather observations with temperature and precipitation data';
COMMENT ON TABLE weather_statistics IS 'Pre-calculated yearly statistics for performance optimization';

COMMENT ON COLUMN weather_observations.max_temperature_celsius IS 'Maximum temperature in Celsius, NULL for missing values (-9999)';
COMMENT ON COLUMN weather_observations.min_temperature_celsius IS 'Minimum temperature in Celsius, NULL for missing values (-9999)';
COMMENT ON COLUMN weather_observations.precipitation_cm IS 'Precipitation in centimeters, NULL for missing values (-9999)';

COMMENT ON COLUMN weather_statistics.observation_count IS 'Total number of observations for the year';
COMMENT ON COLUMN weather_statistics.valid_max_temp_count IS 'Count of non-NULL max temperature observations';
COMMENT ON COLUMN weather_statistics.valid_min_temp_count IS 'Count of non-NULL min temperature observations';
COMMENT ON COLUMN weather_statistics.valid_precipitation_count IS 'Count of non-NULL precipitation observations';
