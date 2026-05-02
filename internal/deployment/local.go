package deployment

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/modelfleet/modelfleet/internal/models"
)

type LocalExecutor struct{}

func NewLocalExecutor() *LocalExecutor {
	return &LocalExecutor{}
}

func (e *LocalExecutor) Start(deployment *models.Deployment, machine *models.Machine, model *models.Model) (*ExecutionResult, error) {
	baseDir := os.ExpandEnv(strings.Replace(machine.BaseDir, "~", "$HOME", 1))

	// Step 1: Create directories
	dirs := []string{"bin", "models", "runtimes", "deployments", "logs"}
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(baseDir, dir), 0755); err != nil {
			return &ExecutionResult{
				Success: false,
				Message: fmt.Sprintf("Failed to create directory %s", dir),
				Error:   err.Error(),
			}, nil
		}
	}

	// Step 2: Check if model file exists
	modelPath := filepath.Join(baseDir, "models", model.Filename)
	_, err := os.Stat(modelPath)
	modelExists := (err == nil)

	if !modelExists && model.SourceURL != "" {
		// Download model file
		cmd := exec.Command("curl", "-L", "-o", modelPath, model.SourceURL)
		if output, err := cmd.CombinedOutput(); err != nil {
			return &ExecutionResult{
				Success: false,
				Message: "Failed to download model file",
				Error:   fmt.Sprintf("%v: %s", err, string(output)),
			}, nil
		}
	}

	// Step 3: Check if llama-server exists
	binPath := filepath.Join(baseDir, "bin", "llama-server")
	_, err = os.Stat(binPath)
	binaryExists := (err == nil)

	if !binaryExists {
		return &ExecutionResult{
			Success: false,
			Message: "llama-server binary not found",
			Error:   fmt.Sprintf("Binary not found at %s. Please download llama-server for %s backend and place it at %s/bin/llama-server", binPath, deployment.Backend, baseDir),
		}, nil
	}

	// Step 4: Make binary executable
	os.Chmod(binPath, 0755)

	// Step 5: Check if port is already in use
	cmd := exec.Command("lsof", "-i", fmt.Sprintf(":%d", deployment.Port))
	if err := cmd.Run(); err == nil {
		return &ExecutionResult{
			Success: false,
			Message: fmt.Sprintf("Port %d is already in use", deployment.Port),
			Error:   "Choose a different port or stop the existing process",
		}, nil
	}

	// Step 6: Build llama-server command
	args := []string{
		"-m", modelPath,
		"--host", deployment.Host,
		"--port", strconv.Itoa(deployment.Port),
	}

	if deployment.ContextSize > 0 {
		args = append(args, "-c", strconv.Itoa(deployment.ContextSize))
	}

	if deployment.GPULayers > 0 {
		args = append(args, "-ngl", strconv.Itoa(deployment.GPULayers))
	}

	if deployment.Threads > 0 {
		args = append(args, "-t", strconv.Itoa(deployment.Threads))
	}

	// Step 7: Start llama-server in background
	logFile := filepath.Join(baseDir, "logs", fmt.Sprintf("%s.log", deployment.Name))
	pidFile := filepath.Join(baseDir, "deployments", fmt.Sprintf("%s.pid", deployment.Name))

	cmd = exec.Command(binPath, args...)
	logF, _ := os.Create(logFile)
	if logF != nil {
		cmd.Stdout = logF
		cmd.Stderr = logF
	}

	if err := cmd.Start(); err != nil {
		return &ExecutionResult{
			Success: false,
			Message: "Failed to start llama-server",
			Error:   err.Error(),
		}, nil
	}

	pid := cmd.Process.Pid

	// Save PID
	os.WriteFile(pidFile, []byte(strconv.Itoa(pid)), 0644)

	// Step 8: Wait a moment and verify process is running
	time.Sleep(2 * time.Second)
	process, err := os.FindProcess(pid)
	if err != nil || process == nil {
		return &ExecutionResult{
			Success: false,
			Message: "llama-server started but process is not running",
			Error:   "Check logs for details",
		}, nil
	}

	return &ExecutionResult{
		Success: true,
		Message: fmt.Sprintf("llama-server started on %s:%d (PID: %d)", deployment.Host, deployment.Port, pid),
	}, nil
}

func (e *LocalExecutor) Stop(deployment *models.Deployment, machine *models.Machine) (*ExecutionResult, error) {
	baseDir := os.ExpandEnv(strings.Replace(machine.BaseDir, "~", "$HOME", 1))
	pidFile := filepath.Join(baseDir, "deployments", fmt.Sprintf("%s.pid", deployment.Name))

	// Read PID
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return &ExecutionResult{
			Success: true,
			Message: "No PID file found, process may already be stopped",
		}, nil
	}

	pidStr := strings.TrimSpace(string(data))
	pid, _ := strconv.Atoi(pidStr)

	if pid <= 0 {
		return &ExecutionResult{
			Success: true,
			Message: "Invalid PID, process may already be stopped",
		}, nil
	}

	// Kill process
	process, err := os.FindProcess(pid)
	if err == nil && process != nil {
		process.Kill()
	}

	// Wait and verify
	time.Sleep(1 * time.Second)
	process, _ = os.FindProcess(pid)
	if process != nil {
		// Force kill
		exec.Command("kill", "-9", strconv.Itoa(pid)).Run()
	}

	// Remove PID file
	os.Remove(pidFile)

	return &ExecutionResult{
		Success: true,
		Message: fmt.Sprintf("Process stopped (PID: %d)", pid),
	}, nil
}

func (e *LocalExecutor) GetLogs(deployment *models.Deployment, machine *models.Machine, lines int) (string, error) {
	baseDir := os.ExpandEnv(strings.Replace(machine.BaseDir, "~", "$HOME", 1))
	logFile := filepath.Join(baseDir, "logs", fmt.Sprintf("%s.log", deployment.Name))

	if lines <= 0 {
		lines = 100
	}

	cmd := exec.Command("tail", "-n", strconv.Itoa(lines), logFile)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return string(output), nil
}
