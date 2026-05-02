package models

import "time"

type MachineFacts struct {
	MachineID        string    `json:"machine_id"`
	OSName           string    `json:"os_name,omitempty"`
	OSVersion        string    `json:"os_version,omitempty"`
	Arch             string    `json:"arch,omitempty"`
	CPUModel         string    `json:"cpu_model,omitempty"`
	CPUCores         int       `json:"cpu_cores,omitempty"`
	RAMBytes         int64     `json:"ram_bytes,omitempty"`
	DiskFreeBytes    int64     `json:"disk_free_bytes,omitempty"`
	GPUVendor        string    `json:"gpu_vendor,omitempty"`
	GPUName          string    `json:"gpu_name,omitempty"`
	GPUVRAMBytes     int64     `json:"gpu_vram_bytes,omitempty"`
	GPUDriverVersion string    `json:"gpu_driver_version,omitempty"`
	CapabilityState  string    `json:"capability_state"`
	HasNVIDIA        bool      `json:"has_nvidia"`
	HasAMD           bool      `json:"has_amd"`
	HasVulkan        bool      `json:"has_vulkan"`
	DetectedAt       time.Time `json:"detected_at"`
}

type DeployableModel struct {
	Model              Model  `json:"model"`
	Compatibility      string `json:"compatibility"` // "optimal", "compatible", "not_recommended"
	Reason             string `json:"reason"`
	RecommendedBackend string `json:"recommended_backend"`
	EstimatedLoad      string `json:"estimated_load"`
}
