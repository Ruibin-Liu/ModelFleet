package models

import "time"

type Event struct {
	ID           string    `json:"id"`
	Type         string    `json:"type"`
	Severity     string    `json:"severity"`
	MachineID    string    `json:"machine_id,omitempty"`
	DeploymentID string    `json:"deployment_id,omitempty"`
	Message      string    `json:"message"`
	DataJSON     string    `json:"data_json,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type APIKey struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	KeyHash    string     `json:"-"`
	Enabled    bool       `json:"enabled"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
}

type Alias struct {
	ID                 string    `json:"id"`
	Alias              string    `json:"alias"`
	TargetType         string    `json:"target_type"`
	TargetDeploymentID string    `json:"target_deployment_id,omitempty"`
	TargetModelID      string    `json:"target_model_id,omitempty"`
	RoutingPolicy      string    `json:"routing_policy"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}
