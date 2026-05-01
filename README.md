# ModelFleet

> **ModelFleet — deploy models anywhere, use them from one API.**

ModelFleet is a lightweight control plane for managing LLM serving across machines you control. Deploy models to remote servers via SSH, manage them through a web GUI, and expose them through a single OpenAI-compatible API endpoint.

## Features

- **Machine Management**: Add remote machines via SSH with key or password authentication
- **Hardware Detection**: Automatically detect OS, CPU, memory, GPU capabilities
- **Model Catalog**: Track GGUF models with metadata (size, quantization, context length)
- **Remote Deployment**: Deploy models to remote machines with automatic directory setup
- **OpenAI-Compatible Gateway**: Access all models through a single `/v1/chat/completions` endpoint
- **Health Monitoring**: Automatic health checks every 30 seconds
- **Event Logging**: Structured logging of all operations
- **Web GUI**: Single-page interface for managing everything

## Quick Start

```bash
# Clone the repository
git clone https://github.com/modelfleet/modelfleet.git
cd modelfleet

# Build
go build -o modelfleet .

# Run
./modelfleet
# Server starts on http://localhost:3456
```

Open your browser to `http://localhost:3456` to access the web GUI.

## Usage

### 1. Add a Remote Machine

Via web GUI or API:
```bash
curl -X POST http://localhost:3456/api/machines \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-server",
    "host": "192.168.1.100",
    "ssh_port": 22,
    "ssh_user": "admin",
    "auth_type": "key",
    "auth_private_key_path": "~/.ssh/id_rsa"
  }'
```

### 2. Test SSH Connection

```bash
curl -X POST http://localhost:3456/api/machines/{id}/test-ssh
```

### 3. Detect Hardware

```bash
curl -X POST http://localhost:3456/api/machines/{id}/detect
```

### 4. Add a Model

```bash
curl -X POST http://localhost:3456/api/models \
  -H "Content-Type: application/json" \
  -d '{
    "display_name": "Qwen2.5 7B Instruct",
    "filename": "qwen2.5-7b-instruct-q4_k_m.gguf",
    "source_url": "https://huggingface.co/...",
    "format": "GGUF",
    "parameter_size": "7B"
  }'
```

### 5. Create a Deployment

```bash
curl -X POST http://localhost:3456/api/deployments \
  -H "Content-Type: application/json" \
  -d '{
    "name": "qwen-local",
    "machine_id": "{machine-id}",
    "model_id": "{model-id}",
    "backend": "cpu",
    "port": 8081
  }'
```

### 6. Start the Deployment

```bash
curl -X POST http://localhost:3456/api/deployments/{id}/start
```

### 7. Use the API

```bash
# List available models
curl http://localhost:3456/v1/models

# Chat completion
curl http://localhost:3456/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "local/qwen-local",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

## Architecture

```
Client Apps
  │
  │ OpenAI-compatible API
  v
ModelFleet Control Plane (Port 3456)
  ├── Web GUI (Single-page app)
  ├── REST API (/api/*)
  └── OpenAI Gateway (/v1/*)
  │
  │ SSH / HTTP health checks
  v
Remote Machines
  │
  │ llama-server
  v
Deployed Models
```

## Configuration

Environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `3456` | Server port |
| `DATA_DIR` | `./data` | Data directory for JSON storage |

## Development

```bash
# Run in development mode
go run main.go

# Run tests
go test ./...

# Build for production
go build -ldflags "-s -w" -o modelfleet .

# Cross-compile
GOOS=linux GOARCH=amd64 go build -o modelfleet-linux-amd64
GOOS=darwin GOARCH=arm64 go build -o modelfleet-darwin-arm64
```

## Prerequisites

- **Control Machine**: Go 1.21+
- **Remote Machines**: SSH access, Linux/macOS
- **LLM Runtime**: llama-server binary on remote machines
- **Model Files**: GGUF format models

## API Endpoints

### Internal API
- `GET /health` - Health check
- `GET/POST /api/machines` - Machine management
- `POST /api/machines/:id/test-ssh` - Test SSH connection
- `POST /api/machines/:id/detect` - Hardware detection
- `GET/POST /api/models` - Model catalog
- `GET/POST /api/deployments` - Deployments
- `POST /api/deployments/:id/start` - Start deployment
- `POST /api/deployments/:id/stop` - Stop deployment
- `GET /api/deployments/:id/logs` - View logs
- `GET /api/events` - Recent events

### Gateway API
- `GET /v1/models` - List available models
- `POST /v1/chat/completions` - Chat completions

## Security

- SSH private key paths stored (not key material)
- No automatic GPU driver installation
- No forced reboots
- No invasive system modifications
- Gateway endpoints require API key (configurable)

## License

MIT License

## Contributing

Contributions welcome! Please ensure:
- Code passes `gofmt` formatting
- Tests pass: `go test ./...`
- Pre-commit hooks pass: `pre-commit run --all-files`
