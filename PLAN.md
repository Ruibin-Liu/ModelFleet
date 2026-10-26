# ModelFleet MVP Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build ModelFleet Phase 1 MVP - a lightweight control plane for managing LLM serving across user-owned machines with unified OpenAI-compatible API.

**Architecture:** Go backend with embedded SvelteKit frontend, SQLite database, SSH-based remote machine management, llama.cpp runtime deployment, and OpenAI-compatible API gateway.

**Tech Stack:** Go 1.21+, SvelteKit, SQLite, native SSH client, llama.cpp

---

## Phase 1A: Project Foundation

### Task 1: Initialize Go Project Structure

**Files:**
- Create: `go.mod`
- Create: `main.go`
- Create: `README.md`
- Create: `.gitignore`

**Step 1: Initialize Go module**

```bash
cd /Users/rliu/Projects/ModelFleet
go mod init github.com/modelfleet/modelfleet
```

**Step 2: Create main.go with basic HTTP server**

```go
package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3456"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	log.Printf("ModelFleet starting on port %s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}
```

**Step 3: Test server starts**

```bash
go run main.go &
curl http://localhost:3456/health
# Expected: {"status":"ok"}
kill %1
```

**Step 4: Commit**

```bash
git add .
git commit -m "feat: initialize Go project with basic HTTP server"
```

---

### Task 2: Database Schema & Connection

**Files:**
- Create: `internal/db/db.go`
- Create: `internal/db/schema.sql`
- Create: `internal/db/migrations.go`

**Step 1: Add SQLite dependency**

```bash
go get github.com/mattn/go-sqlite3
```

**Step 2: Create database package with schema**

```go
// internal/db/db.go
package db

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed schema.sql
var schemaFS embed.FS

type DB struct {
	*sql.DB
}

func New(dataDir string) (*DB, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	dbPath := filepath.Join(dataDir, "modelfleet.db")
	sqlDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}

	schema, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		return nil, fmt.Errorf("read schema: %w", err)
	}

	if _, err := sqlDB.Exec(string(schema)); err != nil {
		return nil, fmt.Errorf("exec schema: %w", err)
	}

	return &DB{sqlDB}, nil
}
```

**Step 3: Create schema.sql**

```sql
-- internal/db/schema.sql
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS machines (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  host TEXT NOT NULL,
  ssh_port INTEGER NOT NULL DEFAULT 22,
  ssh_user TEXT NOT NULL,
  auth_type TEXT NOT NULL,
  auth_private_key_path TEXT,
  auth_password_encrypted TEXT,
  tags TEXT,
  notes TEXT,
  base_dir TEXT NOT NULL DEFAULT '~/modelfleet',
  connection_type TEXT NOT NULL DEFAULT 'ssh',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS machine_facts (
  machine_id TEXT PRIMARY KEY,
  os_name TEXT,
  os_version TEXT,
  arch TEXT,
  cpu_model TEXT,
  cpu_cores INTEGER,
  ram_bytes INTEGER,
  disk_free_bytes INTEGER,
  gpu_vendor TEXT,
  gpu_name TEXT,
  gpu_vram_bytes INTEGER,
  gpu_driver_version TEXT,
  capability_state TEXT,
  raw_json TEXT,
  detected_at TEXT NOT NULL,
  FOREIGN KEY (machine_id) REFERENCES machines(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS models (
  id TEXT PRIMARY KEY,
  display_name TEXT NOT NULL,
  family TEXT,
  parameter_size TEXT,
  format TEXT NOT NULL,
  quantization TEXT,
  context_length INTEGER,
  source_url TEXT,
  filename TEXT NOT NULL,
  sha256 TEXT,
  estimated_ram_bytes INTEGER,
  estimated_vram_bytes INTEGER,
  tags TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS runtime_artifacts (
  id TEXT PRIMARY KEY,
  runtime TEXT NOT NULL,
  version TEXT NOT NULL,
  os TEXT NOT NULL,
  arch TEXT NOT NULL,
  backend TEXT NOT NULL,
  url TEXT NOT NULL,
  sha256 TEXT NOT NULL,
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS deployments (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  machine_id TEXT NOT NULL,
  model_id TEXT NOT NULL,
  runtime TEXT NOT NULL,
  backend TEXT NOT NULL,
  status TEXT NOT NULL,
  host TEXT NOT NULL,
  port INTEGER NOT NULL,
  endpoint_url TEXT NOT NULL,
  api_format TEXT NOT NULL,
  context_size INTEGER,
  gpu_layers INTEGER,
  threads INTEGER,
  process_mode TEXT,
  pid INTEGER,
  service_name TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  last_health_check_at TEXT,
  last_error TEXT,
  FOREIGN KEY (machine_id) REFERENCES machines(id) ON DELETE CASCADE,
  FOREIGN KEY (model_id) REFERENCES models(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS aliases (
  id TEXT PRIMARY KEY,
  alias TEXT NOT NULL UNIQUE,
  target_type TEXT NOT NULL,
  target_deployment_id TEXT,
  target_model_id TEXT,
  routing_policy TEXT NOT NULL DEFAULT 'single_deployment',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (target_deployment_id) REFERENCES deployments(id) ON DELETE SET NULL,
  FOREIGN KEY (target_model_id) REFERENCES models(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS api_keys (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  key_hash TEXT NOT NULL,
  enabled INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL,
  last_used_at TEXT
);

CREATE TABLE IF NOT EXISTS health_checks (
  id TEXT PRIMARY KEY,
  deployment_id TEXT NOT NULL,
  state TEXT NOT NULL,
  message TEXT,
  checked_at TEXT NOT NULL,
  FOREIGN KEY (deployment_id) REFERENCES deployments(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS events (
  id TEXT PRIMARY KEY,
  type TEXT NOT NULL,
  severity TEXT NOT NULL,
  machine_id TEXT,
  deployment_id TEXT,
  message TEXT NOT NULL,
  data_json TEXT,
  created_at TEXT NOT NULL,
  FOREIGN KEY (machine_id) REFERENCES machines(id) ON DELETE SET NULL,
  FOREIGN KEY (deployment_id) REFERENCES deployments(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS settings (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_deployments_machine ON deployments(machine_id);
CREATE INDEX IF NOT EXISTS idx_deployments_status ON deployments(status);
CREATE INDEX IF NOT EXISTS idx_events_created ON events(created_at);
CREATE INDEX IF NOT EXISTS idx_health_checks_deployment ON health_checks(deployment_id);
```

**Step 4: Test database creation**

```bash
go test ./internal/db -v
# Expected: PASS - database initializes correctly
```

**Step 5: Commit**

```bash
git add .
git commit -m "feat: add SQLite database schema and connection"
```

---

## Phase 1B: Core Models & API Foundation

### Task 3: Define Core Data Models

**Files:**
- Create: `internal/models/machine.go`
- Create: `internal/models/model.go`
- Create: `internal/models/deployment.go`

**Step 1: Create machine models**

```go
// internal/models/machine.go
package models

import "time"

type Machine struct {
	ID                  string     `json:"id"`
	Name                string     `json:"name"`
	Host                string     `json:"host"`
	SSHPort             int        `json:"ssh_port"`
	SSHUser             string     `json:"ssh_user"`
	AuthType            string     `json:"auth_type"`
	AuthPrivateKeyPath  string     `json:"auth_private_key_path,omitempty"`
	AuthPasswordEncrypted string   `json:"-"`
	Tags                string     `json:"tags,omitempty"`
	Notes               string     `json:"notes,omitempty"`
	BaseDir             string     `json:"base_dir"`
	ConnectionType      string     `json:"connection_type"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

