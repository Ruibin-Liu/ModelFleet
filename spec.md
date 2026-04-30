# ModelFleet — Product & Engineering Specification

Version: **0.1 MVP Spec**  
Scope: **Phase 1 only**  
Mobile support: **deferred to Phase 2**  
Primary goal: **Deploy models across user-owned machines and expose them through one unified API.**

---

# 1. Product Summary

**ModelFleet** is a lightweight control plane for managing LLM serving across machines the user controls.

It lets a user:

1. Add remote machines via SSH.
2. Detect each machine’s hardware and software environment.
3. Recommend suitable model/runtime deployments.
4. Deploy lightweight model servers remotely.
5. Track all deployed models in one inventory.
6. Expose all deployed models through one OpenAI-compatible API endpoint.
7. Route requests to the correct machine/model automatically.

Primary tagline:

> **ModelFleet — deploy models anywhere, use them from one API.**

---

# 2. Core Principles

## 2.1 Lightweight by default

ModelFleet must avoid unnecessary dependencies.

Default behavior:

- No Docker requirement.
- No Kubernetes requirement.
- No Ansible requirement.
- No Ollama requirement.
- No automatic GPU driver installation.
- No automatic CUDA/ROCm driver installation.
- No forced reboots.
- No invasive system modifications unless the user explicitly approves.

## 2.2 Safe remote configuration

ModelFleet may configure remote machines, but must avoid risky operations.

Allowed by default:

- SSH command execution.
- Creating files/directories under the remote user’s home directory.
- Downloading runtime binaries.
- Downloading model files.
- Starting/stopping user-owned processes.
- Creating user-level service files when supported.

Not allowed by default:

- Installing or upgrading GPU drivers.
- Installing kernel modules.
- Performing OS upgrades.
- Rebooting machines.
- Modifying system services.
- Opening firewall ports.
- Installing Docker.
- Installing CUDA/ROCm system packages.

If a machine needs driver work, ModelFleet must report it and ask the user to fix it manually.

Example message:

```text
NVIDIA GPU detected, but nvidia-smi is unavailable.

ModelFleet will not install or upgrade GPU drivers automatically because this can
require a reboot or break the machine.

You can:
1. Deploy CPU runtime now.
2. Manually install or repair NVIDIA drivers.
3. Rerun detection later.
```

## 2.3 Unified API is central

The most important product feature is not remote installation.

The most important feature is:

> ModelFleet knows what models are deployed where and exposes them through one unified API.

All compatible clients should be able to use:

```text
http://<modelfleet-host>:<port>/v1/models
http://<modelfleet-host>:<port>/v1/chat/completions
```

The API should be OpenAI-compatible where practical.

`llama.cpp`/`llama-server` is an appropriate default runtime because it supports lightweight GGUF serving and OpenAI-compatible endpoints [ref:9].

---

# 3. Phase Scope

## 3.1 Phase 1: MVP

Phase 1 includes:

- Main ModelFleet daemon.
- Web GUI.
- SQLite database.
- SSH machine onboarding.
- Hardware/software detection.
- Lightweight remote runtime deployment.
- GGUF model deployment.
- `llama.cpp`/`llama-server` support.
- Remote process lifecycle management.
- Model inventory.
- Health checks.
- Unified OpenAI-compatible API gateway.
- API key support.
- Basic routing.
- Basic logs/events.

## 3.2 Explicitly out of scope for Phase 1

Do **not** implement in Phase 1:

- Mobile app.
- Mobile model deployment.
- Phone pairing.
- Public marketplace.
- Payment integration.
- Kubernetes.
- Docker-based deployments.
- Ollama integration.
- vLLM/SGLang/TGI support.
- Multi-tenant billing.
- Fine-tuning.
- RAG/document management.
- Automatic GPU driver installation.
- Automatic CUDA/ROCm installation.
- Complex RBAC.
- Cloud provider provisioning.

---

# 4. Recommended Tech Stack

## 4.1 Backend

Use:

```text
Go
```

Reasons:

