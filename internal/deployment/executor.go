package deployment

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/modelfleet/modelfleet/internal/docker"
	"github.com/modelfleet/modelfleet/internal/events"
	"github.com/modelfleet/modelfleet/internal/models"
)

type Executor struct {
	eventLogger *events.Logger
}

func NewExecutor(eventLogger *events.Logger) *Executor {
	return &Executor{eventLogger: eventLogger}
}

type ExecutionResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

func (e *Executor) Start(deployment *models.Deployment, machine *models.Machine, model *models.Model) (*ExecutionResult, error) {
	sshArgs := e.buildSSHArgs(machine)
	baseDir := strings.Replace(machine.BaseDir, "~", "$HOME", 1)

	// Step 1: Create directories
	dirs := []string{"bin", "models", "runtimes", "deployments", "logs"}
	for _, dir := range dirs {
		cmd := exec.Command("ssh", append(sshArgs,
			fmt.Sprintf("mkdir -p %s", filepath.Join(baseDir, dir)))...)
		if output, err := cmd.CombinedOutput(); err != nil {
			return &ExecutionResult{
				Success: false,
				Message: fmt.Sprintf("Failed to create directory %s", dir),
				Error:   fmt.Sprintf("%v: %s", err, string(output)),
			}, nil
		}
	}

	// Step 2: Check if model file exists
	modelPath := filepath.Join(baseDir, "models", model.Filename)
	cmd := exec.Command("ssh", append(sshArgs,
		fmt.Sprintf("test -f %s && echo EXISTS || echo MISSING", modelPath))...)
	output, _ := cmd.Output()
	modelExists := strings.TrimSpace(string(output)) == "EXISTS"

	if !modelExists && model.SourceURL != "" {
		// Download model file
		e.eventLogger.LogDeployment(deployment.ID, machine.ID, "download_model", true, nil)
		cmd = exec.Command("ssh", append(sshArgs,
			fmt.Sprintf("cd %s/models && curl -L -o %s %s", baseDir, model.Filename, model.SourceURL))...)
		if output, err := cmd.CombinedOutput(); err != nil {
			return &ExecutionResult{
				Success: false,
				Message: "Failed to download model file",
				Error:   fmt.Sprintf("%v: %s", err, string(output)),
			}, nil
		}
	}

	// Step 3: Check if llama-server exists (simplified - would need runtime download in production)
	binPath := filepath.Join(baseDir, "bin", "llama-server")
	cmd = exec.Command("ssh", append(sshArgs,
		fmt.Sprintf("test -f %s && echo EXISTS || echo MISSING", binPath))...)
	output, _ = cmd.Output()
	binaryExists := strings.TrimSpace(string(output)) == "EXISTS"

	if !binaryExists {
		// In a real implementation, download the appropriate binary
		// For MVP, we assume the user has installed llama-server or we'll provide instructions
		return &ExecutionResult{
			Success: false,
			Message: "llama-server binary not found on remote machine",
			Error:   fmt.Sprintf("Binary not found at %s. Please download llama-server for %s backend and place it at %s/bin/llama-server", binPath, deployment.Backend, baseDir),
		}, nil
	}

	// Step 4: Make binary executable
	cmd = exec.Command("ssh", append(sshArgs,
		fmt.Sprintf("chmod +x %s", binPath))...)
	cmd.Run()

	// Step 5: Check if port is already in use
	cmd = exec.Command("ssh", append(sshArgs,
		fmt.Sprintf("lsof -i:%d > /dev/null 2>&1 && echo IN_USE || echo FREE", deployment.Port))...)
	output, _ = cmd.Output()
	if strings.TrimSpace(string(output)) == "IN_USE" {
		return &ExecutionResult{
			Success: false,
			Message: fmt.Sprintf("Port %d is already in use on machine %s", deployment.Port, machine.Name),
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

	startCmd := fmt.Sprintf("nohup %s %s > %s 2>&1 & echo $! > %s",
		binPath, strings.Join(args, " "), logFile, pidFile)

	cmd = exec.Command("ssh", append(sshArgs, startCmd)...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return &ExecutionResult{
			Success: false,
			Message: "Failed to start llama-server",
			Error:   fmt.Sprintf("%v: %s", err, string(output)),
		}, nil
	}

	// Step 8: Read PID
	cmd = exec.Command("ssh", append(sshArgs,
		fmt.Sprintf("cat %s", pidFile))...)
	pidOutput, _ := cmd.Output()
	pidStr := strings.TrimSpace(string(pidOutput))
	pid, _ := strconv.Atoi(pidStr)

	// Step 9: Wait a moment and verify process is running
	time.Sleep(2 * time.Second)
	cmd = exec.Command("ssh", append(sshArgs,
		fmt.Sprintf("kill -0 %d 2>/dev/null && echo RUNNING || echo STOPPED", pid))...)
	output, _ = cmd.Output()
	if strings.TrimSpace(string(output)) != "RUNNING" {
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

func (e *Executor) Stop(deployment *models.Deployment, machine *models.Machine) (*ExecutionResult, error) {
	sshArgs := e.buildSSHArgs(machine)
	baseDir := strings.Replace(machine.BaseDir, "~", "$HOME", 1)
	pidFile := filepath.Join(baseDir, "deployments", fmt.Sprintf("%s.pid", deployment.Name))

	// Read PID
	cmd := exec.Command("ssh", append(sshArgs,
		fmt.Sprintf("cat %s 2>/dev/null || echo NONE", pidFile))...)
	output, _ := cmd.Output()
	pidStr := strings.TrimSpace(string(output))

	if pidStr == "NONE" || pidStr == "" {
		return &ExecutionResult{
			Success: true,
			Message: "No PID file found, process may already be stopped",
		}, nil
	}

	pid, _ := strconv.Atoi(pidStr)

	// Kill process
	cmd = exec.Command("ssh", append(sshArgs,
		fmt.Sprintf("kill %d 2>/dev/null || true", pid))...)
	cmd.Run()

	// Wait and verify
	time.Sleep(1 * time.Second)
	cmd = exec.Command("ssh", append(sshArgs,
		fmt.Sprintf("kill -0 %d 2>/dev/null && echo RUNNING || echo STOPPED", pid))...)
	output, _ = cmd.Output()
	isRunning := strings.TrimSpace(string(output)) == "RUNNING"

	if isRunning {
		// Force kill
		cmd = exec.Command("ssh", append(sshArgs,
			fmt.Sprintf("kill -9 %d 2>/dev/null || true", pid))...)
		cmd.Run()
	}

	// Remove PID file
	cmd = exec.Command("ssh", append(sshArgs,
		fmt.Sprintf("rm -f %s", pidFile))...)
	cmd.Run()

	return &ExecutionResult{
		Success: true,
		Message: fmt.Sprintf("Process stopped (PID: %d)", pid),
	}, nil
}

func (e *Executor) GetLogs(deployment *models.Deployment, machine *models.Machine, lines int) (string, error) {
	sshArgs := e.buildSSHArgs(machine)
	baseDir := strings.Replace(machine.BaseDir, "~", "$HOME", 1)
	logFile := filepath.Join(baseDir, "logs", fmt.Sprintf("%s.log", deployment.Name))

	if lines <= 0 {
		lines = 100
	}

	cmd := exec.Command("ssh", append(sshArgs,
		fmt.Sprintf("tail -n %d %s 2>/dev/null || echo 'No logs available'", lines, logFile))...)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return string(output), nil
}

func (e *Executor) buildSSHArgs(machine *models.Machine) []string {
	args := []string{
		"-o", "StrictHostKeyChecking=no",
		"-o", "ConnectTimeout=10",
		"-o", "BatchMode=yes",
		"-p", strconv.Itoa(machine.SSHPort),
	}

	if machine.AuthType == "key" && machine.AuthPrivateKeyPath != "" {
		args = append(args, "-i", machine.AuthPrivateKeyPath)
	}

	args = append(args, fmt.Sprintf("%s@%s", machine.SSHUser, machine.Host))
	return args
}

// Docker deployment methods
func (e *Executor) StartDocker(deployment *models.Deployment, machine *models.Machine, model *models.Model) (*ExecutionResult, error) {
	client := docker.NewClient(machine.DockerHost)
	containerName := fmt.Sprintf("modelfleet-%s", deployment.Name)

	// Step 1: Check if container already exists
	exists, err := client.ContainerExists(containerName)
	if err != nil {
		return &ExecutionResult{
			Success: false,
			Message: "Failed to check container status",
			Error:   err.Error(),
		}, nil
	}

	if exists {
		// Check if running
		running, err := client.ContainerRunning(containerName)
		if err != nil {
			return &ExecutionResult{
				Success: false,
				Message: "Failed to check container status",
				Error:   err.Error(),
			}, nil
		}

		if running {
			return &ExecutionResult{
				Success: true,
				Message: fmt.Sprintf("Container %s is already running", containerName),
			}, nil
		}

		// Start existing container
		if err := client.StartContainer(containerName); err != nil {
			return &ExecutionResult{
				Success: false,
				Message: "Failed to start existing container",
				Error:   err.Error(),
			}, nil
		}

		return &ExecutionResult{
			Success: true,
			Message: fmt.Sprintf("Container %s started", containerName),
		}, nil
	}

	// Step 2: Pull image
	e.eventLogger.LogDeployment(deployment.ID, machine.ID, "pull_image", true, nil)
	if err := client.PullImage(machine.DockerImage); err != nil {
		return &ExecutionResult{
			Success: false,
			Message: "Failed to pull Docker image",
			Error:   err.Error(),
		}, nil
	}

	// Step 3: Prepare volume mounts
	// Map model file from host to container
	modelHostPath := filepath.Join(machine.BaseDir, "models", model.Filename)
	modelContainerPath := "/models/" + model.Filename

	volumes := map[string]string{
		modelHostPath: modelContainerPath,
	}

	// Step 4: Prepare port mappings
	ports := map[string]string{
		fmt.Sprintf("%d", deployment.Port): fmt.Sprintf("%d", deployment.Port),
	}

	// Step 5: Build command arguments
	env := map[string]string{}

	// Step 6: Create and start container
	if err := client.CreateContainer(
		containerName,
		machine.DockerImage,
		ports,
		volumes,
		env,
		machine.DockerNetwork,
	); err != nil {
		return &ExecutionResult{
			Success: false,
			Message: "Failed to create container",
			Error:   err.Error(),
		}, nil
	}

	// Wait a moment and verify
	time.Sleep(2 * time.Second)
	running, err := client.ContainerRunning(containerName)
	if err != nil || !running {
		return &ExecutionResult{
			Success: false,
			Message: "Container created but not running",
			Error:   "Check logs for details",
		}, nil
	}

	return &ExecutionResult{
		Success: true,
		Message: fmt.Sprintf("Container %s started on port %d", containerName, deployment.Port),
	}, nil
}

func (e *Executor) StopDocker(deployment *models.Deployment, machine *models.Machine) (*ExecutionResult, error) {
	client := docker.NewClient(machine.DockerHost)
	containerName := fmt.Sprintf("modelfleet-%s", deployment.Name)

	exists, err := client.ContainerExists(containerName)
	if err != nil {
		return &ExecutionResult{
			Success: false,
			Message: "Failed to check container status",
			Error:   err.Error(),
		}, nil
	}

	if !exists {
		return &ExecutionResult{
			Success: true,
			Message: "Container does not exist",
		}, nil
	}

	if err := client.StopContainer(containerName); err != nil {
		return &ExecutionResult{
			Success: false,
			Message: "Failed to stop container",
			Error:   err.Error(),
		}, nil
	}

	return &ExecutionResult{
		Success: true,
		Message: fmt.Sprintf("Container %s stopped", containerName),
	}, nil
}

func (e *Executor) GetDockerLogs(deployment *models.Deployment, machine *models.Machine, lines int) (string, error) {
	client := docker.NewClient(machine.DockerHost)
	containerName := fmt.Sprintf("modelfleet-%s", deployment.Name)

	if lines <= 0 {
		lines = 100
	}

	return client.GetContainerLogs(containerName, lines)
}
