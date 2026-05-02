package models

import "time"

type Machine struct {
	ID                    string `json:"id"`
	Name                  string `json:"name"`
	Host                  string `json:"host"`
	SSHPort               int    `json:"ssh_port"`
	SSHUser               string `json:"ssh_user"`
	AuthType              string `json:"auth_type"`
	AuthPrivateKeyPath    string `json:"auth_private_key_path,omitempty"`
	AuthPasswordEncrypted string `json:"-"`
	Tags                  string `json:"tags,omitempty"`
	Notes                 string `json:"notes,omitempty"`
	BaseDir               string `json:"base_dir"`
	ConnectionType        string `json:"connection_type"`
	// Docker-specific fields
	DockerHost    string    `json:"docker_host,omitempty"`
	DockerImage   string    `json:"docker_image,omitempty"`
	DockerNetwork string    `json:"docker_network,omitempty"`
	DockerRuntime string    `json:"docker_runtime,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type AddMachineRequest struct {
	Name               string `json:"name"`
	Host               string `json:"host"`
	SSHPort            int    `json:"ssh_port"`
	SSHUser            string `json:"ssh_user"`
	AuthType           string `json:"auth_type"`
	AuthPrivateKeyPath string `json:"auth_private_key_path,omitempty"`
	AuthPassword       string `json:"auth_password,omitempty"`
	Tags               string `json:"tags,omitempty"`
	Notes              string `json:"notes,omitempty"`
	BaseDir            string `json:"base_dir,omitempty"`
	ConnectionType     string `json:"connection_type,omitempty"`
	// Docker-specific
	DockerHost    string `json:"docker_host,omitempty"`
	DockerImage   string `json:"docker_image,omitempty"`
	DockerNetwork string `json:"docker_network,omitempty"`
	DockerRuntime string `json:"docker_runtime,omitempty"`
}
