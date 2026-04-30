package api

import (
	"encoding/json"
	"net/http"

	"github.com/modelfleet/modelfleet/internal/models"
	"github.com/modelfleet/modelfleet/internal/repository"
)

type ModelHandler struct {
	repo *repository.ModelRepository
}

func NewModelHandler(repo *repository.ModelRepository) *ModelHandler {
	return &ModelHandler{repo: repo}
}

func (h *ModelHandler) List(w http.ResponseWriter, r *http.Request) {
	modelList, err := h.repo.GetAll()
	if err != nil {
		JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if modelList == nil {
		modelList = []models.Model{}
	}

	JSONResponse(w, modelList)
}

func (h *ModelHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.CreateModelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.DisplayName == "" || req.Filename == "" {
		JSONError(w, "display_name and filename are required", http.StatusBadRequest)
		return
	}

	if req.Format == "" {
		req.Format = "GGUF"
	}

	model := &models.Model{
		ID:                 generateID(),
		DisplayName:        req.DisplayName,
		Family:             req.Family,
		ParameterSize:      req.ParameterSize,
		Format:             req.Format,
		Quantization:       req.Quantization,
		ContextLength:      req.ContextLength,
		SourceURL:          req.SourceURL,
		Filename:           req.Filename,
		SHA256:             req.SHA256,
		EstimatedRAMBytes:  req.EstimatedRAMBytes,
		EstimatedVRAMBytes: req.EstimatedVRAMBytes,
		Tags:               req.Tags,
	}

	if err := h.repo.Create(model); err != nil {
		JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	JSONResponse(w, model)
}

func (h *ModelHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		JSONError(w, "Model ID required", http.StatusBadRequest)
		return
	}

	model, err := h.repo.GetByID(id)
	if err != nil {
		JSONError(w, "Model not found", http.StatusNotFound)
		return
	}

	JSONResponse(w, model)
}

func (h *ModelHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		JSONError(w, "Model ID required", http.StatusBadRequest)
		return
	}

	if err := h.repo.Delete(id); err != nil {
		JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
