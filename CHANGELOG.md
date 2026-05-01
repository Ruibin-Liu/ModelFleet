# Changelog

All notable changes to ModelFleet will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Complete deployment execution via SSH
  - Remote directory creation (`~/modelfleet/{bin,models,logs,deployments}`)
  - Model file download from source URLs
  - llama-server binary verification
  - Port conflict detection
  - Process start/stop with PID tracking
  - Graceful shutdown with force kill fallback
- Real gateway routing to remote deployments
  - `POST /v1/chat/completions` proxies requests to remote llama-server
  - Proper 503 errors for unhealthy/missing deployments
  - Response streaming support
- Health check system
  - TCP connectivity checks every 30 seconds
  - HTTP endpoint verification
  - Automatic status transitions
- Event logging system
  - Structured logging for all operations
  - Event types: ssh, detection, deployment, health_check, gateway
  - `GET /api/events` endpoint
- Deployment lifecycle management
  - `POST /api/deployments/:id/start`
  - `POST /api/deployments/:id/stop`
  - `GET /api/deployments/:id/logs`
- API key management with SHA256 hashing
- Updated web GUI with start/stop buttons, logs viewer, events page
- Docker support with Dockerfile and docker-compose.yml
- GitHub Actions CI/CD workflows
- Pre-commit hooks configuration
- Comprehensive README and documentation

### Changed

- Improved error handling with user-readable messages
- Gateway returns proper OpenAI-compatible error format
- Deployment status lifecycle: planned → deploying → running → stopped/unhealthy/failed

### Fixed

- Gateway now returns 503 instead of 200 for missing deployments
- All failed operations now create event records
- SSH connection errors properly handled and logged

## [0.1.0] - 2024-05-01

### Added

- Initial MVP release
- Machine management (CRUD, SSH test, hardware detection)
- Model catalog (CRUD)
- Deployment management (CRUD)
- OpenAI-compatible gateway endpoints
- Single-page web GUI
- JSON file-based storage
- Hardware detection (OS, CPU, memory, GPU)

[Unreleased]: https://github.com/modelfleet/modelfleet/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/modelfleet/modelfleet/releases/tag/v0.1.0