type MachineFacts struct {
	MachineID          string    `json:"machine_id"`
	OSName             string    `json:"os_name,omitempty"`
	OSVersion          string    `json:"os_version,omitempty"`
	Arch               string    `json:"arch,omitempty"`
	CPUModel           string    `json:"cpu_model,omitempty"`
	CPUCores           int       `json:"cpu_cores,omitempty"`
	RAMBytes           int64     `json:"ram_bytes,omitempty"`
	DiskFreeBytes      int64     `json:"disk_free_bytes,omitempty"`
	GPUVendor          string    `json:"gpu_vendor,omitempty"`
	GPUName            string    `json:"gpu_name,omitempty"`
	GPUVRAMBytes       int64     `json:"gpu_vram_bytes,omitempty"`
	GPUDriverVersion   string    `json:"gpu_driver_version,omitempty"`
	CapabilityState    string    `json:"capability_state"`
	RawJSON            string    `json:"raw_json,omitempty"`
	DetectedAt         time.Time `json:"detected_at"`
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
}
```

**Step 2: Create model and deployment types**

```go
// internal/models/model.go
package models

import "time"

type Model struct {
	ID               string    `json:"id"`
	DisplayName      string    `json:"display_name"`
	Family           string    `json:"family,omitempty"`
	ParameterSize    string    `json:"parameter_size,omitempty"`
	Format           string    `json:"format"`
	Quantization     string    `json:"quantization,omitempty"`
	ContextLength    int       `json:"context_length,omitempty"`
	SourceURL        string    `json:"source_url,omitempty"`
	Filename         string    `json:"filename"`
	SHA256           string    `json:"sha256,omitempty"`
	EstimatedRAMBytes int64    `json:"estimated_ram_bytes,omitempty"`
	EstimatedVRAMBytes int64   `json:"estimated_vram_bytes,omitempty"`
	Tags             string    `json:"tags,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}
```

```go
// internal/models/deployment.go
package models

import "time"

type Deployment struct {
	ID                string     `json:"id"`
	Name              string     `json:"name"`
	MachineID         string     `json:"machine_id"`
	ModelID           string     `json:"model_id"`
	Runtime           string     `json:"runtime"`
	Backend           string     `json:"backend"`
	Status            string     `json:"status"`
	Host              string     `json:"host"`
	Port              int        `json:"port"`
	EndpointURL       string     `json:"endpoint_url"`
	APIFormat         string     `json:"api_format"`
	ContextSize       int        `json:"context_size,omitempty"`
	GPULayers         int        `json:"gpu_layers,omitempty"`
	Threads           int        `json:"threads,omitempty"`
	ProcessMode       string     `json:"process_mode,omitempty"`
	PID               int        `json:"pid,omitempty"`
	ServiceName       string     `json:"service_name,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	LastHealthCheckAt *time.Time `json:"last_health_check_at,omitempty"`
	LastError         string     `json:"last_error,omitempty"`
}

type CreateDeploymentRequest struct {
	Name        string `json:"name"`
	MachineID   string `json:"machine_id"`
	ModelID     string `json:"model_id"`
	Runtime     string `json:"runtime"`
	Backend     string `json:"backend"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	ContextSize int    `json:"context_size,omitempty"`
	GPULayers   int    `json:"gpu_layers,omitempty"`
	Threads     int    `json:"threads,omitempty"`
}
```

**Step 3: Commit**

```bash
git add .
git commit -m "feat: define core data models"
```

---

### Task 4: Machine Repository & API

**Files:**
- Create: `internal/repository/machine.go`
- Create: `internal/api/machines.go`
- Create: `internal/api/router.go`

**Step 1: Create machine repository**

```go
// internal/repository/machine.go
package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/modelfleet/modelfleet/internal/models"
)

type MachineRepository struct {
	db *sql.DB
}

func NewMachineRepository(db *sql.DB) *MachineRepository {
	return &MachineRepository{db: db}
}

func (r *MachineRepository) Create(m *models.Machine) error {
	m.CreatedAt = time.Now().UTC()
	m.UpdatedAt = m.CreatedAt

	_, err := r.db.Exec(`
		INSERT INTO machines (id, name, host, ssh_port, ssh_user, auth_type,
		auth_private_key_path, auth_password_encrypted, tags, notes, base_dir,
		connection_type, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, m.ID, m.Name, m.Host, m.SSHPort, m.SSHUser, m.AuthType,
		m.AuthPrivateKeyPath, m.AuthPasswordEncrypted, m.Tags, m.Notes, m.BaseDir,
		m.ConnectionType, m.CreatedAt.Format(time.RFC3339), m.UpdatedAt.Format(time.RFC3339))
	return err
}

