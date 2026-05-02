package api

import (
	"encoding/json"
	"net/http"

	"github.com/modelfleet/modelfleet/internal/models"
	"github.com/modelfleet/modelfleet/internal/repository"
)

type MachineHandler struct {
	repo *repository.MachineRepository
}

func NewMachineHandler(repo *repository.MachineRepository) *MachineHandler {
	return &MachineHandler{repo: repo}
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

	JSONResponse(w, machine)
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

	JSONResponse(w, result)
}

func (h *MachineHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		JSONError(w, "Machine ID required", http.StatusBadRequest)
		return
	}

	if err := h.repo.Delete(id); err != nil {
		JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
