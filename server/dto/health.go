package dto

type HealthCheckResponse struct {
	Success bool   `json:"success"`
	Version string `json:"version"`
}