- Single binary deployment.
- Good SSH support.
- Good HTTP reverse proxy support.
- Good concurrency.
- Low memory usage.
- Easy cross-compilation.
- Works well on Linux, macOS, Windows, Raspberry Pi, mini PCs, servers.

## 4.2 Frontend

Use one of:

```text
SvelteKit
React
Vue
```

Recommended:

```text
SvelteKit
```

The frontend should be compiled and embedded into the Go binary.

## 4.3 Database

Use:

```text
SQLite
```

Do not require Postgres in Phase 1.

## 4.4 Remote access

Use:

```text
SSH
SFTP/SCP-compatible file upload
```

The backend must support:

- SSH private key authentication.
- Password authentication, optional.
- SSH port selection.
- Known-host/fingerprint verification.

## 4.5 Runtime

Default remote model runtime:

```text
llama.cpp llama-server
```

Supported model format in Phase 1:

```text
GGUF
```

---

# 5. Deployment Model

ModelFleet itself runs on one always-on control machine.

Examples:

- Mini PC.
- Old laptop.
- NAS.
- Home server.
- Workstation.
- Raspberry Pi 5 with SSD.
- Cloud VPS if remote machines are reachable.

ModelFleet must not require a phone as the control host.

Architecture:

```text
Client Apps
  |
  | OpenAI-compatible API
  v
ModelFleet Control Plane
  |
  | SSH / HTTP health checks
  v
Remote Machines
  |
  | llama-server
  v
Deployed Models
```

---

# 6. Main Components

## 6.1 Web GUI

The GUI must support:

- Adding machines.
- Testing SSH connections.
- Running detection.
- Viewing hardware/software facts.
- Viewing recommended deployment options.
- Deploying models.
- Viewing deployments.
- Starting/stopping deployments.
- Viewing health status.
- Viewing logs/events.
- Managing model aliases.
- Managing API keys.
- Viewing unified API usage instructions.

## 6.2 Control Plane API

The backend must expose internal API endpoints for the GUI.

Example internal routes:

```text
GET    /api/machines
POST   /api/machines
GET    /api/machines/:id
POST   /api/machines/:id/test-ssh
POST   /api/machines/:id/detect
GET    /api/models
POST   /api/models
GET    /api/deployments
POST   /api/deployments/plan
POST   /api/deployments
POST   /api/deployments/:id/start
POST   /api/deployments/:id/stop
GET    /api/deployments/:id/logs
GET    /api/events
GET    /api/settings
```

## 6.3 OpenAI-Compatible Gateway

ModelFleet must expose:

```text
GET  /v1/models
POST /v1/chat/completions
```

Optional later in Phase 1:

```text
POST /v1/completions
POST /v1/embeddings
```

The gateway routes incoming requests to registered deployments.

---

# 7. Machine Onboarding

## 7.1 Add machine form

Required fields:

```text
Machine name
Host/IP
SSH port
SSH username
Authentication method
```

Supported auth methods:

```text
Private key
Password
```

Optional fields:

```text
Tags
Notes
Preferred model directory
Preferred runtime directory
```

Default remote base directory:

```text
~/modelfleet
```

Remote directory layout:

```text
~/modelfleet/
  bin/
  models/
  runtimes/
  deployments/
  logs/
  tmp/
```

## 7.2 SSH test

When the user clicks **Test SSH**, ModelFleet must:

1. Attempt SSH connection.
2. Verify authentication.
3. Run a harmless command:

```bash
echo MODEL_FLEET_SSH_OK
```

4. Return success/failure to GUI.

---

# 8. Detection System

## 8.1 Detection must be read-only

Detection must not modify the remote machine.

## 8.2 Detection commands

The detector should collect:

### OS and architecture

```bash
uname -a
cat /etc/os-release
```

### CPU

```bash
lscpu
```

### Memory

```bash
free -b
```

### Disk

```bash
df -B1 ~
```

### NVIDIA

```bash
command -v nvidia-smi
nvidia-smi --query-gpu=name,memory.total,driver_version --format=csv,noheader
```

### AMD ROCm

```bash
command -v rocminfo
rocminfo
```

### Vulkan

```bash
command -v vulkaninfo
vulkaninfo --summary
```

### Systemd user support

