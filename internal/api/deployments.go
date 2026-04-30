package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/modelfleet/modelfleet/internal/models"
	"github.com/modelfleet/modelfleet/internal/repository"
)

type DeploymentHandler struct {
	repo        *repository.DeploymentRepository
	machineRepo *repository.MachineRepository
	modelRepo   *repository.ModelRepository
}

func NewDeploymentHandler(repo *repository.DeploymentRepository, machineRepo *repository.MachineRepository, modelRepo *repository.ModelRepository) *DeploymentHandler {
	return &DeploymentHandler{
		repo:        repo,
		machineRepo: machineRepo,
		modelRepo:   modelRepo,
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
