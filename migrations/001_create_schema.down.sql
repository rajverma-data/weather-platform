-- Rollback migration 001 - Drop base schema

DROP TRIGGER IF EXISTS update_weather_statistics_updated_at ON weather_statistics;
DROP TRIGGER IF EXISTS update_weather_stations_updated_at ON weather_stations;

DROP FUNCTION IF EXISTS update_updated_at_column();

DROP INDEX IF EXISTS idx_weather_stats_year;
DROP INDEX IF EXISTS idx_weather_stats_station_year;

DROP TABLE IF EXISTS weather_statistics CASCADE;

DROP INDEX IF EXISTS idx_weather_obs_station_date_temps;
DROP INDEX IF EXISTS idx_weather_obs_date_range;
DROP INDEX IF EXISTS idx_weather_obs_station_date;

DROP TABLE IF EXISTS weather_observations CASCADE;

DROP INDEX IF EXISTS idx_weather_stations_state;

DROP TABLE IF EXISTS weather_stations CASCADE;

DROP EXTENSION IF EXISTS "pg_stat_statements";
DROP EXTENSION IF EXISTS "uuid-ossp";
