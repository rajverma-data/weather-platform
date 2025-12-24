package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"weather-platform/internal/models"
	"weather-platform/pkg/logging"
)

// DemoDataProcessing demonstrates the data processing without database
func main() {
	fmt.Println("════════════════════════════════════════════════════════════════")
	fmt.Println("WEATHER PLATFORM - DATA PROCESSING DEMONSTRATION")
	fmt.Println("════════════════════════════════════════════════════════════════")
	fmt.Println()

	// Initialize logger
	logger := logging.NewStructuredLogger("demo", "1.0.0", logging.InfoLevel)
	ctx := context.Background()

	// Process each weather file
	dataDir := "./wx_data"
	files, err := filepath.Glob(filepath.Join(dataDir, "*.txt"))
	if err != nil {
		fmt.Printf("Error reading directory: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Found %d weather station files\n\n", len(files))

	totalRecords := 0
	validRecords := 0
	missingDataCount := 0

	for _, filePath := range files {
		fileName := filepath.Base(filePath)
		stationID := strings.TrimSuffix(fileName, filepath.Ext(fileName))

		fmt.Printf("─────────────────────────────────────────────────────────────\n")
		fmt.Printf("Processing Station: %s\n", stationID)
		fmt.Printf("─────────────────────────────────────────────────────────────\n")

		file, err := os.Open(filePath)
		if err != nil {
			logger.Error(ctx, "Failed to open file", logging.Fields{
				"file": filePath,
			}, err)
			continue
		}

		// Read and parse file
		var lines []string
		content, _ := os.ReadFile(filePath)
		lines = strings.Split(string(content), "\n")

		fileRecords := 0
		fileValid := 0
		fileMissing := 0

		for i, line := range lines {
			if line == "" {
				continue
			}

			totalRecords++
			fileRecords++

			// Parse line
			parts := strings.Split(line, "\t")
			if len(parts) != 4 {
				fmt.Printf("  [%d] Invalid format: %s\n", i+1, line)
				continue
			}

			// Create raw record
			record := &models.RawWeatherRecord{
				Date:                 strings.TrimSpace(parts[0]),
				MaxTemperatureTenths: parseInt(parts[1]),
				MinTemperatureTenths: parseInt(parts[2]),
				PrecipitationTenths:  parseInt(parts[3]),
			}

			// Convert to observation
			obs, err := record.ToObservation(stationID)
			if err != nil {
				fmt.Printf("  [%d] Conversion error: %v\n", i+1, err)
				continue
			}

			fileValid++
			validRecords++

			// Check for missing data
			hasMissing := false
			if obs.MaxTemperatureCelsius == nil {
				hasMissing = true
				fileMissing++
				missingDataCount++
			}
			if obs.MinTemperatureCelsius == nil {
				hasMissing = true
				fileMissing++
				missingDataCount++
			}
			if obs.PrecipitationCm == nil {
				hasMissing = true
				fileMissing++
				missingDataCount++
			}

			// Print first 3 records and any with missing data
			if i < 3 || hasMissing {
				fmt.Printf("  [%d] Date: %s", i+1, obs.ObservationDate.Format("2006-01-02"))

				if obs.MaxTemperatureCelsius != nil {
					fmt.Printf(" | Max: %.1f°C", *obs.MaxTemperatureCelsius)
				} else {
					fmt.Printf(" | Max: NULL")
				}

				if obs.MinTemperatureCelsius != nil {
					fmt.Printf(" | Min: %.1f°C", *obs.MinTemperatureCelsius)
				} else {
					fmt.Printf(" | Min: NULL")
				}

				if obs.PrecipitationCm != nil {
					fmt.Printf(" | Precip: %.2f cm", *obs.PrecipitationCm)
				} else {
					fmt.Printf(" | Precip: NULL")
				}

				if hasMissing {
					fmt.Printf(" ⚠ MISSING DATA")
				}
				fmt.Println()
			}
		}

		fmt.Printf("\n  Station Summary:\n")
		fmt.Printf("    Total records: %d\n", fileRecords)
		fmt.Printf("    Valid conversions: %d\n", fileValid)
		fmt.Printf("    Missing values: %d\n", fileMissing)
		fmt.Println()

		file.Close()
	}

	fmt.Println("════════════════════════════════════════════════════════════════")
	fmt.Println("PROCESSING SUMMARY")
	fmt.Println("════════════════════════════════════════════════════════════════")
	fmt.Printf("Total weather files:    %d\n", len(files))
	fmt.Printf("Total records:          %d\n", totalRecords)
	fmt.Printf("Valid conversions:      %d\n", validRecords)
	fmt.Printf("Missing data points:    %d\n", missingDataCount)
	fmt.Printf("Success rate:           %.2f%%\n", float64(validRecords)/float64(totalRecords)*100)
	fmt.Println()

	// Demonstrate statistics calculation
	fmt.Println("════════════════════════════════════════════════════════════════")
	fmt.Println("STATISTICS CALCULATION DEMONSTRATION")
	fmt.Println("════════════════════════════════════════════════════════════════")

	// Calculate stats for first station
	if len(files) > 0 {
		filePath := files[0]
		fileName := filepath.Base(filePath)
		stationID := strings.TrimSuffix(fileName, filepath.Ext(fileName))

		content, _ := os.ReadFile(filePath)
		lines := strings.Split(string(content), "\n")

		var maxTemps []float64
		var minTemps []float64
		var precips []float64

		for _, line := range lines {
			if line == "" {
				continue
			}

			parts := strings.Split(line, "\t")
			if len(parts) != 4 {
				continue
			}

			record := &models.RawWeatherRecord{
				Date:                 strings.TrimSpace(parts[0]),
				MaxTemperatureTenths: parseInt(parts[1]),
				MinTemperatureTenths: parseInt(parts[2]),
				PrecipitationTenths:  parseInt(parts[3]),
			}

			obs, _ := record.ToObservation(stationID)
			if obs.MaxTemperatureCelsius != nil {
				maxTemps = append(maxTemps, *obs.MaxTemperatureCelsius)
			}
			if obs.MinTemperatureCelsius != nil {
				minTemps = append(minTemps, *obs.MinTemperatureCelsius)
			}
			if obs.PrecipitationCm != nil {
				precips = append(precips, *obs.PrecipitationCm)
			}
		}

		fmt.Printf("Station: %s\n", stationID)
		fmt.Printf("─────────────────────────────────────────────────────────────\n")

		if len(maxTemps) > 0 {
			avgMax := average(maxTemps)
			fmt.Printf("Average Max Temperature:  %.2f°C (from %d readings)\n", avgMax, len(maxTemps))
		}

		if len(minTemps) > 0 {
			avgMin := average(minTemps)
			fmt.Printf("Average Min Temperature:  %.2f°C (from %d readings)\n", avgMin, len(minTemps))
		}

		if len(precips) > 0 {
			totalPrecip := sum(precips)
			fmt.Printf("Total Precipitation:      %.2f cm (from %d readings)\n", totalPrecip, len(precips))
		}
	}

	fmt.Println()
	fmt.Println("════════════════════════════════════════════════════════════════")
	fmt.Println("✅ DATA PROCESSING DEMONSTRATION COMPLETE")
	fmt.Println("════════════════════════════════════════════════════════════════")
	fmt.Println()
	fmt.Println("The system successfully:")
	fmt.Println("  ✓ Parsed tab-delimited weather data")
	fmt.Println("  ✓ Converted units (0.1°C → °C, 0.1mm → cm)")
	fmt.Println("  ✓ Handled missing values (-9999 → NULL)")
	fmt.Println("  ✓ Calculated statistics (averages, totals)")
	fmt.Println("  ✓ Validated data format and types")
	fmt.Println()
	fmt.Println("With a database, this would:")
	fmt.Println("  • Store all observations in weather_observations table")
	fmt.Println("  • Calculate and cache statistics in weather_statistics table")
	fmt.Println("  • Serve data via REST API endpoints")
	fmt.Println("  • Provide real-time metrics and monitoring")
	fmt.Println()
}

func parseInt(s string) int {
	var val int
	fmt.Sscanf(strings.TrimSpace(s), "%d", &val)
	return val
}

func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	return sum(values) / float64(len(values))
}

func sum(values []float64) float64 {
	total := 0.0
	for _, v := range values {
		total += v
	}
	return total
}