func (r *MachineRepository) GetAll() ([]models.Machine, error) {
	rows, err := r.db.Query(`
		SELECT id, name, host, ssh_port, ssh_user, auth_type,
		auth_private_key_path, tags, notes, base_dir, connection_type, created_at, updated_at
		FROM machines ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var machines []models.Machine
	for rows.Next() {
		var m models.Machine
		var createdAt, updatedAt string
		err := rows.Scan(&m.ID, &m.Name, &m.Host, &m.SSHPort, &m.SSHUser, &m.AuthType,
			&m.AuthPrivateKeyPath, &m.Tags, &m.Notes, &m.BaseDir, &m.ConnectionType,
			&createdAt, &updatedAt)
		if err != nil {
			return nil, err
		}
		m.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		m.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		machines = append(machines, m)
	}
	return machines, nil
}

func (r *MachineRepository) GetByID(id string) (*models.Machine, error) {
	var m models.Machine
	var createdAt, updatedAt string
	err := r.db.QueryRow(`
		SELECT id, name, host, ssh_port, ssh_user, auth_type,
		auth_private_key_path, tags, notes, base_dir, connection_type, created_at, updated_at
		FROM machines WHERE id = ?
	`, id).Scan(&m.ID, &m.Name, &m.Host, &m.SSHPort, &m.SSHUser, &m.AuthType,
		&m.AuthPrivateKeyPath, &m.Tags, &m.Notes, &m.BaseDir, &m.ConnectionType,
		&createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	m.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	m.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &m, nil
}
```

**Step 2: Create machines API handler**

```go
// internal/api/machines.go
package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/modelfleet/modelfleet/internal/models"
	"github.com/modelfleet/modelfleet/internal/repository"
)

type MachineHandler struct {
	repo *repository.MachineRepository
}

func NewMachineHandler(repo *repository.MachineRepository) *MachineHandler {
	return &MachineHandler{repo: repo}
}

func (h *MachineHandler) List(w http.ResponseWriter, r *http.Request) {
	machines, err := h.repo.GetAll()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if machines == nil {
		machines = []models.Machine{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(machines)
}

func (h *MachineHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.AddMachineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.Host == "" || req.SSHUser == "" {
		http.Error(w, "name, host, and ssh_user are required", http.StatusBadRequest)
		return
	}

	if req.SSHPort == 0 {
		req.SSHPort = 22
	}

	machine := &models.Machine{
		ID:             uuid.New().String(),
		Name:           req.Name,
		Host:           req.Host,
		SSHPort:        req.SSHPort,
		SSHUser:        req.SSHUser,
		AuthType:       req.AuthType,
		AuthPrivateKeyPath: req.AuthPrivateKeyPath,
		Tags:           req.Tags,
		Notes:          req.Notes,
		BaseDir:        req.BaseDir,
		ConnectionType: "ssh",
	}

	if machine.BaseDir == "" {
		machine.BaseDir = "~/modelfleet"
	}

	if err := h.repo.Create(machine); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(machine)
}
```

**Step 3: Create API router**

```go
// internal/api/router.go
package api

import (
	"net/http"
	"github.com/modelfleet/modelfleet/internal/repository"
)

func NewRouter(machineRepo *repository.MachineRepository) http.Handler {
	mux := http.NewServeMux()

	machineHandler := NewMachineHandler(machineRepo)

	mux.HandleFunc("/api/machines", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			machineHandler.List(w, r)
		case http.MethodPost:
			machineHandler.Create(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	return mux
}
```

**Step 4: Update main.go to use router**

```go
// Update main.go imports and main function
import (
	"log"
	"net/http"
	"os"

	"github.com/modelfleet/modelfleet/internal/api"
	"github.com/modelfleet/modelfleet/internal/db"
	"github.com/modelfleet/modelfleet/internal/repository"
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

	database, err := db.New(dataDir)
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer database.Close()

	machineRepo := repository.NewMachineRepository(database.DB)

	router := api.NewRouter(machineRepo)

	log.Printf("ModelFleet starting on port %s", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatal(err)
	}
}
```

**Step 5: Add uuid dependency and test**

```bash
go get github.com/google/uuid
go run main.go &
curl -X POST http://localhost:3456/api/machines \
  -H "Content-Type: application/json" \
  -d '{"name":"test-machine","host":"192.168.1.100","ssh_user":"admin","auth_type":"key","auth_private_key_path":"/home/user/.ssh/id_rsa"}'
# Expected: JSON with created machine
curl http://localhost:3456/api/machines
# Expected: Array with test machine
kill %1
```

**Step 6: Commit**

```bash
git add .
git commit -m "feat: add machine repository and REST API endpoints"
```

---

## Phase 1C: SSH Management

### Task 5: SSH Client & Connection Testing

**Files:**
- Create: `internal/ssh/client.go`
- Create: `internal/ssh/test.go`
- Modify: `internal/api/machines.go`

**Step 1: Add SSH dependency**

```bash
go get golang.org/x/crypto/ssh
```

**Step 2: Create SSH client**

```go
// internal/ssh/client.go
package ssh

import (
	"fmt"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

type Client struct {
	config *ssh.ClientConfig
}

func NewClient(user, authType, privateKeyPath, password string) (*Client, error) {
	config := &ssh.ClientConfig{
		User:            user,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: implement known_hosts verification
		Timeout:         10 * time.Second,
	}

	switch authType {
	case "key":
		if privateKeyPath == "" {
			privateKeyPath = os.ExpandEnv("$HOME/.ssh/id_rsa")
		}
		key, err := os.ReadFile(privateKeyPath)
		if err != nil {
			return nil, fmt.Errorf("read private key: %w", err)
		}
		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, fmt.Errorf("parse private key: %w", err)
		}
		config.Auth = []ssh.AuthMethod{ssh.PublicKeys(signer)}
	case "password":
		config.Auth = []ssh.AuthMethod{ssh.Password(password)}
	default:
		return nil, fmt.Errorf("unsupported auth type: %s", authType)
	}

	return &Client{config: config}, nil
}

func (c *Client) Connect(host string, port int) (*ssh.Client, error) {
	addr := fmt.Sprintf("%s:%d", host, port)
	return ssh.Dial("tcp", addr, c.config)
}

func (c *Client) TestConnection(host string, port int) error {
	client, err := c.Connect(host, port)
	if err != nil {
		return fmt.Errorf("SSH connection failed: %w", err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("create SSH session: %w", err)
	}
	defer session.Close()

	output, err := session.Output("echo MODEL_FLEET_SSH_OK")
	if err != nil {
		return fmt.Errorf("execute test command: %w", err)
	}

	if string(output) != "MODEL_FLEET_SSH_OK\n" {
		return fmt.Errorf("unexpected response: %s", string(output))
	}

	return nil
}
```

**Step 3: Add test endpoint**

```go
// Add to internal/api/machines.go

func (h *MachineHandler) TestSSH(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Machine ID required", http.StatusBadRequest)
		return
	}

	machine, err := h.repo.GetByID(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if machine == nil {
		http.Error(w, "Machine not found", http.StatusNotFound)
		return
	}

	client, err := ssh.NewClient(machine.SSHUser, machine.AuthType, machine.AuthPrivateKeyPath, "")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := client.TestConnection(machine.Host, machine.SSHPort); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "SSH connection successful",
	})
}
```

**Step 4: Update router**

```go
// Add to internal/api/router.go
mux.HandleFunc("/api/machines/{id}/test-ssh", func(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		machineHandler.TestSSH(w, r)
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
})
```

**Step 5: Commit**

```bash
git add .
git commit -m "feat: add SSH client and connection testing"
```

---

## Phase 1D: Hardware Detection

### Task 6: Detection System

**Files:**
- Create: `internal/detection/detector.go`
- Create: `internal/detection/commands.go`
- Modify: `internal/api/machines.go`

**Step 1: Create detection types**

```go
// internal/detection/detector.go
package detection

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"

	"github.com/modelfleet/modelfleet/internal/ssh"
)

type DetectionResult struct {
	OSName           string            `json:"os_name"`
	OSVersion        string            `json:"os_version"`
	Arch             string            `json:"arch"`
	CPUModel         string            `json:"cpu_model"`
	CPUCores         int               `json:"cpu_cores"`
	RAMBytes         int64             `json:"ram_bytes"`
	DiskFreeBytes    int64             `json:"disk_free_bytes"`
	GPUVendor        string            `json:"gpu_vendor"`
	GPUName          string            `json:"gpu_name"`
	GPUVRAMBytes     int64             `json:"gpu_vram_bytes"`
	GPUDriverVersion string            `json:"gpu_driver_version"`
	CapabilityState  string            `json:"capability_state"`
	HasNVIDIA        bool              `json:"has_nvidia"`
	HasAMD           bool              `json:"has_amd"`
	HasVulkan        bool              `json:"has_vulkan"`
	RawOutputs       map[string]string `json:"raw_outputs"`
}

type Detector struct {
	client *ssh.Client
}

func NewDetector(client *ssh.Client) *Detector {
	return &Detector{client: client}
}

func (d *Detector) Detect(host string, port int) (*DetectionResult, error) {
	result := &DetectionResult{
		RawOutputs: make(map[string]string),
	}

	// Connect to remote
	sshClient, err := d.client.Connect(host, port)
	if err != nil {
		return nil, fmt.Errorf("SSH connect: %w", err)
	}
	defer sshClient.Close()

	// Detect OS and architecture
	if err := d.detectOS(sshClient, result); err != nil {
		return nil, err
	}

	// Detect CPU
	if err := d.detectCPU(sshClient, result); err != nil {
		return nil, err
	}

	// Detect memory
	if err := d.detectMemory(sshClient, result); err != nil {
		return nil, err
	}

	// Detect disk
	if err := d.detectDisk(sshClient, result); err != nil {
		return nil, err
	}

	// Detect GPU capabilities
	if err := d.detectGPU(sshClient, result); err != nil {
		return nil, err
	}

	// Determine capability state
	result.CapabilityState = d.determineCapabilityState(result)

	return result, nil
}

func (d *Detector) detectOS(client *ssh.Client, result *DetectionResult) error {
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	output, err := session.Output("cat /etc/os-release")
	if err != nil {
		return fmt.Errorf("os-release: %w", err)
	}

	result.RawOutputs["os_release"] = string(output)

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "NAME=") {
			result.OSName = strings.Trim(strings.TrimPrefix(line, "NAME="), `"`)
		}
		if strings.HasPrefix(line, "VERSION_ID=") {
			result.OSVersion = strings.Trim(strings.TrimPrefix(line, "VERSION_ID="), `"`)
		}
	}

	// Get architecture
	session2, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session2.Close()

	archOutput, err := session2.Output("uname -m")
	if err == nil {
		result.Arch = strings.TrimSpace(string(archOutput))
	}

	return nil
}

