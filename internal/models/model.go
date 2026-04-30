package models

import "time"

type Model struct {
	ID                 string    `json:"id"`
	DisplayName        string    `json:"display_name"`
	Family             string    `json:"family,omitempty"`
	ParameterSize      string    `json:"parameter_size,omitempty"`
	Format             string    `json:"format"`
	Quantization       string    `json:"quantization,omitempty"`
	ContextLength      int       `json:"context_length,omitempty"`
	SourceURL          string    `json:"source_url,omitempty"`
	Filename           string    `json:"filename"`
	SHA256             string    `json:"sha256,omitempty"`
	EstimatedRAMBytes  int64     `json:"estimated_ram_bytes,omitempty"`
	EstimatedVRAMBytes int64     `json:"estimated_vram_bytes,omitempty"`
	Tags               string    `json:"tags,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type CreateModelRequest struct {
	DisplayName        string `json:"display_name"`
	Family             string `json:"family,omitempty"`
	ParameterSize      string `json:"parameter_size,omitempty"`
	Format             string `json:"format,omitempty"`
	Quantization       string `json:"quantization,omitempty"`
	ContextLength      int    `json:"context_length,omitempty"`
	SourceURL          string `json:"source_url,omitempty"`
	Filename           string `json:"filename"`
	SHA256             string `json:"sha256,omitempty"`
	EstimatedRAMBytes  int64  `json:"estimated_ram_bytes,omitempty"`
	EstimatedVRAMBytes int64  `json:"estimated_vram_bytes,omitempty"`
	Tags               string `json:"tags,omitempty"`
}
