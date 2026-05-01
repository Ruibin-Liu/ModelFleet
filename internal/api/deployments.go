package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/modelfleet/modelfleet/internal/deployment"
	"github.com/modelfleet/modelfleet/internal/events"
	"github.com/modelfleet/modelfleet/internal/models"
	"github.com/modelfleet/modelfleet/internal/repository"
)

type DeploymentHandler struct {
	repo        *repository.DeploymentRepository
	machineRepo *repository.MachineRepository
	modelRepo   *repository.ModelRepository
	executor    *deployment.Executor
}

func NewDeploymentHandler(repo *repository.DeploymentRepository, machineRepo *repository.MachineRepository, modelRepo *repository.ModelRepository, eventLogger *events.Logger) *DeploymentHandler {
	return &DeploymentHandler{
		repo:        repo,
		machineRepo: machineRepo,
		modelRepo:   modelRepo,
		executor:    deployment.NewExecutor(eventLogger),
	}
}

func (h *DeploymentHandler) List(w http.ResponseWriter, r *http.Request) {
	deployments, err := h.repo.GetAll()
	if err != nil {
		JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if deployments == nil {
		deployments = []models.Deployment{}
	}

	JSONResponse(w, deployments)
}

func (h *DeploymentHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.CreateDeploymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.MachineID == "" || req.ModelID == "" {
		JSONError(w, "name, machine_id, and model_id are required", http.StatusBadRequest)
		return
	}

	// Verify machine exists
	_, err := h.machineRepo.GetByID(req.MachineID)
	if err != nil {
		JSONError(w, "Machine not found", http.StatusBadRequest)
		return
	}

	// Verify model exists
	_, err = h.modelRepo.GetByID(req.ModelID)
	if err != nil {
		JSONError(w, "Model not found", http.StatusBadRequest)
		return
	}

	deployment := &models.Deployment{
		ID:          generateID(),
		Name:        req.Name,
		MachineID:   req.MachineID,
		ModelID:     req.ModelID,
		Runtime:     req.Runtime,
		Backend:     req.Backend,
		Status:      "planned",
		Host:        req.Host,
		Port:        req.Port,
		EndpointURL: fmt.Sprintf("http://%s:%d", req.Host, req.Port),
		APIFormat:   "openai",
		ContextSize: req.ContextSize,
		GPULayers:   req.GPULayers,
		Threads:     req.Threads,
	}

	if deployment.Host == "" {
		deployment.Host = "0.0.0.0"
	}
	if deployment.Port == 0 {
		deployment.Port = 8081
	}
	if deployment.Runtime == "" {
		deployment.Runtime = "llama.cpp"
	}
	if deployment.Backend == "" {
		deployment.Backend = "cpu"
	}
	deployment.EndpointURL = fmt.Sprintf("http://%s:%d", deployment.Host, deployment.Port)

	if err := h.repo.Create(deployment); err != nil {
		JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	JSONResponse(w, deployment)
}

func (h *DeploymentHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		JSONError(w, "Deployment ID required", http.StatusBadRequest)
		return
	}

	deployment, err := h.repo.GetByID(id)
	if err != nil {
		JSONError(w, "Deployment not found", http.StatusNotFound)
		return
	}

	JSONResponse(w, deployment)
}

func (h *DeploymentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		JSONError(w, "Deployment ID required", http.StatusBadRequest)
		return
	}

	if err := h.repo.Delete(id); err != nil {
		JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *DeploymentHandler) Start(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		JSONError(w, "Deployment ID required", http.StatusBadRequest)
		return
	}

	// Get deployment
	dep, err := h.repo.GetByID(id)
	if err != nil {
		JSONError(w, "Deployment not found", http.StatusNotFound)
		return
	}

	// Get machine
	machine, err := h.machineRepo.GetByID(dep.MachineID)
	if err != nil {
		JSONError(w, "Machine not found", http.StatusBadRequest)
		return
	}

	// Get model
	model, err := h.modelRepo.GetByID(dep.ModelID)
	if err != nil {
		JSONError(w, "Model not found", http.StatusBadRequest)
		return
	}

	// Update status to deploying
	h.repo.UpdateStatus(id, "deploying")

	// Execute start
	result, err := h.executor.Start(dep, machine, model)
	if err != nil {
		h.repo.UpdateStatus(id, "failed")
		JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if !result.Success {
		h.repo.UpdateStatus(id, "failed")
		JSONError(w, result.Error, http.StatusInternalServerError)
		return
	}

	// Update status to running
	h.repo.UpdateStatus(id, "running")

	JSONResponse(w, map[string]interface{}{
		"success": true,
		"message": result.Message,
	})
}

func (h *DeploymentHandler) Stop(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		JSONError(w, "Deployment ID required", http.StatusBadRequest)
		return
	}

	// Get deployment
	dep, err := h.repo.GetByID(id)
	if err != nil {
		JSONError(w, "Deployment not found", http.StatusNotFound)
		return
	}

	// Get machine
	machine, err := h.machineRepo.GetByID(dep.MachineID)
	if err != nil {
		JSONError(w, "Machine not found", http.StatusBadRequest)
		return
	}

	// Execute stop
	result, err := h.executor.Stop(dep, machine)
	if err != nil {
		JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Update status to stopped
	h.repo.UpdateStatus(id, "stopped")

	JSONResponse(w, map[string]interface{}{
		"success": true,
		"message": result.Message,
	})
}

func (h *DeploymentHandler) GetLogs(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		JSONError(w, "Deployment ID required", http.StatusBadRequest)
		return
	}

	// Get deployment
	dep, err := h.repo.GetByID(id)
	if err != nil {
		JSONError(w, "Deployment not found", http.StatusNotFound)
		return
	}

	// Get machine
	machine, err := h.machineRepo.GetByID(dep.MachineID)
	if err != nil {
		JSONError(w, "Machine not found", http.StatusBadRequest)
		return
	}

	// Parse lines parameter
	linesStr := r.URL.Query().Get("lines")
	lines := 100
	if linesStr != "" {
		if l, err := strconv.Atoi(linesStr); err == nil {
			lines = l
		}
	}

	// Get logs
	logs, err := h.executor.GetLogs(dep, machine, lines)
	if err != nil {
		JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	JSONResponse(w, map[string]interface{}{
		"logs": logs,
	})
}
