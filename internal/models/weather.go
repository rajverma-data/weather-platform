package models

import (
	"time"
)

// WeatherStation represents a weather monitoring station
// Complies with §7 (Layer Algebra) - Domain layer pure data structures
type WeatherStation struct {
	StationID string    `json:"station_id" db:"station_id"`
	State     string    `json:"state" db:"state"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// WeatherObservation represents a single weather data point
// NULL values represented as pointers for -9999 handling
type WeatherObservation struct {
	ID                      int64      `json:"id" db:"id"`
	StationID               string     `json:"station_id" db:"station_id"`
	ObservationDate         time.Time  `json:"observation_date" db:"observation_date"`
	MaxTemperatureCelsius   *float64   `json:"max_temperature_celsius,omitempty" db:"max_temperature_celsius"`
	MinTemperatureCelsius   *float64   `json:"min_temperature_celsius,omitempty" db:"min_temperature_celsius"`
	PrecipitationCm         *float64   `json:"precipitation_cm,omitempty" db:"precipitation_cm"`
	CreatedAt               time.Time  `json:"created_at" db:"created_at"`
}

// WeatherStatistics represents pre-calculated yearly statistics
// Optimized for query performance (§8 Performance Envelope)
type WeatherStatistics struct {
	ID                        int64      `json:"id" db:"id"`
	StationID                 string     `json:"station_id" db:"station_id"`
	Year                      int        `json:"year" db:"year"`
	AvgMaxTemperatureCelsius  *float64   `json:"avg_max_temperature_celsius,omitempty" db:"avg_max_temperature_celsius"`
	AvgMinTemperatureCelsius  *float64   `json:"avg_min_temperature_celsius,omitempty" db:"avg_min_temperature_celsius"`
	TotalPrecipitationCm      *float64   `json:"total_precipitation_cm,omitempty" db:"total_precipitation_cm"`
	ObservationCount          int        `json:"observation_count" db:"observation_count"`
	ValidMaxTempCount         int        `json:"valid_max_temp_count" db:"valid_max_temp_count"`
	ValidMinTempCount         int        `json:"valid_min_temp_count" db:"valid_min_temp_count"`
	ValidPrecipitationCount   int        `json:"valid_precipitation_count" db:"valid_precipitation_count"`
	CreatedAt                 time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt                 time.Time  `json:"updated_at" db:"updated_at"`
}

// RawWeatherRecord represents a single line from input data files
// Used during ingestion process
type RawWeatherRecord struct {
	Date                string
	MaxTemperatureTenths int  // Raw value in 0.1°C (may be -9999)
	MinTemperatureTenths int  // Raw value in 0.1°C (may be -9999)
	PrecipitationTenths  int  // Raw value in 0.1mm (may be -9999)
}

// ToObservation converts RawWeatherRecord to WeatherObservation
// Handles -9999 sentinel values and unit conversions
// Complies with §4 (Complete Implementation) - no TODOs or partial implementation
func (r *RawWeatherRecord) ToObservation(stationID string) (*WeatherObservation, error) {
	// Parse date
	date, err := time.Parse("20060102", r.Date)
	if err != nil {
		return nil, &ValidationError{
			Field:   "date",
			Value:   r.Date,
			Message: "invalid date format, expected YYYYMMDD",
		}
	}

	obs := &WeatherObservation{
		StationID:       stationID,
		ObservationDate: date,
		CreatedAt:       time.Now().UTC(),
	}

	// Convert max temperature: 0.1°C to °C, handle -9999 as NULL
	if r.MaxTemperatureTenths != -9999 {
		temp := float64(r.MaxTemperatureTenths) / 10.0
		obs.MaxTemperatureCelsius = &temp
	}

	// Convert min temperature: 0.1°C to °C, handle -9999 as NULL
	if r.MinTemperatureTenths != -9999 {
		temp := float64(r.MinTemperatureTenths) / 10.0
		obs.MinTemperatureCelsius = &temp
	}

	// Convert precipitation: 0.1mm to cm, handle -9999 as NULL
	if r.PrecipitationTenths != -9999 {
		precip := float64(r.PrecipitationTenths) / 100.0 // 0.1mm to cm
		obs.PrecipitationCm = &precip
	}

	return obs, nil
}

// ValidationError represents a data validation error
// Complies with §13 (Error Algebra) - explicit error classification
type ValidationError struct {
	Field   string
	Value   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// IsTransient returns false as validation errors are permanent
func (e *ValidationError) IsTransient() bool {
	return false
}
