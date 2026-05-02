package docker

import (
	"fmt"
	"os/exec"
	"strings"
)

type Client struct {
	host string // Docker host, e.g., "unix:///var/run/docker.sock" or "tcp://remote:2376"
}

func NewClient(host string) *Client {
	if host == "" {
		host = "unix:///var/run/docker.sock"
	}
	return &Client{host: host}
}

func (c *Client) cmd(args ...string) *exec.Cmd {
	cmdArgs := append([]string{"-H", c.host}, args...)
	return exec.Command("docker", cmdArgs...)
}

// TestConnection checks if Docker daemon is accessible
func (c *Client) TestConnection() error {
	cmd := c.cmd("version", "--format", "{{.Server.Version}}")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker connection failed: %v - %s", err, string(output))
	}
	return nil
}

// PullImage pulls a Docker image
func (c *Client) PullImage(image string) error {
	cmd := c.cmd("pull", image)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker pull failed: %v - %s", err, string(output))
	}
	return nil
}

// CreateContainer creates a new container
func (c *Client) CreateContainer(name, image string, ports map[string]string, volumes map[string]string, env map[string]string, network string) error {
	args := []string{
		"run",
		"-d",
		"--name", name,
	}

	// Port mappings
	for hostPort, containerPort := range ports {
		args = append(args, "-p", fmt.Sprintf("%s:%s", hostPort, containerPort))
	}

	// Volume mappings
	for hostPath, containerPath := range volumes {
		args = append(args, "-v", fmt.Sprintf("%s:%s", hostPath, containerPath))
	}

	// Environment variables
	for key, value := range env {
		args = append(args, "-e", fmt.Sprintf("%s=%s", key, value))
	}

	// Network
	if network != "" {
		args = append(args, "--network", network)
	}

	args = append(args, image)

	cmd := c.cmd(args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker run failed: %v - %s", err, string(output))
	}
	return nil
}

// StartContainer starts an existing container
func (c *Client) StartContainer(name string) error {
	cmd := c.cmd("start", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker start failed: %v - %s", err, string(output))
	}
	return nil
}

// StopContainer stops a container
func (c *Client) StopContainer(name string) error {
	cmd := c.cmd("stop", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker stop failed: %v - %s", err, string(output))
	}
	return nil
}

// RemoveContainer removes a container
func (c *Client) RemoveContainer(name string) error {
	cmd := c.cmd("rm", "-f", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker rm failed: %v - %s", err, string(output))
	}
	return nil
}

// ContainerExists checks if a container exists
func (c *Client) ContainerExists(name string) (bool, error) {
	cmd := c.cmd("ps", "-a", "--filter", fmt.Sprintf("name=%s", name), "--format", "{{.Names}}")
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}
	containers := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, container := range containers {
		if strings.TrimSpace(container) == name {
			return true, nil
		}
	}
	return false, nil
}

// ContainerRunning checks if a container is running
func (c *Client) ContainerRunning(name string) (bool, error) {
	cmd := c.cmd("ps", "--filter", fmt.Sprintf("name=%s", name), "--format", "{{.Names}}")
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}
	containers := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, container := range containers {
		if strings.TrimSpace(container) == name {
			return true, nil
		}
	}
	return false, nil
}

// GetContainerLogs gets logs from a container
func (c *Client) GetContainerLogs(name string, lines int) (string, error) {
	if lines <= 0 {
		lines = 100
	}
	cmd := c.cmd("logs", "--tail", fmt.Sprintf("%d", lines), name)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("docker logs failed: %v", err)
	}
	return string(output), nil
}

// GetContainerPort gets the host port mapped to a container port
func (c *Client) GetContainerPort(name string, containerPort string) (string, error) {
	cmd := c.cmd("port", name, containerPort)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("docker port failed: %v", err)
	}
	// Output format: "0.0.0.0:8080"
	parts := strings.Split(strings.TrimSpace(string(output)), ":")
	if len(parts) >= 2 {
		return parts[len(parts)-1], nil
	}
	return "", fmt.Errorf("could not parse port from: %s", string(output))
}

// ListContainers lists all containers
func (c *Client) ListContainers() ([]ContainerInfo, error) {
	cmd := c.cmd("ps", "-a", "--format", "{{.Names}}|{{.Image}}|{{.Status}}|{{.Ports}}")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("docker ps failed: %v", err)
	}

	var containers []ContainerInfo
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) >= 3 {
			containers = append(containers, ContainerInfo{
				Name:   parts[0],
				Image:  parts[1],
				Status: parts[2],
				Ports:  parts[3],
			})
		}
	}
	return containers, nil
}

type ContainerInfo struct {
	Name   string
	Image  string
	Status string
	Ports  string
}
