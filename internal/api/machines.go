package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/modelfleet/modelfleet/internal/deployment"
	"github.com/modelfleet/modelfleet/internal/models"
	"github.com/modelfleet/modelfleet/internal/repository"
)

type MachineHandler struct {
	repo        *repository.MachineRepository
	factsRepo   *repository.MachineFactsRepository
	modelRepo   *repository.ModelRepository
	recommender *deployment.RecommendationEngine
}

func NewMachineHandler(repo *repository.MachineRepository, factsRepo *repository.MachineFactsRepository, modelRepo *repository.ModelRepository) *MachineHandler {
	return &MachineHandler{
		repo:        repo,
		factsRepo:   factsRepo,
		modelRepo:   modelRepo,
		recommender: deployment.NewRecommendationEngine(),
	}
}

func (h *MachineHandler) List(w http.ResponseWriter, r *http.Request) {
	machines, err := h.repo.GetAll()
	if err != nil {
		JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if machines == nil {
		machines = []models.Machine{}
	}

	JSONResponse(w, machines)
}

func (h *MachineHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.AddMachineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.Host == "" {
		JSONError(w, "name and host are required", http.StatusBadRequest)
		return
	}

	// Set default connection type
	if req.ConnectionType == "" {
		req.ConnectionType = "ssh"
	}

	machine := &models.Machine{
		ID:                 generateID(),
		Name:               req.Name,
		Host:               req.Host,
		SSHPort:            req.SSHPort,
		SSHUser:            req.SSHUser,
		AuthType:           req.AuthType,
		AuthPrivateKeyPath: req.AuthPrivateKeyPath,
		Tags:               req.Tags,
		Notes:              req.Notes,
		BaseDir:            req.BaseDir,
		ConnectionType:     req.ConnectionType,
		DockerHost:         req.DockerHost,
		DockerImage:        req.DockerImage,
		DockerNetwork:      req.DockerNetwork,
		DockerRuntime:      req.DockerRuntime,
	}

	// Validate based on connection type
	switch machine.ConnectionType {
	case "ssh":
		if machine.SSHUser == "" {
			JSONError(w, "ssh_user is required for SSH machines", http.StatusBadRequest)
			return
		}
		if machine.SSHPort == 0 {
			machine.SSHPort = 22
		}
		if machine.AuthType == "" {
			machine.AuthType = "key"
		}
	case "docker":
		if machine.DockerImage == "" {
			machine.DockerImage = "ghcr.io/ggml-org/llama.cpp:server"
		}
		// Docker machines don't need SSH credentials
	default:
		JSONError(w, "connection_type must be 'ssh' or 'docker'", http.StatusBadRequest)
		return
	}

	if machine.BaseDir == "" {
		machine.BaseDir = "~/modelfleet"
	}

	if err := h.repo.Create(machine); err != nil {
		JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	JSONResponse(w, machine)
}

func (h *MachineHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		JSONError(w, "Machine ID required", http.StatusBadRequest)
		return
	}

	machine, err := h.repo.GetByID(id)
	if err != nil {
		JSONError(w, "Machine not found", http.StatusNotFound)
		return
	}

	// Get machine facts if available
	facts, _ := h.factsRepo.GetByMachineID(id)

	// Get deployments on this machine
	deployments, _ := h.getMachineDeployments(id)

	JSONResponse(w, map[string]interface{}{
		"machine":     machine,
		"facts":       facts,
		"deployments": deployments,
	})
}

func (h *MachineHandler) getMachineDeployments(machineID string) ([]models.Deployment, error) {
	// This would ideally be in deployment repo, but for now filter all
	// In a real implementation, add GetByMachineID to deployment repo
	return []models.Deployment{}, nil
}

func (h *MachineHandler) TestSSH(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		JSONError(w, "Machine ID required", http.StatusBadRequest)
		return
	}

	machine, err := h.repo.GetByID(id)
	if err != nil {
		JSONError(w, "Machine not found", http.StatusNotFound)
		return
	}

	// Handle Docker machines
	if machine.ConnectionType == "docker" {
		result, err := testDockerConnection(machine)
		if err != nil {
			JSONResponse(w, map[string]interface{}{
				"success": false,
				"error":   err.Error(),
			})
			return
		}
		JSONResponse(w, result)
		return
	}

	result, err := testSSHConnection(machine)
	if err != nil {
		JSONResponse(w, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	JSONResponse(w, result)
}

func (h *MachineHandler) Detect(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		JSONError(w, "Machine ID required", http.StatusBadRequest)
		return
	}

	machine, err := h.repo.GetByID(id)
	if err != nil {
		JSONError(w, "Machine not found", http.StatusNotFound)
		return
	}

	// For Docker machines, detection is limited
	if machine.ConnectionType == "docker" {
		facts := &models.MachineFacts{
			MachineID:       id,
			CapabilityState: "docker_ready",
		}
		h.factsRepo.Save(facts)
		JSONResponse(w, map[string]interface{}{
			"connection_type":  "docker",
			"docker_host":      machine.DockerHost,
			"docker_image":     machine.DockerImage,
			"capability_state": "docker_ready",
			"message":          "Docker machine - hardware detection not applicable",
		})
		return
	}

	result, err := detectMachine(machine)
	if err != nil {
		JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Persist detection results
	facts := &models.MachineFacts{
		MachineID:        id,
		OSName:           result.OSName,
		OSVersion:        result.OSVersion,
		Arch:             result.Arch,
		CPUModel:         result.CPUModel,
		CPUCores:         result.CPUCores,
		RAMBytes:         result.RAMBytes,
		DiskFreeBytes:    result.DiskFreeBytes,
		GPUVendor:        result.GPUVendor,
		GPUName:          result.GPUName,
		GPUVRAMBytes:     result.GPUVRAMBytes,
		GPUDriverVersion: result.GPUDriverVersion,
		CapabilityState:  result.CapabilityState,
		HasNVIDIA:        result.HasNVIDIA,
		HasAMD:           result.HasAMD,
		HasVulkan:        result.HasVulkan,
	}

	if err := h.factsRepo.Save(facts); err != nil {
		JSONError(w, fmt.Sprintf("Detection completed but failed to save: %v", err), http.StatusInternalServerError)
		return
	}

	JSONResponse(w, result)
}

func (h *MachineHandler) GetDeployableModels(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		JSONError(w, "Machine ID required", http.StatusBadRequest)
		return
	}

	// Get machine facts
	facts, err := h.factsRepo.GetByMachineID(id)
	if err != nil {
		JSONError(w, "Machine facts not found. Please run detection first.", http.StatusBadRequest)
		return
	}

	// Get all models
	models, err := h.modelRepo.GetAll()
	if err != nil {
		JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get recommendations
	recommendations := h.recommender.GetDeployableModels(facts, models)

	JSONResponse(w, map[string]interface{}{
		"machine_id":      id,
		"facts":           facts,
		"recommendations": recommendations,
	})
}

func (h *MachineHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		JSONError(w, "Machine ID required", http.StatusBadRequest)
		return
	}

	// Delete associated facts
	h.factsRepo.Delete(id)

	if err := h.repo.Delete(id); err != nil {
		JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
