# Nimbus - Function as a Service (FaaS) Engine

A lightweight, serverless computing platform written in Go that allows you to deploy, manage, and invoke containerized functions via a simple CLI and HTTP gateway.

## Project Overview

**Nimbus** is a simplified **Function as a Service (FaaS)** platform similar to AWS Lambda. It enables developers to:

- 🚀 Deploy serverless functions with a single CLI command
- 🐳 Run functions in isolated Docker containers
- ⚡ Invoke functions via HTTP endpoints
- 📅 Schedule function execution with cron-style scheduling
- 📊 Stream function logs in real-time
- 🔌 Support multiple runtimes (Node.js, Python, Go)

### Core Features

- **CLI Tool:** Deploy, invoke, manage, and delete functions from the command line
- **HTTP Gateway:** Routes requests to function containers with load balancing
- **Container Management:** Automatic Docker container lifecycle management
- **Persistent Storage:** SQLite database for function metadata and invocation logs
- **Cron Scheduler:** Schedule functions to run at specified intervals
- **Real-time Logging:** Stream and view function execution logs

### Architecture

The system consists of three main components:

1. **CLI (Client):** Command-line tool for deploying and managing functions
2. **API Gateway:** HTTP server that routes requests and manages deployments
3. **Runtime Manager:** Manages Docker containers and function execution


---

## Installation

### Prerequisites

Before installing Nimbus, ensure you have the following installed:

- **Go 1.25.6+** — [Download](https://golang.org/dl/)
- **Docker** — [Download](https://www.docker.com/products/docker-desktop)
- **Git** — [Download](https://git-scm.com/)

### Quick Install

#### Linux & macOS

Download and run the installer script:

```bash
curl -fsSL https://raw.githubusercontent.com/infraspecdev/faas-engine-go/main/installer/install.sh | bash
```

Or specify a version:

```bash
curl -fsSL https://raw.githubusercontent.com/infraspecdev/faas-engine-go/main/installer/install.sh | bash -s v1.0.0
```

Verify installation:

```bash
nimbus --version
```

#### Windows

Download the latest release from [GitHub Releases](https://github.com/infraspecdev/faas-engine-go/releases) and add it to your PATH.

### Build from Source

```bash
# Clone the repository
git clone https://github.com/infraspecdev/faas-engine-go.git
cd faas-engine-go

# Build the CLI
go build -o nimbus ./cmd/cli

# Build the gateway (API server)
go build -o nimbus-gateway ./cmd/proxy

# Build the runtime manager
go build -o nimbus-runtime ./cmd/runtime-manager
```

---

## Configuration

### Environment Variables

Create a `.env` file in your project root or set environment variables:

```env
# Proxy/Gateway Configuration
PROXY_URL=http://localhost:8080
PROXY_PORT=8080

# Runtime Manager Configuration
RUNTIME_PORT=9090

# Database Configuration
DB_PATH=./data/nimbus.db

# Docker Configuration
DOCKER_HOST=unix:///var/run/docker.sock
```

### Setting PROXY_URL

The `PROXY_URL` environment variable specifies the address where the FaaS gateway is running. This is used by the CLI to communicate with the server.

#### On Linux/macOS

```bash
# Temporary (current session only)
export PROXY_URL=http://localhost:8080

# Permanent (add to ~/.bash_profile or ~/.zshrc)
echo 'export PROXY_URL=http://localhost:8080' >> ~/.bash_profile
source ~/.bash_profile
```

#### On Windows (PowerShell)

```powershell
# Temporary (current session only)
$env:PROXY_URL = "http://localhost:8080"

# Permanent (system-wide)
setx PROXY_URL "http://localhost:8080"

# Verify
echo $env:PROXY_URL
```

#### Using .env File

Create a `.env` file in your working directory:

```env
PROXY_URL=http://localhost:8080
PROXY_PORT=8080
```

Load it in your shell:

```bash
source .env  # Linux/macOS
# or
Get-Content .env | ForEach-Object { $_.split('=') | % { if ($_.length -eq 2) { [Environment]::SetEnvironmentVariable($_[0], $_[1]) } } }  # PowerShell
```

---

## Getting Started

### 1. Start the Proxy Gateway

```bash
./cmd/proxy
```

The Proxy API will be available at `http://localhost:80`

### 2. Start the Runtime Manager

In another terminal:

```bash
./cmd/runtime-manager
```

### 3. Configure the CLI

Set the `PROXY_URL` to point to your gateway:

```bash
export PROXY_URL=http://localhost:8080  # Linux/macOS
# or
setx PROXY_URL http://localhost:8080    # Windows
```

### 4. Deploy a Function

```bash
nimbus deploy ./samples/hello --name hello-func --runtime node
```

### 5. Invoke the Function

```bash
# Via CLI
nimbus invoke hello-func --data "{}"

# Via HTTP
curl http://localhost:8080/hello-func
```

### 6. View Logs

```bash
nimbus logs hello-func
```

### 7. List All Functions

```bash
nimbus list
```

---

## CLI Commands

| Command | Description |
|---------|-------------|
| `nimbus deploy <path>` | Deploy a function from a directory |
| `nimbus invoke <name>` | Invoke a function |
| `nimbus list` | List all deployed functions |
| `nimbus logs <name>` | Stream function logs |
| `nimbus delete <name>` | Delete a function |
| `nimbus schedule <name>` | Schedule a function |
| `nimbus --help` | Show help menu |

---

## Project Structure

```
faas-engine-go/
├── cmd/
│   ├── cli/              # Command-line client
│   ├── proxy/            # HTTP Gateway/API server
│   └── runtime-manager/  # Container runtime manager
├── internal/
│   ├── api/              # API handlers
│   ├── service/          # Business logic
│   ├── sqlite/           # Database layer
│   ├── sdk/              # Docker SDK wrapper
│   └── types/            # Type definitions
├── samples/              # Example functions
├── runtimes/             # Runtime Docker images
└── installer/            # Installation scripts
```

---

## Example Functions

The project includes sample functions in the `samples/` directory:

### Node.js Example

```javascript
// samples/echo/index.js
module.exports = (event) => {
  return {
    statusCode: 200,
    body: JSON.stringify({ message: event.body || "Hello!" })
  };
};
```

### Python Example

```python
# samples/calculator/index.py
def handler(event, context):
    return {
        "statusCode": 200,
        "body": event.get("body", "Hello from Python!")
    }
```

### Go Example

```go
// samples/hello/main.go
package main

import (
	"fmt"
	"log"
	"net/http"
)

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello from Go!")
}

func main() {
	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
```