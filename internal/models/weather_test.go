package models

import (
	"testing"
	"time"
)

// TestRawWeatherRecord_ToObservation tests the conversion logic
// This covers 100% of mathematical operations as required by §10
func TestRawWeatherRecord_ToObservation(t *testing.T) {
	tests := []struct {
		name        string
		record      RawWeatherRecord
		stationID   string
		wantErr     bool
		checkValues func(*testing.T, *WeatherObservation)
	}{
		{
			name: "valid record with all values",
			record: RawWeatherRecord{
				Date:                "20230115",
				MaxTemperatureTenths: 250,
				MinTemperatureTenths: 150,
				PrecipitationTenths:  100,
			},
			stationID: "TEST001",
			wantErr:   false,
			checkValues: func(t *testing.T, obs *WeatherObservation) {
				if obs.StationID != "TEST001" {
					t.Errorf("StationID = %v, want %v", obs.StationID, "TEST001")
				}

				expectedDate := time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)
				if !obs.ObservationDate.Equal(expectedDate) {
					t.Errorf("ObservationDate = %v, want %v", obs.ObservationDate, expectedDate)
				}

				if obs.MaxTemperatureCelsius == nil {
					t.Error("MaxTemperatureCelsius should not be nil")
				} else if *obs.MaxTemperatureCelsius != 25.0 {
					t.Errorf("MaxTemperatureCelsius = %v, want %v", *obs.MaxTemperatureCelsius, 25.0)
				}

				if obs.MinTemperatureCelsius == nil {
					t.Error("MinTemperatureCelsius should not be nil")
				} else if *obs.MinTemperatureCelsius != 15.0 {
					t.Errorf("MinTemperatureCelsius = %v, want %v", *obs.MinTemperatureCelsius, 15.0)
				}

				if obs.PrecipitationCm == nil {
					t.Error("PrecipitationCm should not be nil")
				} else if *obs.PrecipitationCm != 1.0 {
					t.Errorf("PrecipitationCm = %v, want %v", *obs.PrecipitationCm, 1.0)
				}
			},
		},
		{
			name: "missing value (-9999) for max temperature",
			record: RawWeatherRecord{
				Date:                "20230115",
				MaxTemperatureTenths: -9999,
				MinTemperatureTenths: 150,
				PrecipitationTenths:  100,
			},
			stationID: "TEST001",
			wantErr:   false,
			checkValues: func(t *testing.T, obs *WeatherObservation) {
				if obs.MaxTemperatureCelsius != nil {
					t.Error("MaxTemperatureCelsius should be nil for -9999")
				}

				if obs.MinTemperatureCelsius == nil {
					t.Error("MinTemperatureCelsius should not be nil")
				} else if *obs.MinTemperatureCelsius != 15.0 {
					t.Errorf("MinTemperatureCelsius = %v, want %v", *obs.MinTemperatureCelsius, 15.0)
				}
			},
		},
		{
			name: "missing value (-9999) for min temperature",
			record: RawWeatherRecord{
				Date:                "20230115",
				MaxTemperatureTenths: 250,
				MinTemperatureTenths: -9999,
				PrecipitationTenths:  100,
			},
			stationID: "TEST001",
			wantErr:   false,
			checkValues: func(t *testing.T, obs *WeatherObservation) {
				if obs.MinTemperatureCelsius != nil {
					t.Error("MinTemperatureCelsius should be nil for -9999")
				}

				if obs.MaxTemperatureCelsius == nil {
					t.Error("MaxTemperatureCelsius should not be nil")
				} else if *obs.MaxTemperatureCelsius != 25.0 {
					t.Errorf("MaxTemperatureCelsius = %v, want %v", *obs.MaxTemperatureCelsius, 25.0)
				}
			},
		},
		{
			name: "missing value (-9999) for precipitation",
			record: RawWeatherRecord{
				Date:                "20230115",
				MaxTemperatureTenths: 250,
				MinTemperatureTenths: 150,
				PrecipitationTenths:  -9999,
			},
			stationID: "TEST001",
			wantErr:   false,
			checkValues: func(t *testing.T, obs *WeatherObservation) {
				if obs.PrecipitationCm != nil {
					t.Error("PrecipitationCm should be nil for -9999")
				}
			},
		},
		{
			name: "all missing values (-9999)",
			record: RawWeatherRecord{
				Date:                "20230115",
				MaxTemperatureTenths: -9999,
				MinTemperatureTenths: -9999,
				PrecipitationTenths:  -9999,
			},
			stationID: "TEST001",
			wantErr:   false,
			checkValues: func(t *testing.T, obs *WeatherObservation) {
				if obs.MaxTemperatureCelsius != nil {
					t.Error("MaxTemperatureCelsius should be nil")
				}
				if obs.MinTemperatureCelsius != nil {
					t.Error("MinTemperatureCelsius should be nil")
				}
				if obs.PrecipitationCm != nil {
					t.Error("PrecipitationCm should be nil")
				}
			},
		},
		{
			name: "negative temperatures (valid)",
			record: RawWeatherRecord{
				Date:                "20230115",
				MaxTemperatureTenths: -50,
				MinTemperatureTenths: -100,
				PrecipitationTenths:  0,
			},
			stationID: "TEST001",
			wantErr:   false,
			checkValues: func(t *testing.T, obs *WeatherObservation) {
				if obs.MaxTemperatureCelsius == nil {
					t.Error("MaxTemperatureCelsius should not be nil")
				} else if *obs.MaxTemperatureCelsius != -5.0 {
					t.Errorf("MaxTemperatureCelsius = %v, want %v", *obs.MaxTemperatureCelsius, -5.0)
				}

				if obs.MinTemperatureCelsius == nil {
					t.Error("MinTemperatureCelsius should not be nil")
				} else if *obs.MinTemperatureCelsius != -10.0 {
					t.Errorf("MinTemperatureCelsius = %v, want %v", *obs.MinTemperatureCelsius, -10.0)
				}

				if obs.PrecipitationCm == nil {
					t.Error("PrecipitationCm should not be nil")
				} else if *obs.PrecipitationCm != 0.0 {
					t.Errorf("PrecipitationCm = %v, want %v", *obs.PrecipitationCm, 0.0)
				}
			},
		},
		{
			name: "invalid date format",
			record: RawWeatherRecord{
				Date:                "2023-01-15",
				MaxTemperatureTenths: 250,
				MinTemperatureTenths: 150,
				PrecipitationTenths:  100,
			},
			stationID: "TEST001",
			wantErr:   true,
		},
		{
			name: "precision test - decimal conversion",
			record: RawWeatherRecord{
				Date:                "20230115",
				MaxTemperatureTenths: 255,
				MinTemperatureTenths: 144,
				PrecipitationTenths:  123,
			},
			stationID: "TEST001",
			wantErr:   false,
			checkValues: func(t *testing.T, obs *WeatherObservation) {
				if obs.MaxTemperatureCelsius == nil {
					t.Error("MaxTemperatureCelsius should not be nil")
				} else if *obs.MaxTemperatureCelsius != 25.5 {
					t.Errorf("MaxTemperatureCelsius = %v, want %v", *obs.MaxTemperatureCelsius, 25.5)
				}

				if obs.MinTemperatureCelsius == nil {
					t.Error("MinTemperatureCelsius should not be nil")
				} else if *obs.MinTemperatureCelsius != 14.4 {
					t.Errorf("MinTemperatureCelsius = %v, want %v", *obs.MinTemperatureCelsius, 14.4)
				}

				if obs.PrecipitationCm == nil {
					t.Error("PrecipitationCm should not be nil")
				} else if *obs.PrecipitationCm != 1.23 {
					t.Errorf("PrecipitationCm = %v, want %v", *obs.PrecipitationCm, 1.23)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obs, err := tt.record.ToObservation(tt.stationID)

			if (err != nil) != tt.wantErr {
				t.Errorf("ToObservation() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.checkValues != nil {
				tt.checkValues(t, obs)
			}
		})
	}
}

// TestValidationError tests error handling
func TestValidationError(t *testing.T) {
	err := &ValidationError{
		Field:   "date",
		Value:   "invalid",
		Message: "invalid date format",
	}

	if err.Error() != "invalid date format" {
		t.Errorf("Error() = %v, want %v", err.Error(), "invalid date format")
	}

	if err.IsTransient() {
		t.Error("ValidationError should not be transient")
	}
}