```bash
systemctl --user --version
systemctl --user status
```

### Shell tools

```bash
command -v curl
command -v wget
command -v tar
command -v gzip
command -v sha256sum
```

## 8.3 Normalized machine capability states

ModelFleet must normalize detection results into capability states.

Possible values:

```text
cpu_only
nvidia_ready
nvidia_present_driver_missing
nvidia_driver_too_old
amd_rocm_ready
amd_present_rocm_missing
vulkan_ready
mac_metal_ready
unknown
```

## 8.4 Driver policy

If GPU hardware is detected but runtime support is not working, ModelFleet must not fix it automatically.

Examples:

- NVIDIA card exists but `nvidia-smi` missing.
- AMD GPU exists but ROCm unavailable.
- Driver version appears too old.

In these cases, ModelFleet should offer CPU deployment and display manual remediation guidance.

---

# 9. Runtime Management

## 9.1 Phase 1 runtime

Phase 1 supports only:

```text
llama.cpp llama-server
```

## 9.2 Runtime artifacts

ModelFleet should maintain a runtime artifact manifest.

Example:

```json
{
  "runtime": "llama.cpp",
  "version": "bXXXX",
  "artifacts": [
    {
      "os": "linux",
      "arch": "x86_64",
      "backend": "cpu",
      "url": "https://example.com/llama-server-linux-x64-cpu.tar.gz",
      "sha256": "..."
    },
    {
      "os": "linux",
      "arch": "x86_64",
      "backend": "cuda",
      "url": "https://example.com/llama-server-linux-x64-cuda.tar.gz",
      "sha256": "..."
    }
  ]
}
```

## 9.3 Runtime selection rules

Use these default rules:

```text
If macOS arm64:
  recommend llama.cpp Metal build.

If Linux + NVIDIA + nvidia-smi works:
  recommend llama.cpp CUDA build.

If Linux + Vulkan available:
  recommend llama.cpp Vulkan build.

Otherwise:
  recommend llama.cpp CPU build.
```

If uncertain, recommend CPU build.

---

# 10. Model Catalog

## 10.1 Model record

Each model should have:

```text
id
display_name
family
parameter_size
format
quantization
context_length
source_url
filename
sha256_optional
estimated_ram_bytes
estimated_vram_bytes
tags
created_at
updated_at
```

Example:

```json
{
  "id": "qwen2.5-7b-instruct-q4km",
  "display_name": "Qwen2.5 7B Instruct Q4_K_M",
  "family": "qwen",
  "parameter_size": "7B",
  "format": "GGUF",
  "quantization": "Q4_K_M",
  "context_length": 32768,
  "source_url": "https://...",
  "filename": "qwen2.5-7b-instruct-q4_k_m.gguf",
  "estimated_ram_bytes": 7000000000,
  "estimated_vram_bytes": 6000000000,
  "tags": ["chat", "instruct"]
}
```

## 10.2 Phase 1 model sources

Phase 1 may support:

- User-provided direct URL.
- User-uploaded model metadata.
- Manual Hugging Face URL.

Do not build a full model marketplace in Phase 1.

---

# 11. Deployment Planning

## 11.1 Deployment wizard

The GUI must provide a deployment flow:

```text
Select machine
Select model
Review detected capabilities
Review recommended runtime
Review deployment settings
Show deployment plan
User confirms
Execute deployment
Register deployment
Health check
```

## 11.2 Deployment plan

Before execution, ModelFleet must show a plan.

Example:

```text
Deployment Plan

Machine:
  office-gpu
  Ubuntu 24.04 x86_64
  RAM: 64 GB
  GPU: RTX 3090 24 GB
  Driver: working

Model:
  Qwen2.5 7B Instruct Q4_K_M

Runtime:
  llama.cpp CUDA

Actions:
  1. Create ~/modelfleet directories.
  2. Download llama-server CUDA binary.
  3. Verify runtime checksum.
  4. Download model file.
  5. Create deployment config.
  6. Start llama-server on port 8081.
  7. Health check endpoint.
  8. Register deployment in ModelFleet gateway.

Requires root:
  No

Requires reboot:
  No
```

