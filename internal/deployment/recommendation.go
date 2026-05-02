package deployment

import (
	"fmt"

	"github.com/modelfleet/modelfleet/internal/models"
)

type RecommendationEngine struct{}

func NewRecommendationEngine() *RecommendationEngine {
	return &RecommendationEngine{}
}

func (re *RecommendationEngine) GetDeployableModels(facts *models.MachineFacts, modelsList []models.Model) []models.DeployableModel {
	var deployable []models.DeployableModel

	for _, model := range modelsList {
		compat := re.checkCompatibility(facts, model)
		if compat.Compatibility != "incompatible" {
			deployable = append(deployable, compat)
		}
	}

	return deployable
}

func (re *RecommendationEngine) checkCompatibility(facts *models.MachineFacts, model models.Model) models.DeployableModel {
	result := models.DeployableModel{
		Model:              model,
		Compatibility:      "compatible",
		RecommendedBackend: "cpu",
	}

	// Determine recommended backend based on hardware
	switch facts.CapabilityState {
	case "nvidia_ready":
		result.RecommendedBackend = "cuda"
	case "amd_rocm_ready":
		result.RecommendedBackend = "rocm"
	case "vulkan_ready":
		result.RecommendedBackend = "vulkan"
	case "mac_metal_ready":
		result.RecommendedBackend = "metal"
	default:
		result.RecommendedBackend = "cpu"
	}

	// Check GPU VRAM requirements
	if facts.GPUVRAMBytes > 0 && model.EstimatedVRAMBytes > 0 {
		if model.EstimatedVRAMBytes > facts.GPUVRAMBytes {
			// Check if it can run on CPU instead
			if model.EstimatedRAMBytes > 0 && model.EstimatedRAMBytes < facts.RAMBytes {
				result.RecommendedBackend = "cpu"
				result.Compatibility = "not_recommended"
				result.Reason = fmt.Sprintf("Model requires %s VRAM but GPU has %s. Will run on CPU.",
					formatBytes(model.EstimatedVRAMBytes), formatBytes(facts.GPUVRAMBytes))
			} else {
				result.Compatibility = "incompatible"
				result.Reason = fmt.Sprintf("Model requires %s VRAM but GPU has %s. CPU RAM also insufficient.",
					formatBytes(model.EstimatedVRAMBytes), formatBytes(facts.GPUVRAMBytes))
				return result
			}
		} else {
			result.Compatibility = "optimal"
			result.Reason = fmt.Sprintf("Model fits in GPU VRAM (%s / %s)",
				formatBytes(model.EstimatedVRAMBytes), formatBytes(facts.GPUVRAMBytes))
		}
	} else {
		// CPU-only check
		if model.EstimatedRAMBytes > 0 {
			if model.EstimatedRAMBytes > facts.RAMBytes {
				result.Compatibility = "incompatible"
				result.Reason = fmt.Sprintf("Model requires %s RAM but machine has %s",
					formatBytes(model.EstimatedRAMBytes), formatBytes(facts.RAMBytes))
				return result
			}

			// Check if there's enough headroom
			ramUsage := float64(model.EstimatedRAMBytes) / float64(facts.RAMBytes)
			if ramUsage > 0.8 {
				result.Compatibility = "not_recommended"
				result.Reason = fmt.Sprintf("Model uses %.0f%% of available RAM. May cause system instability.", ramUsage*100)
			} else {
				result.Compatibility = "compatible"
				result.Reason = fmt.Sprintf("Model uses %.0f%% of available RAM (%s / %s)",
					ramUsage*100, formatBytes(model.EstimatedRAMBytes), formatBytes(facts.RAMBytes))
			}
		}
	}

	// Calculate estimated load
	if facts.GPUVRAMBytes > 0 && model.EstimatedVRAMBytes > 0 {
		usage := float64(model.EstimatedVRAMBytes) / float64(facts.GPUVRAMBytes)
		result.EstimatedLoad = fmt.Sprintf("%.0f%% GPU VRAM", usage*100)
	} else if model.EstimatedRAMBytes > 0 {
		usage := float64(model.EstimatedRAMBytes) / float64(facts.RAMBytes)
		result.EstimatedLoad = fmt.Sprintf("%.0f%% RAM", usage*100)
	}

	return result
}

func formatBytes(bytes int64) string {
	if bytes >= 1024*1024*1024 {
		return fmt.Sprintf("%.1f GB", float64(bytes)/(1024*1024*1024))
	} else if bytes >= 1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
	} else if bytes >= 1024 {
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	}
	return fmt.Sprintf("%d B", bytes)
}
