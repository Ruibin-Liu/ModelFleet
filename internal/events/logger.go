package events

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/modelfleet/modelfleet/internal/models"
	"github.com/modelfleet/modelfleet/internal/store"
)

type Logger struct {
	store *store.JSONStore
}

func NewLogger(store *store.JSONStore) *Logger {
	return &Logger{store: store}
}

func (l *Logger) Log(eventType, severity, message string, machineID, deploymentID string, data map[string]interface{}) {
	event := models.Event{
		ID:        generateEventID(),
		Type:      eventType,
		Severity:  severity,
		Message:   message,
		CreatedAt: time.Now().UTC(),
	}

	if machineID != "" {
		event.MachineID = machineID
	}
	if deploymentID != "" {
		event.DeploymentID = deploymentID
	}
	if data != nil {
		bytes, _ := json.Marshal(data)
		event.DataJSON = string(bytes)
	}

	// Store to JSON file
	if err := l.store.Save("events", event.ID, event); err != nil {
		log.Printf("Failed to save event: %v", err)
	}

	// Also log to stdout (structured)
	log.Printf("[%s] %s: %s", severity, eventType, message)
}

func (l *Logger) LogSSH(machineID string, success bool, err error) {
	severity := "info"
	msg := "SSH connection successful"
	if !success {
		severity = "error"
		msg = fmt.Sprintf("SSH connection failed: %v", err)
	}
	l.Log("ssh", severity, msg, machineID, "", nil)
}

func (l *Logger) LogDetection(machineID string, success bool, err error) {
	severity := "info"
	msg := "Hardware detection completed"
	if !success {
		severity = "error"
		msg = fmt.Sprintf("Hardware detection failed: %v", err)
	}
	l.Log("detection", severity, msg, machineID, "", nil)
}

func (l *Logger) LogDeployment(deploymentID, machineID, action string, success bool, err error) {
	severity := "info"
	msg := fmt.Sprintf("Deployment %s: %s", action, "success")
	if !success {
		severity = "error"
		msg = fmt.Sprintf("Deployment %s failed: %v", action, err)
	}
	l.Log("deployment", severity, msg, machineID, deploymentID, map[string]interface{}{
		"action": action,
	})
}

func (l *Logger) LogHealthCheck(deploymentID, machineID, state, message string) {
	severity := "info"
	if state == "unreachable" || state == "unhealthy" {
		severity = "warning"
	}
	l.Log("health_check", severity, message, machineID, deploymentID, map[string]interface{}{
		"state": state,
	})
}

func (l *Logger) LogGateway(deploymentID, model string, success bool, err error) {
	severity := "info"
	msg := fmt.Sprintf("Gateway request for model %s", model)
	if !success {
		severity = "error"
		msg = fmt.Sprintf("Gateway routing failed for model %s: %v", model, err)
	}
	l.Log("gateway", severity, msg, "", deploymentID, map[string]interface{}{
		"model": model,
	})
}

func (l *Logger) GetRecent(limit int) ([]models.Event, error) {
	data, err := l.store.GetAll("events")
	if err != nil {
		return nil, err
	}

	var events []models.Event
	for _, item := range data {
		var e models.Event
		if err := json.Unmarshal(item, &e); err != nil {
			continue
		}
		events = append(events, e)
	}

	// Sort by created_at descending (newest first)
	for i := 0; i < len(events); i++ {
		for j := i + 1; j < len(events); j++ {
			if events[j].CreatedAt.After(events[i].CreatedAt) {
				events[i], events[j] = events[j], events[i]
			}
		}
	}

	if limit > 0 && len(events) > limit {
		events = events[:limit]
	}

	return events, nil
}

func generateEventID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