## 11.3 Deployment settings

Deployment settings must include:

```text
deployment_name
machine_id
model_id
runtime
backend
port
host
context_size
gpu_layers
threads
batch_size_optional
extra_args_optional
```

Defaults:

```text
host = 127.0.0.1 or 0.0.0.0 depending on reachability
context_size = model default or 4096
gpu_layers = 0 for CPU
threads = auto
```

Important: if ModelFleet gateway must access the remote runtime over LAN, the runtime must bind to a reachable address.

Recommended default:

```text
host = 0.0.0.0
```

But the GUI should warn:

```text
This exposes the runtime on the remote machine's network interface.
Use firewall/VPN/private LAN where appropriate.
```

---

# 12. Remote Deployment Execution

## 12.1 Required actions

The deployment executor must support these idempotent actions:

```text
ensure_directory
download_file
upload_file
verify_checksum
make_executable
write_config
start_process
stop_process
check_http
tail_logs
```

## 12.2 Idempotency

Every action should be safe to retry.

Examples:

- If directory exists, do nothing.
- If runtime binary exists and checksum matches, do not redownload.
- If model file exists and checksum matches, do not redownload.
- If process is already running for the deployment, do not start duplicate process.
- If port is occupied by another process, fail with a clear error.

## 12.3 Process management

Phase 1 supports two process modes:

### Mode A: direct background process

Start with:

```bash
nohup ~/modelfleet/bin/llama-server ... > ~/modelfleet/logs/<deployment>.log 2>&1 &
echo $! > ~/modelfleet/deployments/<deployment>.pid
```

### Mode B: user systemd service

If user systemd is available, ModelFleet may create:

```text
~/.config/systemd/user/modelfleet-<deployment>.service
```

User systemd is preferred when available.

Do not require root.

Do not enable linger unless the user explicitly approves.

---

# 13. Deployment Record

A deployment represents one running or configured model server.

Fields:

```text
id
name
machine_id
model_id
runtime
backend
status
host
port
endpoint_url
api_format
context_size
gpu_layers
threads
process_mode
pid_optional
service_name_optional
created_at
updated_at
last_health_check_at
last_error
```

Deployment statuses:

```text
planned
deploying
running
stopped
unhealthy
failed
unknown
```

---

# 14. Health Checks

ModelFleet must health-check each deployment periodically.

Minimum checks:

1. TCP connect to deployment host/port.
2. HTTP request to runtime endpoint.
3. Optional model list check.

Health states:

```text
healthy
degraded
unreachable
stopped
unknown
```

Recommended interval:

```text
30 seconds default
configurable
```

---

# 15. Unified Model Registry

## 15.1 `/v1/models`

ModelFleet’s `/v1/models` endpoint should return models available through the gateway.

Example response:

```json
{
  "object": "list",
  "data": [
    {
      "id": "local/qwen-7b",
      "object": "model",
      "owned_by": "modelfleet"
    },
    {
      "id": "local/coder-small",
      "object": "model",
      "owned_by": "modelfleet"
    }
  ]
}
```

## 15.2 Aliases

Users must be able to create model aliases.

Example aliases:

```text
local/fast       -> qwen2.5-3b deployment
local/best       -> qwen2.5-14b deployment
local/coder      -> coder model deployment
local/private    -> any private approved local model
```

Alias fields:

```text
id
alias
target_type
target_deployment_id_optional
target_model_id_optional
routing_policy
created_at
updated_at
```

---

# 16. Gateway Routing

## 16.1 Explicit routing

If a request specifies a deployment-backed model ID:

```json
{
  "model": "local/qwen-7b",
  "messages": []
}
```

Route to the matching healthy deployment.

## 16.2 Alias routing

If a request specifies an alias:

```json
{
  "model": "local/fast",
  "messages": []
}
```

Resolve the alias to a deployment or routing policy.

## 16.3 Routing policies

Phase 1 should support simple policies:

```text
single_deployment
first_healthy
least_recently_used
fallback_order
```

Do not implement complex load balancing in Phase 1.

## 16.4 Failure behavior

If no healthy deployment exists, return:

```http
503 Service Unavailable
```

