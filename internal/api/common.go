package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"

	"github.com/modelfleet/modelfleet/internal/models"
)

func generateID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func JSONResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func JSONError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]interface{}{
			"message": message,
			"type":    "api_error",
		},
	})
}

func testSSHConnection(machine *models.Machine) (map[string]interface{}, error) {
	var cmd *exec.Cmd

	if machine.AuthType == "key" {
		keyPath := machine.AuthPrivateKeyPath
		if keyPath == "" {
			keyPath = "~/.ssh/id_rsa"
		}
		cmd = exec.Command("ssh",
			"-o", "StrictHostKeyChecking=no",
			"-o", "ConnectTimeout=10",
			"-i", keyPath,
			"-p", fmt.Sprintf("%d", machine.SSHPort),
			fmt.Sprintf("%s@%s", machine.SSHUser, machine.Host),
			"echo MODEL_FLEET_SSH_OK")
	} else {
		cmd = exec.Command("ssh",
			"-o", "StrictHostKeyChecking=no",
			"-o", "ConnectTimeout=10",
			"-p", fmt.Sprintf("%d", machine.SSHPort),
			fmt.Sprintf("%s@%s", machine.SSHUser, machine.Host),
			"echo MODEL_FLEET_SSH_OK")
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("SSH connection failed: %v - %s", err, string(output)),
		}, nil
	}

	if strings.TrimSpace(string(output)) == "MODEL_FLEET_SSH_OK" {
		return map[string]interface{}{
			"success": true,
			"message": "SSH connection successful",
		}, nil
	}

	return map[string]interface{}{
		"success": false,
		"error":   fmt.Sprintf("Unexpected response: %s", string(output)),
	}, nil
}

type DetectionResult struct {
	OSName           string `json:"os_name"`
	OSVersion        string `json:"os_version"`
	Arch             string `json:"arch"`
	CPUModel         string `json:"cpu_model"`
	CPUCores         int    `json:"cpu_cores"`
	RAMBytes         int64  `json:"ram_bytes"`
	DiskFreeBytes    int64  `json:"disk_free_bytes"`
	GPUVendor        string `json:"gpu_vendor"`
	GPUName          string `json:"gpu_name"`
	GPUVRAMBytes     int64  `json:"gpu_vram_bytes"`
	GPUDriverVersion string `json:"gpu_driver_version"`
	CapabilityState  string `json:"capability_state"`
	HasNVIDIA        bool   `json:"has_nvidia"`
	HasAMD           bool   `json:"has_amd"`
	HasVulkan        bool   `json:"has_vulkan"`
}

func detectMachine(machine *models.Machine) (*DetectionResult, error) {
	result := &DetectionResult{}

	// Run detection via SSH
	var sshArgs []string
	if machine.AuthType == "key" && machine.AuthPrivateKeyPath != "" {
		sshArgs = append(sshArgs, "-i", machine.AuthPrivateKeyPath)
	}
	sshArgs = append(sshArgs,
		"-o", "StrictHostKeyChecking=no",
		"-o", "ConnectTimeout=10",
		"-p", fmt.Sprintf("%d", machine.SSHPort),
		fmt.Sprintf("%s@%s", machine.SSHUser, machine.Host),
	)

	// Detect OS
	cmd := exec.Command("ssh", append(sshArgs, "cat /etc/os-release")...)
	output, err := cmd.Output()
	if err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "NAME=") {
				result.OSName = strings.Trim(strings.TrimPrefix(line, "NAME="), `"`)
			}
			if strings.HasPrefix(line, "VERSION_ID=") {
				result.OSVersion = strings.Trim(strings.TrimPrefix(line, "VERSION_ID="), `"`)
			}
		}
	}

	// Detect arch
	cmd = exec.Command("ssh", append(sshArgs, "uname -m")...)
	output, _ = cmd.Output()
	result.Arch = strings.TrimSpace(string(output))

	// Detect CPU
	cmd = exec.Command("ssh", append(sshArgs, "lscpu")...)
	output, _ = cmd.Output()
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "Model name:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) > 1 {
				result.CPUModel = strings.TrimSpace(parts[1])
			}
		}
		if strings.Contains(line, "CPU(s):") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) > 1 {
				var cores int
				fmt.Sscanf(strings.TrimSpace(parts[1]), "%d", &cores)
				result.CPUCores = cores
			}
		}
	}

	// Detect memory
	cmd = exec.Command("ssh", append(sshArgs, "free -b | awk 'NR==2{print $2}'")...)
	output, _ = cmd.Output()
	fmt.Sscanf(strings.TrimSpace(string(output)), "%d", &result.RAMBytes)

	// Detect disk
	cmd = exec.Command("ssh", append(sshArgs, "df -B1 ~ | awk 'NR==2{print $4}'")...)
	output, _ = cmd.Output()
	fmt.Sscanf(strings.TrimSpace(string(output)), "%d", &result.DiskFreeBytes)

	// Detect GPU
	cmd = exec.Command("ssh", append(sshArgs, "command -v nvidia-smi")...)
	err = cmd.Run()
	result.HasNVIDIA = (err == nil)

	if result.HasNVIDIA {
		cmd = exec.Command("ssh", append(sshArgs, "nvidia-smi --query-gpu=name,memory.total,driver_version --format=csv,noheader")...)
		output, err = cmd.Output()
		if err == nil {
			parts := strings.Split(strings.TrimSpace(string(output)), ", ")
			if len(parts) >= 3 {
				result.GPUVendor = "NVIDIA"
				result.GPUName = strings.TrimSpace(parts[0])
				result.GPUDriverVersion = strings.TrimSpace(parts[2])
				memStr := strings.TrimSpace(strings.TrimSuffix(parts[1], " MiB"))
				var memMiB int64
				fmt.Sscanf(memStr, "%d", &memMiB)
				result.GPUVRAMBytes = memMiB * 1024 * 1024
			}
		}
	}

	// Check AMD
	cmd = exec.Command("ssh", append(sshArgs, "command -v rocminfo")...)
	err = cmd.Run()
	result.HasAMD = (err == nil)

	// Check Vulkan
	cmd = exec.Command("ssh", append(sshArgs, "command -v vulkaninfo")...)
	err = cmd.Run()
	result.HasVulkan = (err == nil)

	// Determine capability
	if result.HasNVIDIA && result.GPUName != "" {
		result.CapabilityState = "nvidia_ready"
	} else if result.HasAMD {
		result.CapabilityState = "amd_rocm_ready"
	} else if result.HasVulkan {
		result.CapabilityState = "vulkan_ready"
	} else {
		result.CapabilityState = "cpu_only"
	}

	return result, nil
}