func (d *Detector) detectCPU(client *ssh.Client, result *DetectionResult) error {
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	output, err := session.Output("lscpu")
	if err != nil {
		return fmt.Errorf("lscpu: %w", err)
	}

	result.RawOutputs["lscpu"] = string(output)

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "Model name:") {
			result.CPUModel = strings.TrimSpace(strings.SplitN(line, ":", 2)[1])
		}
		if strings.Contains(line, "CPU(s):") {
			coresStr := strings.TrimSpace(strings.SplitN(line, ":", 2)[1])
			result.CPUCores, _ = strconv.Atoi(coresStr)
		}
	}

	return nil
}

func (d *Detector) detectMemory(client *ssh.Client, result *DetectionResult) error {
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	output, err := session.Output("free -b | awk 'NR==2{print $2}'")
	if err != nil {
		return fmt.Errorf("free: %w", err)
	}

	result.RAMBytes, _ = strconv.ParseInt(strings.TrimSpace(string(output)), 10, 64)
	return nil
}

func (d *Detector) detectDisk(client *ssh.Client, result *DetectionResult) error {
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	output, err := session.Output("df -B1 ~ | awk 'NR==2{print $4}'")
	if err != nil {
		return fmt.Errorf("df: %w", err)
	}

	result.DiskFreeBytes, _ = strconv.ParseInt(strings.TrimSpace(string(output)), 10, 64)
	return nil
}

func (d *Detector) detectGPU(client *ssh.Client, result *DetectionResult) error {
	// Check NVIDIA
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	_, err = session.Output("command -v nvidia-smi")
	result.HasNVIDIA = (err == nil)

	if result.HasNVIDIA {
		session2, err := client.NewSession()
		if err != nil {
			return err
		}
		defer session2.Close()

		output, err := session2.Output("nvidia-smi --query-gpu=name,memory.total,driver_version --format=csv,noheader")
		if err == nil {
			result.RawOutputs["nvidia_smi"] = string(output)
			parts := strings.Split(strings.TrimSpace(string(output)), ", ")
			if len(parts) >= 3 {
				result.GPUVendor = "NVIDIA"
				result.GPUName = parts[0]
				result.GPUDriverVersion = parts[2]
				// Parse memory like "24576 MiB"
				memStr := strings.TrimSpace(strings.TrimSuffix(parts[1], " MiB"))
				memMiB, _ := strconv.ParseInt(memStr, 10, 64)
				result.GPUVRAMBytes = memMiB * 1024 * 1024
			}
		}
	}

	// Check AMD ROCm
	session3, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session3.Close()

	_, err = session3.Output("command -v rocminfo")
	result.HasAMD = (err == nil)

	// Check Vulkan
	session4, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session4.Close()

	_, err = session4.Output("command -v vulkaninfo")
	result.HasVulkan = (err == nil)

	return nil
}

func (d *Detector) determineCapabilityState(result *DetectionResult) string {
	if result.HasNVIDIA && result.GPUName != "" {
		return "nvidia_ready"
	}
	if result.HasAMD {
		return "amd_rocm_ready"
	}
	if result.HasVulkan {
		return "vulkan_ready"
	}
	return "cpu_only"
}
```

**Step 2: Add detection endpoint**

```go
// Add to internal/api/machines.go