Example body:

```json
{
  "error": {
    "message": "No healthy deployment available for model local/qwen-7b",
    "type": "modelfleet_no_healthy_deployment"
  }
}
```

---

# 17. API Keys

ModelFleet must support API keys for gateway access.

Fields:

```text
id
name
key_hash
created_at
last_used_at
enabled
```

The raw key must only be shown once.

Gateway requests should use:

```http
Authorization: Bearer <api_key>
```

Phase 1 permissions can be simple:

```text
enabled/disabled
```

No complex RBAC required.

---

# 18. Database Schema

Minimum tables:

```text
machines
machine_facts
models
runtime_artifacts
deployments
aliases
api_keys
health_checks
events
settings
```

## 18.1 `machines`

```sql
CREATE TABLE machines (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  host TEXT NOT NULL,
  ssh_port INTEGER NOT NULL DEFAULT 22,
  ssh_user TEXT NOT NULL,
  auth_type TEXT NOT NULL,
  tags TEXT,
  notes TEXT,
  base_dir TEXT NOT NULL DEFAULT '~/modelfleet',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
```

## 18.2 `machine_facts`

```sql
CREATE TABLE machine_facts (
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
  detected_at TEXT NOT NULL
);
```

## 18.3 `models`

```sql
CREATE TABLE models (
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
```

## 18.4 `deployments`

```sql
CREATE TABLE deployments (
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
  last_error TEXT
);
```

## 18.5 `aliases`

```sql
CREATE TABLE aliases (
  id TEXT PRIMARY KEY,
  alias TEXT NOT NULL UNIQUE,
  target_type TEXT NOT NULL,
  target_deployment_id TEXT,
  target_model_id TEXT,
  routing_policy TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
```

## 18.6 `api_keys`

```sql
CREATE TABLE api_keys (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  key_hash TEXT NOT NULL,
  enabled INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL,
  last_used_at TEXT
);
```

## 18.7 `events`

```sql
CREATE TABLE events (
  id TEXT PRIMARY KEY,
  type TEXT NOT NULL,
  severity TEXT NOT NULL,
  machine_id TEXT,
  deployment_id TEXT,
  message TEXT NOT NULL,
  data_json TEXT,
  created_at TEXT NOT NULL
);
```

---

# 19. Security Requirements

## 19.1 SSH credentials

ModelFleet must store SSH credentials securely.

Minimum acceptable for MVP:

- Store private key references by path when possible.
- Avoid storing private key material in the database unless encrypted.
- If passwords are stored, encrypt them with an application secret.
- Never log passwords or private keys.

## 19.2 API keys

- Store only hashed API keys.
- Show raw API keys once.
- Allow disabling keys.
- Require API key for `/v1/*` endpoints by default.

## 19.3 Network exposure

The GUI should clearly distinguish:

```text
Local-only access
LAN access
Public internet access
```

If the user binds ModelFleet to `0.0.0.0`, show a warning.

If the user exposes remote runtimes on `0.0.0.0`, show a warning.

## 19.4 Public serving

Public selling/sharing of LLM services is not Phase 1.

Do not add payment or public marketplace features yet.

---

# 20. GUI Pages

## 20.1 Dashboard

Shows:

- Number of machines.
- Number of healthy deployments.
- Number of unhealthy deployments.
- Gateway URL.
- Recent events.
- Quick API example.

## 20.2 Machines page

Shows:

```text
Name
Host
OS
CPU
GPU
RAM
Capability state
Last detected
Status
```

Actions:

```text
Add machine
Test SSH
Detect
Edit
Remove
```

## 20.3 Machine detail page

Shows:

- SSH info.
- Hardware facts.
- Software facts.
- Capability classification.
- Deployed models on this machine.
- Recommended deployments.
- Event history.

## 20.4 Models page

Shows:

- Known model catalog.
- Add model by URL.
- Edit metadata.
- Delete model metadata.

## 20.5 Deployments page

Shows:

```text
Deployment
Model
Machine
Runtime
Backend
Endpoint
Status
Health
```

Actions:

```text
Deploy new model
Start
Stop
Restart
View logs
Remove
```

