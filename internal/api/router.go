package api

import (
	"net/http"
	"os"

	"github.com/modelfleet/modelfleet/internal/repository"
)

func NewRouter(machineRepo *repository.MachineRepository, modelRepo *repository.ModelRepository, deploymentRepo *repository.DeploymentRepository) http.Handler {
	mux := http.NewServeMux()

	machineHandler := NewMachineHandler(machineRepo)
	modelHandler := NewModelHandler(modelRepo)
	deploymentHandler := NewDeploymentHandler(deploymentRepo, machineRepo, modelRepo)
	gatewayHandler := NewGatewayHandler(deploymentRepo)

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

	// Gateway routes
	mux.HandleFunc("/v1/models", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			gatewayHandler.ListModels(w, r)
		} else {
			JSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			gatewayHandler.ChatCompletions(w, r)
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
