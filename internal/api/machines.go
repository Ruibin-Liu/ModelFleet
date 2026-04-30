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

	if req.Name == "" || req.Host == "" || req.SSHUser == "" {
		JSONError(w, "name, host, and ssh_user are required", http.StatusBadRequest)
		return
	}

	if req.SSHPort == 0 {
		req.SSHPort = 22
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
		ConnectionType:     "ssh",
	}

	if machine.BaseDir == "" {
		machine.BaseDir = "~/modelfleet"
	}

	if machine.AuthType == "" {
		machine.AuthType = "key"
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
