# ModelFleet Code Review: Implementation vs Specification

## Overall Assessment

The current implementation provides a **functional foundation** (roughly 30% of Phase 1 MVP) with clean architecture but has **significant gaps** in core operational features required by the specification.

---

## What Is Implemented ✅

### 1. Project Foundation
- [x] Go backend with single binary
- [x] Default port 3456
- [x] Zero external dependencies
- [x] JSON file-based storage (deviation: spec requires SQLite)
- [x] Basic HTTP server with routing

### 2. REST API (Partial)
- [x] **Machines**: CRUD (GET /api/machines, POST /api/machines, GET /api/machines/:id, DELETE)
- [x] **SSH Test**: POST /api/machines/:id/test-ssh
- [x] **Hardware Detection**: POST /api/machines/:id/detect
- [x] **Models**: CRUD (GET /api/models, POST /api/models, GET /api/models/:id, DELETE)
- [x] **Deployments**: CRUD (GET /api/deployments, POST /api/deployments, GET /api/deployments/:id, DELETE)

### 3. OpenAI-Compatible Gateway (Partial)
- [x] GET /v1/models (lists deployment names, but doesn't verify health)
- [x] POST /v1/chat/completions (returns placeholder response, no actual routing)

### 4. Hardware Detection
- [x] Runs via SSH using os/exec
- [x] Collects OS, arch, CPU, memory, disk info
- [x] Detects NVIDIA (nvidia-smi), AMD ROCm, Vulkan presence
- [x] Normalizes to capability states (cpu_only, nvidia_ready, etc.)
- [x] Detection is read-only

### 5. SSH Client
- [x] Private key authentication (via ssh command)
- [x] Password authentication placeholder
- [x] SSH port selection
- [x] Runs `echo MODEL_FLEET_SSH_OK` test

### 6. Web GUI
- [x] Single-page HTML/JS frontend (no build step)
- [x] Dashboard with basic stats
- [x] Machines page (add, list, test SSH, detect)
- [x] Models page (add, list)
- [x] Deployments page (create, list)
- [x] Gateway page (API docs, examples)
- [x] Navigation between pages

### 7. Data Models
- [x] Machine model with all required fields
- [x] Model catalog fields
- [x] Deployment record fields
- [x] Default values (base_dir: ~/modelfleet, port: 8081)

### 8. Core Principles
- [x] Lightweight (zero deps, no Docker/K8s)
- [x] Safe remote config (only SSH commands, no driver install)
- [x] No forced reboots
- [x] No invasive system modifications

---

## What Is Missing or Incomplete ❌

### CRITICAL GAPS (Required for MVP)

#### 1. Database: SQLite → JSON Files (Section 4.3, 18)
**Spec Requirement**: SQLite database with 10 tables
**Current**: JSON file storage with 3 collections
**Impact**: MEDIUM - Functionally works but not spec-compliant

Missing tables/collections:
- `machine_facts` - Detection results not persisted
- `runtime_artifacts` - No runtime manifest
- `aliases` - No alias system
- `api_keys` - No API key management
- `health_checks` - No health check history
- `events` - No event logging
- `settings` - No configuration storage

#### 2. Deployment Execution (Section 12) - MAJOR GAP
**Spec Requirement**: Full remote deployment execution
**Current**: Only creates deployment record locally

Missing:
- No remote directory creation (`ensure_directory`)
- No runtime binary download (`download_file`)
- No model file download
- No checksum verification (`verify_checksum`)
- No process start/stop (`start_process`, `stop_process`)
- No HTTP health check after start (`check_http`)
- No log tailing (`tail_logs`)
- No idempotency logic
- No port conflict detection

**Acceptance Criteria Affected**: Steps 10-14, 16 completely broken

#### 3. Gateway Routing (Section 16) - MAJOR GAP
**Spec Requirement**: Route requests to correct healthy deployment
**Current**: Returns hardcoded placeholder response

Missing:
- No deployment lookup by model ID
- No health state verification before routing
- No proxy/forward to remote llama-server
- No alias resolution
- No routing policies (single_deployment, first_healthy, etc.)
- No 503 error for unhealthy deployments
- Proper error format with `modelfleet_no_healthy_deployment`

**Acceptance Criteria Affected**: Steps 18-20 completely broken

#### 4. Health Checks (Section 14) - MAJOR GAP
**Spec Requirement**: Periodic health checking (30s interval)
**Current**: No health check system at all

Missing:
- No TCP connect checks
- No HTTP endpoint checks
- No periodic scheduler
- No health state transitions (healthy, degraded, unreachable, stopped)
- No deployment status updates based on health

**Acceptance Criteria Affected**: Steps 15-16 completely broken

#### 5. API Keys (Section 17) - MISSING
**Spec Requirement**: API key support for gateway access
**Current**: No API key system

Missing:
- No API key generation
- No key hashing
- No Bearer token authentication on /v1/* endpoints
- No enabled/disabled key management
- No "show once" behavior

#### 6. Events & Logging (Section 22-23) - MISSING
**Spec Requirement**: Every failed operation creates event record
**Current**: No event system, only basic stdout logging

Missing:
- No events table/collection
- No event types (ssh_error, detection, deployment, health_check)
- No severity levels
- No structured logging
- SSH attempts not logged
- Deployment steps not logged
- Gateway errors not logged
- No log viewing endpoint

#### 7. Deployment Lifecycle (Section 12.3, 20.5) - MISSING
**Spec Requirement**: Start/stop/restart deployments, view logs
**Current**: Deployments created in "planned" status forever

Missing API Endpoints:
- POST /api/deployments/:id/start
- POST /api/deployments/:id/stop
- POST /api/deployments/:id/restart
- GET /api/deployments/:id/logs

#### 8. Machine Facts Persistence (Section 8, 18.2) - MISSING
**Spec Requirement**: Persist detection results in machine_facts table
**Current**: Detection returns result but doesn't store it

Missing:
- No machine_facts collection
- No ability to view historical detection results
- No capability state persistence
- No "last detected" tracking in machines list

#### 9. Deployment Planning (Section 11) - MISSING
**Spec Requirement**: Show deployment plan before execution
**Current**: No plan endpoint

Missing:
- POST /api/deployments/plan endpoint
- No deployment wizard flow
- No plan display with actions list
- No "requires root/reboot" indicators
- No runtime recommendation based on detection

#### 10. Runtime Management (Section 9) - MISSING
**Spec Requirement**: Runtime artifact manifest, selection rules
**Current**: Hardcoded "llama.cpp" with "cpu" backend

Missing:
- No runtime_artifacts collection
- No backend recommendation logic (Metal for macOS ARM64, CUDA for Linux+NVIDIA, etc.)
- No runtime download/verification
- No binary management

#### 11. Aliases (Section 15.2) - MISSING
**Spec Requirement**: Model aliases with routing policies
**Current**: No alias system

Missing:
- No alias creation/management
- No alias resolution in gateway
- No routing policies

---

### PARTIAL IMPLEMENTATIONS

#### 12. GUI Pages (Section 20)
**Status**: Basic pages exist but missing features

Missing GUI Features:
- [ ] Machine detail page (Section 20.3) - no detailed hardware view
- [ ] Deployment wizard (Section 20.6) - no step-by-step flow
- [ ] Events/logs page (Section 20.8) - no event viewer
- [ ] Start/stop/restart deployment buttons
- [ ] View deployment logs
- [ ] Edit machine/model metadata
- [ ] Network exposure warnings (Section 19.3)
- [ ] API key management page
- [ ] Alias management

#### 13. Deployment Statuses (Section 13)
**Spec Requirement**: planned, deploying, running, stopped, unhealthy, failed, unknown
**Current**: Only "planned" status ever used

No transitions between states because:
- No deployment execution
- No health checks
- No process management

#### 14. SSH Authentication (Section 7.1, 19.1)
**Status**: Partial

Issues:
- Password auth not implemented (only key auth works)
- No known_hosts verification (uses StrictHostKeyChecking=no)
- No password encryption
- Private key paths stored in plaintext

#### 15. Port Management (Section 21)
**Spec Requirement**: Auto-find unused port in range 8081-8099
**Current**: Fixed default 8081, no collision detection

---

## SPECIFIC SPEC VIOLATIONS

### Section 4.3: Database
- **Violation**: Using JSON files instead of SQLite
- **Severity**: MEDIUM (functional but not spec-compliant)
- **Reason**: Network constraints prevented fetching SQLite driver

### Section 16.4: Failure Behavior
- **Violation**: Returns 200 with placeholder instead of 503 for missing deployments
- **Severity**: HIGH
- **Code**: `internal/api/gateway.go:55-70`

### Section 19.2: API Keys
- **Violation**: Gateway endpoints completely open, no authentication
- **Severity**: HIGH
- **Code**: `internal/api/router.go:109-123`

### Section 22: Error Handling
- **Violation**: Errors not always user-readable, no event records
- **Severity**: MEDIUM
- **Example**: SSH errors show raw command output

### Section 12.2: Idempotency
- **Violation**: No idempotency checks anywhere
- **Severity**: HIGH
- **Missing**: Directory existence checks, checksum verification, duplicate process detection

---

## CODE QUALITY ISSUES

### Architecture
1. **No separation of concerns**: SSH execution mixed with HTTP handlers in `internal/api/common.go`
2. **No service layer**: Business logic directly in handlers
3. **No dependency injection**: Repositories created inline in router

### Security
1. **Open gateway**: No API key validation on /v1/* endpoints
2. **Insecure SSH**: `StrictHostKeyChecking=no` bypasses host verification
3. **No input validation**: No sanitization of machine names, hosts, etc.
4. **No rate limiting**: Gateway open to abuse

### Error Handling
1. **Silent failures**: Detection errors often ignored
2. **No retry logic**: SSH commands fail permanently
3. **Poor error messages**: Raw OS errors exposed to users

### Testing
1. **No unit tests**: Zero test coverage
2. **No integration tests**: Manual testing only
3. **No mocking**: SSH tests require actual remote machine

---

## MVP ACCEPTANCE CRITERIA STATUS

| Step | Requirement | Status | Notes |
|------|-------------|--------|-------|
| 1 | Install ModelFleet | ✅ | Go binary works |
| 2 | Open web GUI | ✅ | Frontend loads |
| 3 | Add remote machine | ✅ | API works |
| 4 | Test SSH | ✅ | Test command works |
| 5 | Detect hardware | ✅ | Detection runs |
| 6 | Add GGUF model | ✅ | Model CRUD works |
| 7 | Start deployment wizard | ❌ | No wizard exists |
| 8 | Recommend backend | ❌ | No recommendation logic |
| 9 | Review deployment plan | ❌ | No plan endpoint |
| 10 | Click Deploy | ⚠️ | Creates record only, no remote action |
| 11 | Create remote directories | ❌ | Not implemented |
| 12 | Install/download runtime | ❌ | Not implemented |
| 13 | Download model | ❌ | Not implemented |
| 14 | Start llama-server | ❌ | Not implemented |
| 15 | Health-check deployment | ❌ | No health checks |
| 16 | Deployment appears healthy | ❌ | Status never changes from "planned" |
| 17 | /v1/models shows alias | ⚠️ | Shows deployment name, not alias |
| 18 | Send chat completion request | ✅ | Endpoint accepts request |
| 19 | Route to remote deployment | ❌ | Returns placeholder |
| 20 | Receive valid model response | ❌ | Placeholder response only |

**Score**: 6/20 steps working end-to-end (30%)

---

## PRIORITIZED RECOMMENDATIONS

### P0 - Must Have for MVP
1. **Deployment Execution**: Implement remote SSH commands to start llama-server
2. **Gateway Routing**: Forward requests to remote deployments
3. **Health Checks**: Periodic TCP/HTTP checks with status updates
4. **API Keys**: Add authentication to gateway endpoints

### P1 - Should Have
5. **SQLite Database**: Replace JSON files with SQLite (go-sqlite3)
6. **Events System**: Log all operations to events table
7. **Deployment Lifecycle**: Start/stop endpoints with process management
8. **Machine Facts**: Persist detection results

### P2 - Nice to Have
9. **Deployment Planning**: Add plan endpoint with action preview
10. **Runtime Management**: Download and verify llama.cpp binaries
11. **Aliases**: Model alias system
12. **GUI Polish**: Machine detail page, deployment wizard, events viewer

---

## CONCLUSION

The current implementation is a **solid architectural foundation** with clean separation of concerns and proper API structure. However, it is **not yet an MVP** because the core operational loop is incomplete:

**Missing**: Deploying models remotely, routing requests, health monitoring, and authentication.

**What works**: Machine onboarding, hardware detection, model catalog, and basic CRUD operations.

**Estimated effort to MVP**: ~2-3 weeks for a single developer to implement P0 items (deployment execution, gateway routing, health checks, API keys).
