package main

import (
	"log"
	"net/http"
	"os"

	"github.com/modelfleet/modelfleet/internal/api"
	"github.com/modelfleet/modelfleet/internal/repository"
	"github.com/modelfleet/modelfleet/internal/store"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3456"
	}

	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
	}

	// Initialize JSON store
	jsonStore, err := store.New(dataDir)
	if err != nil {
		log.Fatal("Failed to initialize store:", err)
	}

	// Initialize repositories
	machineRepo := repository.NewMachineRepository(jsonStore)
	modelRepo := repository.NewModelRepository(jsonStore)
	deploymentRepo := repository.NewDeploymentRepository(jsonStore)

	// Create router
	router := api.NewRouter(machineRepo, modelRepo, deploymentRepo)

	log.Printf("ModelFleet starting on port %s", port)
	log.Printf("API: http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatal(err)
	}
}