## 20.6 Deployment wizard

Steps:

```text
1. Select machine.
2. Select model.
3. Review recommendation.
4. Configure runtime options.
5. Review plan.
6. Deploy.
```

## 20.7 Gateway page

Shows:

- Base URL.
- API keys.
- Model aliases.
- `/v1/models` preview.
- Example curl command.

Example:

```bash
curl http://localhost:3456/v1/chat/completions \
  -H "Authorization: Bearer <MODEL_FLEET_API_KEY>" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "local/fast",
    "messages": [
      {"role": "user", "content": "Hello"}
    ]
  }'
```

## 20.8 Events/logs page

Shows:

- Deployment events.
- Detection events.
- SSH errors.
- Health check failures.
- Runtime logs.

---

# 21. Default Ports

ModelFleet control plane default:

```text
3456
```

Remote deployment default port range:

```text
8081-8099
```

When deploying a new model, ModelFleet should automatically find an unused port on the remote machine.

---

# 22. Error Handling

Errors must be user-readable.

Bad:

```text
exit status 1
```

Good:

```text
Could not start llama-server because port 8081 is already in use on machine office-gpu.
Choose a different port or stop the existing process.
```

Every failed operation should create an event record.

---

# 23. Logging

ModelFleet must log:

- SSH connection attempts.
- Detection runs.
- Deployment plans.
- Deployment execution steps.
- Runtime start/stop.
- Health check failures.
- Gateway routing errors.

Do not log:

- API key raw values.
- SSH passwords.
- Private keys.
- Full prompt bodies by default.

Prompt logging should be disabled by default.

---

# 24. Phase 2 Placeholder: Mobile

Mobile is explicitly Phase 2.

Phase 2 may include:

- Simple mobile companion app.
- QR pairing.
- Fleet status dashboard.
- Start/stop deployments.
- Notifications.
- Optional phone model testing.
- Optional tiny GGUF model deployment to phone.

Phase 1 code should not depend on mobile features.

However, the architecture should not prevent adding future non-SSH node types.

Therefore, machine/node records should eventually support:

```text
connection_type = ssh | agent | local
```

For Phase 1, only this is required:

```text
connection_type = ssh
```

---

# 25. Future Roadmap

## Phase 1

Private/local fleet MVP.

## Phase 2

Mobile companion and optional mobile model testing.

## Phase 3

Team sharing:

- User accounts.
- Better permissions.
- Per-key usage tracking.
- Internal model sharing.

## Phase 4

Public service mode:

- Public endpoint hardening.
- Rate limits.
- Quotas.
- Billing hooks.
- Abuse protection.
- TLS automation.
- Usage analytics.

---

# 26. Non-Goals

ModelFleet is **not**:

- A chat UI replacement.
- A full RAG platform.
- A Kubernetes platform.
- A cloud GPU provider.
- A model marketplace.
- A driver installer.
- A fine-tuning platform.
- A Docker management UI.
- An Ansible GUI.

ModelFleet is:

> A lightweight LLM fleet control plane with deployment, inventory, health, and unified API routing.

---

# 27. MVP Acceptance Criteria

The MVP is successful when the following workflow works end-to-end:

1. User installs ModelFleet on a control machine.
2. User opens web GUI.
3. User adds a remote Linux machine over SSH.
4. ModelFleet tests SSH successfully.
5. ModelFleet detects hardware/software.
6. User adds a GGUF model URL.
7. User starts deployment wizard.
8. ModelFleet recommends `llama.cpp` CPU/CUDA/Vulkan/Metal backend.
9. User reviews deployment plan.
10. User clicks Deploy.
11. ModelFleet creates remote directories.
12. ModelFleet installs/downloads runtime.
13. ModelFleet downloads model.
14. ModelFleet starts `llama-server`.
15. ModelFleet health-checks the deployment.
16. Deployment appears as healthy.
17. `/v1/models` shows the deployed model alias.
18. User sends a request to `/v1/chat/completions`.
19. ModelFleet routes request to remote deployment.
20. User receives a valid model response.

If all 20 steps work reliably, Phase 1 MVP is complete.

