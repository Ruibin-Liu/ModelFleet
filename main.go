package main

import (
	"log"
	"net/http"
	"os"

	"github.com/modelfleet/modelfleet/internal/api"
	"github.com/modelfleet/modelfleet/internal/events"
	"github.com/modelfleet/modelfleet/internal/health"
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
	factsRepo := repository.NewMachineFactsRepository(jsonStore)
	eventRepo := repository.NewEventRepository(jsonStore)
	apiKeyRepo := repository.NewAPIKeyRepository(jsonStore)

	// Initialize event logger
	eventLogger := events.NewLogger(jsonStore)

	// Initialize health check scheduler
	healthScheduler := health.NewScheduler(deploymentRepo, eventLogger)
	healthScheduler.Start()
	defer healthScheduler.Stop()

	// Create router
	router := api.NewRouter(machineRepo, modelRepo, deploymentRepo, factsRepo, eventRepo, apiKeyRepo, eventLogger)

	log.Printf("ModelFleet starting on port %s", port)
	log.Printf("API: http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatal(err)
	}
}