func (h *MachineHandler) Detect(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Machine ID required", http.StatusBadRequest)
		return
	}

	machine, err := h.repo.GetByID(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if machine == nil {
		http.Error(w, "Machine not found", http.StatusNotFound)
		return
	}

	client, err := ssh.NewClient(machine.SSHUser, machine.AuthType, machine.AuthPrivateKeyPath, "")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	detector := detection.NewDetector(client)
	result, err := detector.Detect(machine.Host, machine.SSHPort)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
```

**Step 3: Update router**

```go
// Add to internal/api/router.go
mux.HandleFunc("/api/machines/{id}/detect", func(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		machineHandler.Detect(w, r)
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
})
```

**Step 4: Commit**

```bash
git add .
git commit -m "feat: add hardware detection system"
```

---

## Phase 1E: Model Management

### Task 7: Model CRUD API

**Files:**
- Create: `internal/repository/model.go`
- Create: `internal/api/models.go`
- Modify: `internal/api/router.go`

**Step 1: Create model repository**

```go
// internal/repository/model.go
package repository

import (
	"database/sql"
	"time"

	"github.com/modelfleet/modelfleet/internal/models"
)

type ModelRepository struct {
	db *sql.DB
}

func NewModelRepository(db *sql.DB) *ModelRepository {
	return &ModelRepository{db: db}
}

func (r *ModelRepository) Create(m *models.Model) error {
	m.CreatedAt = time.Now().UTC()
	m.UpdatedAt = m.CreatedAt

	_, err := r.db.Exec(`
		INSERT INTO models (id, display_name, family, parameter_size, format,
		quantization, context_length, source_url, filename, sha256,
		estimated_ram_bytes, estimated_vram_bytes, tags, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, m.ID, m.DisplayName, m.Family, m.ParameterSize, m.Format,
		m.Quantization, m.ContextLength, m.SourceURL, m.Filename, m.SHA256,
		m.EstimatedRAMBytes, m.EstimatedVRAMBytes, m.Tags,
		m.CreatedAt.Format(time.RFC3339), m.UpdatedAt.Format(time.RFC3339))
	return err
}

func (r *ModelRepository) GetAll() ([]models.Model, error) {
	rows, err := r.db.Query(`
		SELECT id, display_name, family, parameter_size, format, quantization,
		context_length, source_url, filename, sha256, estimated_ram_bytes,
		estimated_vram_bytes, tags, created_at, updated_at
		FROM models ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var models_ []models.Model
	for rows.Next() {
		var m models.Model
		var createdAt, updatedAt string
		err := rows.Scan(&m.ID, &m.DisplayName, &m.Family, &m.ParameterSize, &m.Format,
			&m.Quantization, &m.ContextLength, &m.SourceURL, &m.Filename, &m.SHA256,
			&m.EstimatedRAMBytes, &m.EstimatedVRAMBytes, &m.Tags,
			&createdAt, &updatedAt)
		if err != nil {
			return nil, err
		}
		m.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		m.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		models_ = append(models_, m)
	}
	return models_, nil
}
```

**Step 2: Create models API handler**

```go
// internal/api/models.go
package api

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/modelfleet/modelfleet/internal/models"
	"github.com/modelfleet/modelfleet/internal/repository"
)

type ModelHandler struct {
	repo *repository.ModelRepository
}

func NewModelHandler(repo *repository.ModelRepository) *ModelHandler {
	return &ModelHandler{repo: repo}
}

func (h *ModelHandler) List(w http.ResponseWriter, r *http.Request) {
	models, err := h.repo.GetAll()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if models == nil {
		models = []models.Model{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models)
}

func (h *ModelHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.Model
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.DisplayName == "" || req.Filename == "" {
		http.Error(w, "display_name and filename are required", http.StatusBadRequest)
		return
	}

	req.ID = uuid.New().String()
	if req.Format == "" {
		req.Format = "GGUF"
	}

	if err := h.repo.Create(&req); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(req)
}
```

**Step 3: Update router and main.go**

```go
// Update internal/api/router.go
func NewRouter(machineRepo *repository.MachineRepository, modelRepo *repository.ModelRepository) http.Handler {
	mux := http.NewServeMux()

	machineHandler := NewMachineHandler(machineRepo)
	modelHandler := NewModelHandler(modelRepo)

	// ... existing machine routes ...

	mux.HandleFunc("/api/models", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			modelHandler.List(w, r)
		case http.MethodPost:
			modelHandler.Create(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	return mux
}
```

**Step 4: Update main.go**

```go
// In main.go
modelRepo := repository.NewModelRepository(database.DB)
router := api.NewRouter(machineRepo, modelRepo)
```

**Step 5: Commit**

```bash
git add .
git commit -m "feat: add model CRUD API endpoints"
```

---

## Phase 1F: Deployment System

### Task 8: Deployment Planning & Execution

**Files:**
- Create: `internal/repository/deployment.go`
- Create: `internal/deployment/planner.go`
- Create: `internal/deployment/executor.go`
- Create: `internal/api/deployments.go`
- Modify: `internal/api/router.go`

**Step 1: Create deployment repository**

```go
// internal/repository/deployment.go
package repository

import (
	"database/sql"
	"time"

	"github.com/modelfleet/modelfleet/internal/models"
)

type DeploymentRepository struct {
	db *sql.DB
}

func NewDeploymentRepository(db *sql.DB) *DeploymentRepository {
	return &DeploymentRepository{db: db}
}

func (r *DeploymentRepository) Create(d *models.Deployment) error {
	d.CreatedAt = time.Now().UTC()
	d.UpdatedAt = d.CreatedAt

	_, err := r.db.Exec(`
		INSERT INTO deployments (id, name, machine_id, model_id, runtime, backend,
		status, host, port, endpoint_url, api_format, context_size, gpu_layers,
		threads, process_mode, pid, service_name, created_at, updated_at,
		last_health_check_at, last_error)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, d.ID, d.Name, d.MachineID, d.ModelID, d.Runtime, d.Backend,
		d.Status, d.Host, d.Port, d.EndpointURL, d.APIFormat, d.ContextSize,
		d.GPULayers, d.Threads, d.ProcessMode, d.PID, d.ServiceName,
		d.CreatedAt.Format(time.RFC3339), d.UpdatedAt.Format(time.RFC3339),
		d.LastHealthCheckAt, d.LastError)
	return err
}

func (r *DeploymentRepository) GetAll() ([]models.Deployment, error) {
	rows, err := r.db.Query(`
		SELECT id, name, machine_id, model_id, runtime, backend, status,
		host, port, endpoint_url, api_format, context_size, gpu_layers,
		threads, process_mode, pid, service_name, created_at, updated_at,
		last_health_check_at, last_error
		FROM deployments ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deployments []models.Deployment
	for rows.Next() {
		var d models.Deployment
		var createdAt, updatedAt, lastHealthCheck string
		err := rows.Scan(&d.ID, &d.Name, &d.MachineID, &d.ModelID, &d.Runtime, &d.Backend,
			&d.Status, &d.Host, &d.Port, &d.EndpointURL, &d.APIFormat, &d.ContextSize,
			&d.GPULayers, &d.Threads, &d.ProcessMode, &d.PID, &d.ServiceName,
			&createdAt, &updatedAt, &lastHealthCheck, &d.LastError)
		if err != nil {
			return nil, err
		}
		d.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		d.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		if lastHealthCheck != "" {
			t, _ := time.Parse(time.RFC3339, lastHealthCheck)
			d.LastHealthCheckAt = &t
		}
		deployments = append(deployments, d)
	}
	return deployments, nil
}

func (r *DeploymentRepository) GetByID(id string) (*models.Deployment, error) {
	var d models.Deployment
	var createdAt, updatedAt, lastHealthCheck string
	err := r.db.QueryRow(`
		SELECT id, name, machine_id, model_id, runtime, backend, status,
		host, port, endpoint_url, api_format, context_size, gpu_layers,
		threads, process_mode, pid, service_name, created_at, updated_at,
		last_health_check_at, last_error
		FROM deployments WHERE id = ?
	`, id).Scan(&d.ID, &d.Name, &d.MachineID, &d.ModelID, &d.Runtime, &d.Backend,
		&d.Status, &d.Host, &d.Port, &d.EndpointURL, &d.APIFormat, &d.ContextSize,
		&d.GPULayers, &d.Threads, &d.ProcessMode, &d.PID, &d.ServiceName,
		&createdAt, &updatedAt, &lastHealthCheck, &d.LastError)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	d.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	d.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	if lastHealthCheck != "" {
		t, _ := time.Parse(time.RFC3339, lastHealthCheck)
		d.LastHealthCheckAt = &t
	}
	return &d, nil
}

func (r *DeploymentRepository) UpdateStatus(id string, status string) error {
	_, err := r.db.Exec(`
		UPDATE deployments SET status = ?, updated_at = ? WHERE id = ?
	`, status, time.Now().UTC().Format(time.RFC3339), id)
	return err
}
```

**Step 2: Create deployment planner**

```go
// internal/deployment/planner.go
package deployment

import (
	"fmt"

	"github.com/modelfleet/modelfleet/internal/models"
)

type Plan struct {
	Machine      *models.Machine      `json:"machine"`
	MachineFacts *models.MachineFacts `json:"machine_facts,omitempty"`
	Model        *models.Model        `json:"model"`
	Runtime      string               `json:"runtime"`
	Backend      string               `json:"backend"`
	Actions      []string             `json:"actions"`
	RequiresRoot bool                 `json:"requires_root"`
	RequiresReboot bool               `json:"requires_reboot"`
}

type Planner struct {
	machineRepo     *repository.MachineRepository
	modelRepo       *repository.ModelRepository
	machineFactsRepo *repository.MachineFactsRepository
}

func NewPlanner(machineRepo *repository.MachineRepository, modelRepo *repository.ModelRepository) *Planner {
	return &Planner{
		machineRepo: machineRepo,
		modelRepo:   modelRepo,
	}
}

func (p *Planner) CreatePlan(machineID, modelID string) (*Plan, error) {
	machine, err := p.machineRepo.GetByID(machineID)
	if err != nil {
		return nil, fmt.Errorf("get machine: %w", err)
	}
	if machine == nil {
		return nil, fmt.Errorf("machine not found")
	}

	model, err := p.modelRepo.GetByID(modelID)
	if err != nil {
		return nil, fmt.Errorf("get model: %w", err)
	}
	if model == nil {
		return nil, fmt.Errorf("model not found")
	}

	plan := &Plan{
		Machine: machine,
		Model:   model,
		Runtime: "llama.cpp",
		Actions: []string{
			fmt.Sprintf("Create %s directories on remote machine", machine.BaseDir),
			"Download llama-server binary",
			"Verify runtime checksum",
			fmt.Sprintf("Download model: %s", model.Filename),
			"Create deployment configuration",
			"Start llama-server process",
			"Health check endpoint",
			"Register deployment in ModelFleet gateway",
		},
		RequiresRoot:   false,
		RequiresReboot: false,
	}

	// Determine backend based on capability
	// For now, default to CPU
	plan.Backend = "cpu"

	return plan, nil
}
```

**Step 3: Create deployment executor**

```go
// internal/deployment/executor.go
package deployment

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/modelfleet/modelfleet/internal/models"
	"github.com/modelfleet/modelfleet/internal/ssh"
)

type Executor struct {
	client *ssh.Client
}

func NewExecutor(client *ssh.Client) *Executor {
	return &Executor{client: client}
}

func (e *Executor) ExecutePlan(machine *models.Machine, deployment *models.Deployment) error {
	// Connect to remote
	sshClient, err := e.client.Connect(machine.Host, machine.SSHPort)
	if err != nil {
		return fmt.Errorf("SSH connect: %w", err)
	}
	defer sshClient.Close()

	// Create directories
	baseDir := strings.Replace(machine.BaseDir, "~", "$HOME", 1)
	dirs := []string{"bin", "models", "runtimes", "deployments", "logs", "tmp"}
	for _, dir := range dirs {
		session, err := sshClient.NewSession()
		if err != nil {
			return fmt.Errorf("create session: %w", err)
		}
		cmd := fmt.Sprintf("mkdir -p %s", filepath.Join(baseDir, dir))
		if err := session.Run(cmd); err != nil {
			session.Close()
			return fmt.Errorf("create directory %s: %w", dir, err)
		}
		session.Close()
	}

	// TODO: Download runtime, download model, start process
	// For MVP, we'll create a basic deployment config and start process

	return nil
}
```

**Step 4: Create deployments API handler**

```go
// internal/api/deployments.go
package api

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/modelfleet/modelfleet/internal/deployment"
	"github.com/modelfleet/modelfleet/internal/models"
	"github.com/modelfleet/modelfleet/internal/repository"
)

type DeploymentHandler struct {
	repo    *repository.DeploymentRepository
	planner *deployment.Planner
}

func NewDeploymentHandler(repo *repository.DeploymentRepository, planner *deployment.Planner) *DeploymentHandler {
	return &DeploymentHandler{repo: repo, planner: planner}
}

func (h *DeploymentHandler) List(w http.ResponseWriter, r *http.Request) {
	deployments, err := h.repo.GetAll()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if deployments == nil {
		deployments = []models.Deployment{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(deployments)
}

func (h *DeploymentHandler) CreatePlan(w http.ResponseWriter, r *http.Request) {
	var req struct {
		MachineID string `json:"machine_id"`
		ModelID   string `json:"model_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	plan, err := h.planner.CreatePlan(req.MachineID, req.ModelID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(plan)
}

func (h *DeploymentHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.CreateDeploymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.MachineID == "" || req.ModelID == "" {
		http.Error(w, "name, machine_id, and model_id are required", http.StatusBadRequest)
		return
	}

	deployment := &models.Deployment{
		ID:          uuid.New().String(),
		Name:        req.Name,
		MachineID:   req.MachineID,
		ModelID:     req.ModelID,
		Runtime:     req.Runtime,
		Backend:     req.Backend,
		Status:      "planned",
		Host:        req.Host,
		Port:        req.Port,
		EndpointURL: fmt.Sprintf("http://%s:%d", req.Host, req.Port),
		APIFormat:   "openai",
		ContextSize: req.ContextSize,
		GPULayers:   req.GPULayers,
		Threads:     req.Threads,
	}

	if deployment.Host == "" {
		deployment.Host = "0.0.0.0"
	}
	if deployment.Port == 0 {
		deployment.Port = 8081
	}
	if deployment.Runtime == "" {
		deployment.Runtime = "llama.cpp"
	}
	if deployment.Backend == "" {
		deployment.Backend = "cpu"
	}

	if err := h.repo.Create(deployment); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(deployment)
}
```

**Step 5: Update router and main.go**

```go
// Update router
func NewRouter(machineRepo *repository.MachineRepository, modelRepo *repository.ModelRepository, deploymentRepo *repository.DeploymentRepository) http.Handler {
	mux := http.NewServeMux()

	machineHandler := NewMachineHandler(machineRepo)
	modelHandler := NewModelHandler(modelRepo)
	planner := deployment.NewPlanner(machineRepo, modelRepo)
	deploymentHandler := NewDeploymentHandler(deploymentRepo, planner)

	// ... existing routes ...

	mux.HandleFunc("/api/deployments", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			deploymentHandler.List(w, r)
		case http.MethodPost:
			deploymentHandler.Create(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/deployments/plan", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			deploymentHandler.CreatePlan(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	return mux
}
```

**Step 6: Commit**

```bash
git add .
git commit -m "feat: add deployment planning and execution framework"
```

---

## Phase 1G: OpenAI-Compatible Gateway

### Task 9: Gateway API

**Files:**
- Create: `internal/gateway/gateway.go`
- Create: `internal/gateway/router.go`
- Modify: `internal/api/router.go`

**Step 1: Create gateway handler**

```go
// internal/gateway/gateway.go
package gateway

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/modelfleet/modelfleet/internal/models"
	"github.com/modelfleet/modelfleet/internal/repository"
)

type Gateway struct {
	deploymentRepo *repository.DeploymentRepository
	aliasRepo      *repository.AliasRepository
}

func NewGateway(deploymentRepo *repository.DeploymentRepository, aliasRepo *repository.AliasRepository) *Gateway {
	return &Gateway{
		deploymentRepo: deploymentRepo,
		aliasRepo:      aliasRepo,
	}
}

func (g *Gateway) ListModels(w http.ResponseWriter, r *http.Request) {
	deployments, err := g.deploymentRepo.GetAll()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var models []map[string]interface{}
	for _, d := range deployments {
		if d.Status == "running" {
			models = append(models, map[string]interface{}{
				"id":       fmt.Sprintf("local/%s", d.Name),
				"object":   "model",
				"owned_by": "modelfleet",
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"object": "list",
		"data":   models,
	})
}

func (g *Gateway) ChatCompletions(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Model    string                   `json:"model"`
		Messages []map[string]interface{} `json:"messages"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// TODO: Route to appropriate deployment
	// For MVP, return a simple response

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":      "chatcmpl-test",
		"object":  "chat.completion",
		"created": 1234567890,
		"model":   req.Model,
		"choices": []map[string]interface{}{
			{
				"index": 0,
				"message": map[string]interface{}{
					"role":    "assistant",
					"content": "This is a placeholder response. The actual model routing will be implemented in a future update.",
				},
				"finish_reason": "stop",
			},
		},
	})
}
```

**Step 2: Update router to include gateway routes**

```go
// Add to internal/api/router.go
func NewRouter(machineRepo *repository.MachineRepository, modelRepo *repository.ModelRepository, deploymentRepo *repository.DeploymentRepository, aliasRepo *repository.AliasRepository) http.Handler {
	mux := http.NewServeMux()

	// ... internal API routes ...

	// Gateway routes
	gatewayHandler := gateway.NewGateway(deploymentRepo, aliasRepo)
	mux.HandleFunc("/v1/models", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			gatewayHandler.ListModels(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			gatewayHandler.ChatCompletions(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	return mux
}
```

**Step 3: Commit**

```bash
git add .
git commit -m "feat: add OpenAI-compatible gateway endpoints"
```

---

## Phase 1H: Frontend Foundation

### Task 10: SvelteKit Frontend Setup

**Files:**
- Create: `frontend/package.json`
- Create: `frontend/svelte.config.js`
- Create: `frontend/src/app.html`
- Create: `frontend/src/routes/+page.svelte`

**Step 1: Initialize SvelteKit project**

```bash
cd /Users/rliu/Projects/ModelFleet/frontend
npm create svelte@latest . -- --template skeleton --types typescript
npm install
```

**Step 2: Create basic layout and pages**

```svelte
<!-- frontend/src/routes/+layout.svelte -->
<script>
	import '../app.css';
</script>

<nav>
	<a href="/">Dashboard</a>
	<a href="/machines">Machines</a>
	<a href="/models">Models</a>
	<a href="/deployments">Deployments</a>
	<a href="/gateway">Gateway</a>
</nav>

<main>
	<slot />
</main>

<style>
	nav {
		display: flex;
		gap: 1rem;
		padding: 1rem;
		background: #1a1a2e;
	}

	nav a {
		color: #eee;
		text-decoration: none;
	}

	nav a:hover {
		color: #fff;
	}

	main {
		padding: 2rem;
		max-width: 1200px;
		margin: 0 auto;
	}
</style>
```

```svelte
<!-- frontend/src/routes/+page.svelte -->
<script>
	import { onMount } from 'svelte';

	let stats = {
		machines: 0,
		healthy: 0,
		unhealthy: 0
	};

	onMount(async () => {
		// TODO: Fetch actual stats from API
	});
</script>

<h1>ModelFleet Dashboard</h1>

<div class="stats">
	<div class="stat">
		<h3>Machines</h3>
		<p>{stats.machines}</p>
	</div>
	<div class="stat">
		<h3>Healthy</h3>
		<p>{stats.healthy}</p>
	</div>
	<div class="stat">
		<h3>Unhealthy</h3>
		<p>{stats.unhealthy}</p>
	</div>
</div>

<div class="quick-start">
	<h2>Quick Start</h2>
	<ol>
		<li><a href="/machines">Add a machine</a></li>
		<li><a href="/models">Add a model</a></li>
		<li><a href="/deployments">Deploy the model</a></li>
		<li><a href="/gateway">Use the API</a></li>
	</ol>
</div>

<style>
	.stats {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: 1rem;
		margin: 2rem 0;
	}

	.stat {
		background: #f5f5f5;
		padding: 1.5rem;
		border-radius: 8px;
		text-align: center;
	}

	.stat h3 {
		margin: 0 0 0.5rem 0;
		color: #666;
	}

	.stat p {
		margin: 0;
		font-size: 2rem;
		font-weight: bold;
	}

	.quick-start {
		margin-top: 2rem;
	}

	.quick-start ol {
		font-size: 1.1rem;
	}

	.quick-start li {
		margin: 0.5rem 0;
	}
</style>
```

**Step 3: Build frontend**

```bash
npm run build
```

**Step 4: Commit**

```bash
cd /Users/rliu/Projects/ModelFleet
git add .
git commit -m "feat: add SvelteKit frontend foundation"
```

---

## Phase 1I: Frontend Pages

### Task 11: Machines, Models, Deployments Pages

**Files:**
- Create: `frontend/src/routes/machines/+page.svelte`
- Create: `frontend/src/routes/machines/+page.ts`
- Create: `frontend/src/routes/models/+page.svelte`
- Create: `frontend/src/routes/deployments/+page.svelte`

**Step 1: Create machines page**

```svelte
<!-- frontend/src/routes/machines/+page.svelte -->
<script>
	import { onMount } from 'svelte';

	let machines = [];
	let loading = true;
	let error = null;

	let newMachine = {
		name: '',
		host: '',
		ssh_port: 22,
		ssh_user: '',
		auth_type: 'key',
		auth_private_key_path: ''
	};

	onMount(async () => {
		await loadMachines();
	});

	async function loadMachines() {
		try {
			const res = await fetch('/api/machines');
			machines = await res.json();
		} catch (e) {
			error = e.message;
		} finally {
			loading = false;
		}
	}

	async function addMachine() {
		try {
			const res = await fetch('/api/machines', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(newMachine)
			});
			if (res.ok) {
				await loadMachines();
				newMachine = {
					name: '', host: '', ssh_port: 22,
					ssh_user: '', auth_type: 'key',
					auth_private_key_path: ''
				};
			}
		} catch (e) {
			error = e.message;
		}
	}
</script>

<h1>Machines</h1>

{#if error}
	<div class="error">{error}</div>
{/if}

<div class="add-form">
	<h2>Add Machine</h2>
	<input bind:value={newMachine.name} placeholder="Name" />
	<input bind:value={newMachine.host} placeholder="Host/IP" />
	<input type="number" bind:value={newMachine.ssh_port} placeholder="SSH Port" />
	<input bind:value={newMachine.ssh_user} placeholder="SSH User" />
	<select bind:value={newMachine.auth_type}>
		<option value="key">Private Key</option>
		<option value="password">Password</option>
	</select>
	{#if newMachine.auth_type === 'key'}
		<input bind:value={newMachine.auth_private_key_path} placeholder="Private Key Path" />
	{/if}
	<button on:click={addMachine}>Add Machine</button>
</div>

{#if loading}
	<p>Loading...</p>
{:else}
	<table>
		<thead>
			<tr>
				<th>Name</th>
				<th>Host</th>
				<th>User</th>
				<th>Port</th>
				<th>Actions</th>
			</tr>
		</thead>
		<tbody>
			{#each machines as machine}
				<tr>
					<td>{machine.name}</td>
					<td>{machine.host}</td>
					<td>{machine.ssh_user}</td>
					<td>{machine.ssh_port}</td>
					<td>
						<button on:click={() => testSSH(machine.id)}>Test SSH</button>
						<button on:click={() => detect(machine.id)}>Detect</button>
					</td>
				</tr>
			{/each}
		</tbody>
	</table>
{/if}

<style>
	.add-form {
		background: #f5f5f5;
		padding: 1.5rem;
		border-radius: 8px;
		margin-bottom: 2rem;
	}

	.add-form input, .add-form select {
		display: block;
		width: 100%;
		margin: 0.5rem 0;
		padding: 0.5rem;
	}

	table {
		width: 100%;
		border-collapse: collapse;
	}

	th, td {
		padding: 0.75rem;
		text-align: left;
		border-bottom: 1px solid #ddd;
	}

	.error {
		color: red;
		padding: 1rem;
		background: #fee;
		border-radius: 4px;
		margin-bottom: 1rem;
	}
</style>
```

**Step 2: Create similar pages for models and deployments**

```svelte
<!-- frontend/src/routes/models/+page.svelte -->
<script>
	import { onMount } from 'svelte';

	let models = [];
	let loading = true;

	let newModel = {
		display_name: '',
		filename: '',
		source_url: '',
		format: 'GGUF'
	};

	onMount(async () => {
		await loadModels();
	});

	async function loadModels() {
		const res = await fetch('/api/models');
		models = await res.json();
		loading = false;
	}

	async function addModel() {
		const res = await fetch('/api/models', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify(newModel)
		});
		if (res.ok) {
			await loadModels();
			newModel = { display_name: '', filename: '', source_url: '', format: 'GGUF' };
		}
	}
</script>

<h1>Models</h1>

<div class="add-form">
	<h2>Add Model</h2>
	<input bind:value={newModel.display_name} placeholder="Display Name" />
	<input bind:value={newModel.filename} placeholder="Filename (e.g., model.gguf)" />
	<input bind:value={newModel.source_url} placeholder="Source URL (optional)" />
	<button on:click={addModel}>Add Model</button>
</div>

{#if loading}
	<p>Loading...</p>
{:else}
	<table>
		<thead>
			<tr><th>Name</th><th>Filename</th><th>Format</th></tr>
		</thead>
		<tbody>
			{#each models as model}
				<tr>
					<td>{model.display_name}</td>
					<td>{model.filename}</td>
					<td>{model.format}</td>
				</tr>
			{/each}
		</tbody>
	</table>
{/if}
```

```svelte
<!-- frontend/src/routes/deployments/+page.svelte -->
<script>
	import { onMount } from 'svelte';

	let deployments = [];
	let loading = true;

	onMount(async () => {
		const res = await fetch('/api/deployments');
		deployments = await res.json();
		loading = false;
	});
</script>

<h1>Deployments</h1>

{#if loading}
	<p>Loading...</p>
{:else}
	<table>
		<thead>
			<tr>
				<th>Name</th>
				<th>Model</th>
				<th>Machine</th>
				<th>Status</th>
				<th>Endpoint</th>
			</tr>
		</thead>
		<tbody>
			{#each deployments as deployment}
				<tr>
					<td>{deployment.name}</td>
					<td>{deployment.model_id}</td>
					<td>{deployment.machine_id}</td>
					<td>{deployment.status}</td>
					<td>{deployment.endpoint_url}</td>
				</tr>
			{/each}
		</tbody>
	</table>
{/if}
```

**Step 3: Build and commit**

```bash
cd /Users/rliu/Projects/ModelFleet/frontend
npm run build
cd /Users/rliu/Projects/ModelFleet
git add .
git commit -m "feat: add machines, models, and deployments pages"
```

---

## Phase 1J: Integration & Embedding

### Task 12: Embed Frontend in Go Binary

**Files:**
- Create: `internal/web/embed.go`
- Modify: `main.go`
- Modify: `internal/api/router.go`

**Step 1: Create embed package**

```go
// internal/web/embed.go
package web

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed all:build
var buildFS embed.FS

func FS() (fs.FS, error) {
	return fs.Sub(buildFS, "build")
}

func Handler() (http.Handler, error) {
	fsys, err := FS()
	if err != nil {
		return nil, err
	}
	return http.FileServer(http.FS(fsys)), nil
}
```

**Step 2: Update router to serve static files**

```go
// Add to internal/api/router.go
func NewRouter(...) http.Handler {
	mux := http.NewServeMux()

	// API routes...

	// Static files
	staticHandler, err := web.Handler()
	if err == nil {
		mux.Handle("/", staticHandler)
	}

	return mux
}
```

**Step 3: Update build process**

```bash
# Build frontend first
cd /Users/rliu/Projects/ModelFleet/frontend
npm run build

# Copy build to internal/web/build
cp -r build ../internal/web/

# Build Go binary
cd /Users/rliu/Projects/ModelFleet
go build -o modelfleet .
```

**Step 4: Commit**

```bash
git add .
git commit -m "feat: embed frontend in Go binary"
```

---

## Phase 1K: Health Checks & Events

### Task 13: Health Check System

**Files:**
- Create: `internal/health/checker.go`
- Create: `internal/health/scheduler.go`
- Create: `internal/events/logger.go`

**Step 1: Create health checker**

```go
// internal/health/checker.go
package health

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/modelfleet/modelfleet/internal/models"
)

type Checker struct {
	httpClient *http.Client
}

func NewChecker() *Checker {
	return &Checker{
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (c *Checker) Check(deployment *models.Deployment) (string, string) {
	// TCP check
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", deployment.Host, deployment.Port), 3*time.Second)
	if err != nil {
		return "unreachable", fmt.Sprintf("TCP connect failed: %v", err)
	}
	conn.Close()

	// HTTP check
	url := fmt.Sprintf("http://%s:%d/health", deployment.Host, deployment.Port)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return "degraded", fmt.Sprintf("HTTP check failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return "healthy", ""
	}
	return "degraded", fmt.Sprintf("HTTP status: %d", resp.StatusCode)
}
```

**Step 2: Create scheduler**

```go
// internal/health/scheduler.go
package health

import (
	"time"

	"github.com/modelfleet/modelfleet/internal/models"
	"github.com/modelfleet/modelfleet/internal/repository"
)

type Scheduler struct {
	checker        *Checker
	deploymentRepo *repository.DeploymentRepository
	interval       time.Duration
	stop           chan bool
}

func NewScheduler(deploymentRepo *repository.DeploymentRepository) *Scheduler {
	return &Scheduler{
		checker:        NewChecker(),
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
		if d.Status != "running" {
			continue
		}

		state, message := s.checker.Check(&d)

		// Update deployment status based on health
		if state == "unreachable" {
			s.deploymentRepo.UpdateStatus(d.ID, "unhealthy")
		}

		// TODO: Log health check result
		_ = message
	}
}
```

**Step 3: Commit**

```bash
git add .
git commit -m "feat: add health check system"
```

---

## Phase 1L: Final Integration

### Task 14: Wire Everything Together

**Files:**
- Modify: `main.go`
- Modify: `CODE_STATUS.md`

**Step 1: Update main.go with all components**

```go
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/modelfleet/modelfleet/internal/api"
	"github.com/modelfleet/modelfleet/internal/db"
	"github.com/modelfleet/modelfleet/internal/health"
	"github.com/modelfleet/modelfleet/internal/repository"
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

	// Initialize database
	database, err := db.New(dataDir)
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer database.Close()

	// Initialize repositories
	machineRepo := repository.NewMachineRepository(database.DB)
	modelRepo := repository.NewModelRepository(database.DB)
	deploymentRepo := repository.NewDeploymentRepository(database.DB)

	// Initialize health checker
	scheduler := health.NewScheduler(deploymentRepo)
	scheduler.Start()
	defer scheduler.Stop()

	// Create router
	router := api.NewRouter(machineRepo, modelRepo, deploymentRepo, nil)

	log.Printf("ModelFleet starting on port %s", port)
	log.Printf("API: http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatal(err)
	}
}
```

**Step 2: Update CODE_STATUS.md**

```markdown
# ModelFleet — Code Status

## Project State

**Phase:** Phase 1 MVP - Foundation Complete
**Status:** Ready for Testing

## Completed

- [x] Project initialization
- [x] Database schema (SQLite)
- [x] Core data models
- [x] Machine management API
- [x] SSH client and connection testing
- [x] Hardware detection system
- [x] Model CRUD API
- [x] Deployment planning framework
- [x] OpenAI-compatible gateway (basic)
- [x] SvelteKit frontend foundation
- [x] Frontend pages (Dashboard, Machines, Models, Deployments)
- [x] Frontend embedding in Go binary
- [x] Health check system

## In Progress

- [ ] End-to-end testing
- [ ] Remote deployment execution
- [ ] Model downloading
- [ ] Runtime binary management

## Next Steps

1. Test SSH connection flow
2. Implement remote runtime deployment
3. Add model download functionality
4. Implement actual gateway routing
5. Add API key authentication
6. Complete frontend functionality

## Notes

Foundation is complete. The architecture supports all Phase 1 features.
Remaining work focuses on remote execution and polish.
```

**Step 3: Final commit**

```bash
git add .
git commit -m "feat: integrate all components and complete MVP foundation"
```

---

## Testing

### Manual Testing Checklist

1. **Start server**
   ```bash
   go run main.go
   ```

2. **Test health endpoint**
   ```bash
   curl http://localhost:3456/health
   ```

3. **Add machine via API**
   ```bash
   curl -X POST http://localhost:3456/api/machines \
     -H "Content-Type: application/json" \
     -d '{"name":"test","host":"localhost","ssh_user":"user","auth_type":"key"}'
   ```

4. **List machines**
   ```bash
   curl http://localhost:3456/api/machines
   ```

5. **Test SSH**
   ```bash
   curl -X POST http://localhost:3456/api/machines/{id}/test-ssh
   ```

6. **Detect hardware**
   ```bash
   curl -X POST http://localhost:3456/api/machines/{id}/detect
   ```

7. **Add model**
   ```bash
   curl -X POST http://localhost:3456/api/models \
     -H "Content-Type: application/json" \
     -d '{"display_name":"Test Model","filename":"test.gguf"}'
   ```

8. **Create deployment plan**
   ```bash
   curl -X POST http://localhost:3456/api/deployments/plan \
     -H "Content-Type: application/json" \
     -d '{"machine_id":"...","model_id":"..."}'
   ```

9. **Test gateway**
   ```bash
   curl http://localhost:3456/v1/models
   ```

---

## Build Commands

```bash
# Development
go run main.go

# Production build
cd frontend && npm run build && cd ..
cp -r frontend/build internal/web/
go build -ldflags "-s -w" -o modelfleet .

# Cross-compilation
GOOS=linux GOARCH=amd64 go build -o modelfleet-linux-amd64
GOOS=darwin GOARCH=amd64 go build -o modelfleet-darwin-amd64
GOOS=darwin GOARCH=arm64 go build -o modelfleet-darwin-arm64
```
