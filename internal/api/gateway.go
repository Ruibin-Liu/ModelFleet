package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/modelfleet/modelfleet/internal/repository"
)

type GatewayHandler struct {
	deploymentRepo *repository.DeploymentRepository
}

func NewGatewayHandler(deploymentRepo *repository.DeploymentRepository) *GatewayHandler {
	return &GatewayHandler{deploymentRepo: deploymentRepo}
}

func (g *GatewayHandler) ListModels(w http.ResponseWriter, r *http.Request) {
	deployments, err := g.deploymentRepo.GetAll()
	if err != nil {
		JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var models []map[string]interface{}
	for _, d := range deployments {
		if d.Status == "running" || d.Status == "planned" {
			models = append(models, map[string]interface{}{
				"id":       fmt.Sprintf("local/%s", d.Name),
				"object":   "model",
				"owned_by": "modelfleet",
			})
		}
	}

	JSONResponse(w, map[string]interface{}{
		"object": "list",
		"data":   models,
	})
}

func (g *GatewayHandler) ChatCompletions(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Model    string                   `json:"model"`
		Messages []map[string]interface{} `json:"messages"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// For MVP, return a placeholder response
	// TODO: Implement actual routing to remote deployments
	JSONResponse(w, map[string]interface{}{
		"id":      "chatcmpl-" + generateID(),
		"object":  "chat.completion",
		"created": 1234567890,
		"model":   req.Model,
		"choices": []map[string]interface{}{
			{
				"index": 0,
				"message": map[string]interface{}{
					"role":    "assistant",
					"content": "This is a placeholder response. ModelFleet MVP is running but actual model routing requires a deployed and running llama-server instance.",
				},
				"finish_reason": "stop",
			},
		},
	})
}
