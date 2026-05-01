package api

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"os"
	"strings"

	"github.com/modelfleet/modelfleet/internal/events"
	"github.com/modelfleet/modelfleet/internal/gateway"
	"github.com/modelfleet/modelfleet/internal/models"
	"github.com/modelfleet/modelfleet/internal/repository"
)

func NewRouter(
	machineRepo *repository.MachineRepository,
	modelRepo *repository.ModelRepository,
	deploymentRepo *repository.DeploymentRepository,
	eventRepo *repository.EventRepository,
	apiKeyRepo *repository.APIKeyRepository,
	eventLogger *events.Logger,
) http.Handler {
	mux := http.NewServeMux()

	// Initialize handlers
	machineHandler := NewMachineHandler(machineRepo)
	modelHandler := NewModelHandler(modelRepo)
	deploymentHandler := NewDeploymentHandler(deploymentRepo, machineRepo, modelRepo, eventLogger)
	gatewayProxy := gateway.NewProxy(deploymentRepo, eventLogger)

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		JSONResponse(w, map[string]string{"status": "ok"})
	})

	// Machine routes
	mux.HandleFunc("/api/machines", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			machineHandler.List(w, r)
		case http.MethodPost:
			machineHandler.Create(w, r)
		default:
			JSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/machines/{id}", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			machineHandler.Get(w, r)
		case http.MethodDelete:
			machineHandler.Delete(w, r)
		default:
			JSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/machines/{id}/test-ssh", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			machineHandler.TestSSH(w, r)
		} else {
			JSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/machines/{id}/detect", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			machineHandler.Detect(w, r)
		} else {
			JSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Model routes
	mux.HandleFunc("/api/models", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			modelHandler.List(w, r)
		case http.MethodPost:
			modelHandler.Create(w, r)
		default:
			JSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/models/{id}", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			modelHandler.Get(w, r)
		case http.MethodDelete:
			modelHandler.Delete(w, r)
		default:
			JSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Deployment routes
	mux.HandleFunc("/api/deployments", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			deploymentHandler.List(w, r)
		case http.MethodPost:
			deploymentHandler.Create(w, r)
		default:
			JSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/deployments/{id}", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			deploymentHandler.Get(w, r)
		case http.MethodDelete:
			deploymentHandler.Delete(w, r)
		default:
			JSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/deployments/{id}/start", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			deploymentHandler.Start(w, r)
		} else {
			JSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/deployments/{id}/stop", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			deploymentHandler.Stop(w, r)
		} else {
			JSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/deployments/{id}/logs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			deploymentHandler.GetLogs(w, r)
		} else {
			JSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Events route
	mux.HandleFunc("/api/events", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			events, err := eventRepo.GetRecent(100)
			if err != nil {
				JSONError(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if events == nil {
				events = []models.Event{}
			}
			JSONResponse(w, events)
		} else {
			JSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Gateway routes
	mux.HandleFunc("/v1/models", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			gatewayProxy.ListModels(w, r)
		} else {
			JSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			gatewayProxy.ChatCompletions(w, r)
		} else {
			JSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Static files (frontend)
	staticDir := "frontend/build"
	if _, err := os.Stat(staticDir); os.IsNotExist(err) {
		staticDir = "/Users/rliu/Projects/ModelFleet/frontend/build"
	}
	if _, err := os.Stat(staticDir); err == nil {
		fs := http.FileServer(http.Dir(staticDir))
		mux.Handle("/", fs)
	}

	return mux
}

// Helper for generating IDs
func GenerateID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// API Key middleware (basic version - can be enhanced)
func APIKeyAuth(apiKeyRepo *repository.APIKeyRepository, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			JSONError(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		// Extract Bearer token
		parts := strings.SplitN(auth, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			JSONError(w, "Invalid authorization format. Use: Bearer <token>", http.StatusUnauthorized)
			return
		}

		token := parts[1]
		hash := repository.HashAPIKey(token)

		key, err := apiKeyRepo.ValidateKey(hash)
		if err != nil {
			JSONError(w, "Invalid API key", http.StatusUnauthorized)
			return
		}

		// Store key info in context for later use
		_ = key
		next(w, r)
	}
}
