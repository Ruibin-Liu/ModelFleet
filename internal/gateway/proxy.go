package gateway

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/modelfleet/modelfleet/internal/events"
	"github.com/modelfleet/modelfleet/internal/models"
	"github.com/modelfleet/modelfleet/internal/repository"
)

type Proxy struct {
	deploymentRepo *repository.DeploymentRepository
	eventLogger    *events.Logger
	httpClient     *http.Client
}

func NewProxy(deploymentRepo *repository.DeploymentRepository, eventLogger *events.Logger) *Proxy {
	return &Proxy{
		deploymentRepo: deploymentRepo,
		eventLogger:    eventLogger,
		httpClient: &http.Client{
			Timeout: 120 * time.Second, // Long timeout for LLM inference
		},
	}
}

func (p *Proxy) ListModels(w http.ResponseWriter, r *http.Request) {
	deployments, err := p.deploymentRepo.GetAll()
	if err != nil {
		http.Error(w, `{"error":{"message":"Internal server error","type":"api_error"}}`, http.StatusInternalServerError)
		return
	}

	var modelList []map[string]interface{}
	for _, d := range deployments {
		// Only show running or planned deployments
		if d.Status == "running" || d.Status == "planned" {
			modelList = append(modelList, map[string]interface{}{
				"id":       fmt.Sprintf("local/%s", d.Name),
				"object":   "model",
				"owned_by": "modelfleet",
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"object": "list",
		"data":   modelList,
	})
}

func (p *Proxy) ChatCompletions(w http.ResponseWriter, r *http.Request) {
	// Read the request body
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		p.writeError(w, http.StatusBadRequest, "Failed to read request body")
		return
	}
	r.Body.Close()

	// Parse to get the model
	var reqBody struct {
		Model string `json:"model"`
	}
	if err := json.Unmarshal(bodyBytes, &reqBody); err != nil {
		p.writeError(w, http.StatusBadRequest, "Invalid JSON request body")
		return
	}

	if reqBody.Model == "" {
		p.writeError(w, http.StatusBadRequest, "Model field is required")
		return
	}

	// Find the deployment for this model
	// Model ID format: "local/deployment-name"
	modelName := strings.TrimPrefix(reqBody.Model, "local/")

	// Find deployment by name
	deployments, err := p.deploymentRepo.GetAll()
	if err != nil {
		p.eventLogger.LogGateway("", reqBody.Model, false, err)
		p.writeError(w, http.StatusInternalServerError, "Failed to lookup deployments")
		return
	}

	var targetDeployment *models.Deployment
	for _, d := range deployments {
		if d.Name == modelName {
			targetDeployment = &d
			break
		}
	}

	if targetDeployment == nil {
		p.eventLogger.LogGateway("", reqBody.Model, false, fmt.Errorf("no deployment found"))
		p.writeError(w, http.StatusServiceUnavailable,
			fmt.Sprintf("No healthy deployment available for model %s", reqBody.Model))
		return
	}

	if targetDeployment.Status != "running" {
		p.eventLogger.LogGateway(targetDeployment.ID, reqBody.Model, false, fmt.Errorf("deployment not running"))
		p.writeError(w, http.StatusServiceUnavailable,
			fmt.Sprintf("No healthy deployment available for model %s", reqBody.Model))
		return
	}

	// Proxy the request to the remote llama-server
	targetURL := fmt.Sprintf("http://%s:%d/v1/chat/completions", targetDeployment.Host, targetDeployment.Port)

	proxyReq, err := http.NewRequest("POST", targetURL, bytes.NewReader(bodyBytes))
	if err != nil {
		p.eventLogger.LogGateway(targetDeployment.ID, reqBody.Model, false, err)
		p.writeError(w, http.StatusInternalServerError, "Failed to create proxy request")
		return
	}

	// Copy headers
	proxyReq.Header.Set("Content-Type", "application/json")
	if auth := r.Header.Get("Authorization"); auth != "" {
		proxyReq.Header.Set("Authorization", auth)
	}

	// Forward the request
	resp, err := p.httpClient.Do(proxyReq)
	if err != nil {
		p.eventLogger.LogGateway(targetDeployment.ID, reqBody.Model, false, err)
		p.writeError(w, http.StatusServiceUnavailable,
			fmt.Sprintf("Failed to connect to deployment: %v", err))
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)

	p.eventLogger.LogGateway(targetDeployment.ID, reqBody.Model, true, nil)
}

func (p *Proxy) writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]interface{}{
			"message": message,
			"type":    "modelfleet_no_healthy_deployment",
		},
	})
}
