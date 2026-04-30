package models

import "time"

type Deployment struct {
	ID                string     `json:"id"`
	Name              string     `json:"name"`
	MachineID         string     `json:"machine_id"`
	ModelID           string     `json:"model_id"`
	Runtime           string     `json:"runtime"`
	Backend           string     `json:"backend"`
	Status            string     `json:"status"`
	Host              string     `json:"host"`
	Port              int        `json:"port"`
	EndpointURL       string     `json:"endpoint_url"`
	APIFormat         string     `json:"api_format"`
	ContextSize       int        `json:"context_size,omitempty"`
	GPULayers         int        `json:"gpu_layers,omitempty"`
	Threads           int        `json:"threads,omitempty"`
	ProcessMode       string     `json:"process_mode,omitempty"`
	PID               int        `json:"pid,omitempty"`
	ServiceName       string     `json:"service_name,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	LastHealthCheckAt *time.Time `json:"last_health_check_at,omitempty"`
	LastError         string     `json:"last_error,omitempty"`
}

type CreateDeploymentRequest struct {
	Name        string `json:"name"`
	MachineID   string `json:"machine_id"`
	ModelID     string `json:"model_id"`
	Runtime     string `json:"runtime,omitempty"`
	Backend     string `json:"backend,omitempty"`
	Host        string `json:"host,omitempty"`
	Port        int    `json:"port,omitempty"`
	ContextSize int    `json:"context_size,omitempty"`
	GPULayers   int    `json:"gpu_layers,omitempty"`
	Threads     int    `json:"threads,omitempty"`
}
