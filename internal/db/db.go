package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	*sql.DB
}

func New(dataDir string) (*DB, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	dbPath := filepath.Join(dataDir, "modelfleet.db")
	sqlDB, err := sql.Open("sqlite3", dbPath+"?_fk=1")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}

	if err := migrate(sqlDB); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return &DB{sqlDB}, nil
}

func migrate(db *sql.DB) error {
	schema := `
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
`

	_, err := db.Exec(schema)
	return err
}
