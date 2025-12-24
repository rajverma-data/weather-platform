package handlers

import (
	"encoding/json"
	"net/http"
)

// OpenAPISpec returns the OpenAPI 3.0 specification for the Weather Platform API
func OpenAPISpec(w http.ResponseWriter, r *http.Request) {
	spec := map[string]interface{}{
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title":       "Weather Platform API",
			"description": "Production-grade weather data engineering platform with PostgreSQL, REST API, and batch processing",
			"version":     "1.0.0",
			"contact": map[string]string{
				"name": "Weather Platform Team",
			},
		},
		"servers": []map[string]string{
			{"url": "http://localhost:8080", "description": "Local development server"},
		},
		"paths": map[string]interface{}{
			"/api/weather": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "Get weather observations",
					"description": "Retrieve weather observations with filtering and pagination",
					"parameters": []map[string]interface{}{
						{
							"name":        "station_id",
							"in":          "query",
							"description": "Filter by weather station ID",
							"required":    false,
							"schema":      map[string]string{"type": "string"},
						},
						{
							"name":        "start_date",
							"in":          "query",
							"description": "Filter by start date (YYYY-MM-DD)",
							"required":    false,
							"schema":      map[string]string{"type": "string", "format": "date"},
						},
						{
							"name":        "end_date",
							"in":          "query",
							"description": "Filter by end date (YYYY-MM-DD)",
							"required":    false,
							"schema":      map[string]string{"type": "string", "format": "date"},
						},
						{
							"name":        "page",
							"in":          "query",
							"description": "Page number (default: 1)",
							"required":    false,
							"schema":      map[string]interface{}{"type": "integer", "default": 1},
						},
						{
							"name":        "limit",
							"in":          "query",
							"description": "Records per page (default: 100)",
							"required":    false,
							"schema":      map[string]interface{}{"type": "integer", "default": 100},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Successful response",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"data": map[string]interface{}{
												"type": "array",
												"items": map[string]interface{}{
													"type": "object",
													"properties": map[string]interface{}{
														"id":                        map[string]string{"type": "integer"},
														"station_id":                map[string]string{"type": "string"},
														"observation_date":          map[string]string{"type": "string", "format": "date-time"},
														"max_temperature_celsius":   map[string]interface{}{"type": "number", "nullable": true},
														"min_temperature_celsius":   map[string]interface{}{"type": "number", "nullable": true},
														"precipitation_cm":          map[string]interface{}{"type": "number", "nullable": true},
														"created_at":                map[string]string{"type": "string", "format": "date-time"},
													},
												},
											},
											"total":       map[string]string{"type": "integer"},
											"page":        map[string]string{"type": "integer"},
											"limit":       map[string]string{"type": "integer"},
											"total_pages": map[string]string{"type": "integer"},
										},
									},
								},
							},
						},
					},
				},
			},
			"/api/weather/stats": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "Get weather statistics",
					"description": "Retrieve calculated yearly statistics per station",
					"parameters": []map[string]interface{}{
						{
							"name":        "station_id",
							"in":          "query",
							"description": "Filter by weather station ID",
							"required":    false,
							"schema":      map[string]string{"type": "string"},
						},
						{
							"name":        "year",
							"in":          "query",
							"description": "Filter by year",
							"required":    false,
							"schema":      map[string]string{"type": "integer"},
						},
						{
							"name":        "page",
							"in":          "query",
							"description": "Page number (default: 1)",
							"required":    false,
							"schema":      map[string]interface{}{"type": "integer", "default": 1},
						},
						{
							"name":        "limit",
							"in":          "query",
							"description": "Records per page (default: 100)",
							"required":    false,
							"schema":      map[string]interface{}{"type": "integer", "default": 100},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Successful response",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"data": map[string]interface{}{
												"type": "array",
												"items": map[string]interface{}{
													"type": "object",
													"properties": map[string]interface{}{
														"id":                           map[string]string{"type": "integer"},
														"station_id":                   map[string]string{"type": "string"},
														"year":                         map[string]string{"type": "integer"},
														"avg_max_temperature_celsius":  map[string]interface{}{"type": "number", "nullable": true},
														"avg_min_temperature_celsius":  map[string]interface{}{"type": "number", "nullable": true},
														"total_precipitation_cm":       map[string]interface{}{"type": "number", "nullable": true},
														"observation_count":            map[string]string{"type": "integer"},
														"valid_max_temp_count":         map[string]string{"type": "integer"},
														"valid_min_temp_count":         map[string]string{"type": "integer"},
														"valid_precipitation_count":    map[string]string{"type": "integer"},
													},
												},
											},
											"total":       map[string]string{"type": "integer"},
											"page":        map[string]string{"type": "integer"},
											"limit":       map[string]string{"type": "integer"},
											"total_pages": map[string]string{"type": "integer"},
										},
									},
								},
							},
						},
					},
				},
			},
			"/health": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "Health check",
					"description": "Check if the API is running",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "API is healthy",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"status": map[string]string{"type": "string"},
										},
									},
								},
							},
						},
					},
				},
			},
			"/metrics": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "Prometheus metrics",
					"description": "Prometheus metrics endpoint for monitoring",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Prometheus metrics in text format",
							"content": map[string]interface{}{
								"text/plain": map[string]interface{}{
									"schema": map[string]string{"type": "string"},
								},
							},
						},
					},
				},
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(spec)
}
