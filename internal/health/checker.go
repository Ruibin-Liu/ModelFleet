package health

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/modelfleet/modelfleet/internal/events"
	"github.com/modelfleet/modelfleet/internal/models"
	"github.com/modelfleet/modelfleet/internal/repository"
)

type Checker struct {
	client      *http.Client
	eventLogger *events.Logger
}

func NewChecker(eventLogger *events.Logger) *Checker {
	return &Checker{
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		eventLogger: eventLogger,
	}
}

func (c *Checker) Check(deployment *models.Deployment) (string, string) {
	// TCP check
	addr := fmt.Sprintf("%s:%d", deployment.Host, deployment.Port)
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		msg := fmt.Sprintf("TCP connect failed to %s: %v", addr, err)
		c.eventLogger.LogHealthCheck(deployment.ID, deployment.MachineID, "unreachable", msg)
		return "unreachable", msg
	}
	conn.Close()

	// HTTP check - try the health endpoint first
	healthURL := fmt.Sprintf("http://%s/health", addr)
	resp, err := c.client.Get(healthURL)
	if err == nil && resp.StatusCode == http.StatusOK {
		resp.Body.Close()
		msg := fmt.Sprintf("Deployment %s is healthy (HTTP %d)", deployment.Name, resp.StatusCode)
		c.eventLogger.LogHealthCheck(deployment.ID, deployment.MachineID, "healthy", msg)
		return "healthy", msg
	}
	if resp != nil {
		resp.Body.Close()
	}

	// Try the models endpoint (llama-server specific)
	modelsURL := fmt.Sprintf("http://%s/v1/models", addr)
	resp, err = c.client.Get(modelsURL)
	if err == nil && resp.StatusCode == http.StatusOK {
		resp.Body.Close()
		msg := fmt.Sprintf("Deployment %s is healthy (models endpoint OK)", deployment.Name)
		c.eventLogger.LogHealthCheck(deployment.ID, deployment.MachineID, "healthy", msg)
		return "healthy", msg
	}
	if resp != nil {
		resp.Body.Close()
	}

	// Try a simple GET to root
	rootURL := fmt.Sprintf("http://%s", addr)
	resp, err = c.client.Get(rootURL)
	if err == nil {
		resp.Body.Close()
		if resp.StatusCode < 500 {
			msg := fmt.Sprintf("Deployment %s is degraded (HTTP %d)", deployment.Name, resp.StatusCode)
			c.eventLogger.LogHealthCheck(deployment.ID, deployment.MachineID, "degraded", msg)
			return "degraded", msg
		}
	}

	msg := fmt.Sprintf("Deployment %s is unreachable: %v", deployment.Name, err)
	c.eventLogger.LogHealthCheck(deployment.ID, deployment.MachineID, "unreachable", msg)
	return "unreachable", msg
}

type Scheduler struct {
	checker        *Checker
	deploymentRepo *repository.DeploymentRepository
	interval       time.Duration
	stop           chan bool
}

func NewScheduler(deploymentRepo *repository.DeploymentRepository, eventLogger *events.Logger) *Scheduler {
	return &Scheduler{
		checker:        NewChecker(eventLogger),
		deploymentRepo: deploymentRepo,
		interval:       30 * time.Second,
		stop:           make(chan bool),
	}
}

func (s *Scheduler) Start() {
	ticker := time.NewTicker(s.interval)
	go func() {
		for {
			select {
			case <-ticker.C:
				s.checkAll()
			case <-s.stop:
				ticker.Stop()
				return
			}
		}
	}()
}

func (s *Scheduler) Stop() {
	close(s.stop)
}

func (s *Scheduler) checkAll() {
	deployments, err := s.deploymentRepo.GetAll()
	if err != nil {
		return
	}

	for _, d := range deployments {
		// Only check deployments that should be running
		if d.Status != "running" && d.Status != "unhealthy" {
			continue
		}

		state, _ := s.checker.Check(&d)

		// Update deployment status based on health
		switch state {
		case "healthy":
			if d.Status != "running" {
				s.deploymentRepo.UpdateStatus(d.ID, "running")
			}
		case "degraded":
			if d.Status != "degraded" {
				s.deploymentRepo.UpdateStatus(d.ID, "degraded")
			}
		case "unreachable":
			if d.Status != "unhealthy" {
				s.deploymentRepo.UpdateStatus(d.ID, "unhealthy")
			}
		}
	}
}
