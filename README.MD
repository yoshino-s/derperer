# Derperer

![GitHub Release](https://img.shields.io/github/v/release/yoshino-s/derperer) [![goreleaser](https://github.com/yoshino-s/derperer/actions/workflows/publish.yaml/badge.svg)](https://github.com/yoshino-s/derperer/actions/workflows/publish.yaml) ![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/yoshino-s/derperer)

**Language:** [English](README.MD) | [中文](README_zh.md)

Derperer is mother of [derper](https://pkg.go.dev/tailscale.com/cmd/derper) - A tool for managing and testing DERP (Designated Encrypted Relay for Packets) servers, providing discovery, speed testing, and monitoring capabilities.

## Features

- 🔍 **DERP Server Discovery**: Automatically discover and monitor DERP servers using FOFA search engine
- 🚀 **Speed Testing**: Test connection speed and latency to DERP servers
- 📊 **HTTP API**: RESTful API with Swagger documentation for programmatic access
- 📈 **Real-time Monitoring**: Continuous health checking and monitoring of DERP endpoints
- 🇨🇳 **China Region Support**: Special support for China region DERP servers
- 📝 **Flexible Configuration**: Support for YAML, JSON, TOML, and environment variable configuration

## Installation

### From Source

```bash
git clone https://github.com/yoshino-s/derperer.git
cd derperer
go build -o derperer .
```

### Using Go Install

```bash
go install github.com/yoshino-s/derperer@latest
```

### Using Docker

```bash
# Pull and run the latest image
docker run --rm -p 8080:8080 ghcr.io/yoshino-s/derperer:latest serve

# Run speed test
docker run --rm ghcr.io/yoshino-s/derperer:latest speedtest
```

### Using Docker Compose

```bash
# Clone the repository and use docker-compose
git clone https://github.com/yoshino-s/derperer.git
cd derperer
docker-compose up -d
```

## Usage

### Quick Start

```bash
# Run the server
derperer serve

# Run a speed test
derperer speedtest --derp_region_id 1

# Generate configuration file
derperer --generate-config.enable --generate-config.path config.yaml
```

### Commands

#### Main Command

```bash
derperer [command]
```

**Available Commands:**
- `completion` - Generate the autocompletion script for the specified shell
- `help` - Help about any command
- `serve` - Serve runs the HTTP server
- `speedtest` - Run a speed test
- `version` - Show version information

#### Serve Command

Start the HTTP server for DERP server discovery and monitoring:

```bash
derperer serve [flags]
```

**Flags:**
- `--derperer.check_concurrency int` - The number of concurrent tests to run (default 10)
- `--derperer.check_duration duration` - The duration for which to check nodes (default 10s)
- `--derperer.cn` - Only fetch nodes in China
- `--derperer.fetch_limit int` - Limit of FOFA result to fetch (default 100)
- `--derperer.recheck_interval duration` - The interval at which to recheck abandoned nodes (default 10s)
- `--derperer.refetch_interval duration` - The interval at which to fetch data (default 10m0s)
- `--fofa.email string` - FOFA email
- `--fofa.endpoint string` - FOFA endpoint (default "https://fofa.info/api/v1")
- `--fofa.key string` - FOFA key
- `--http.addr string` - HTTP listen address (default ":8080")
- `--http.behind_proxy` - HTTP behind proxy
- `--http.external_url string` - External URL (default "http://127.0.0.1:8080")
- `--http.feature uint16` - HTTP feature (default 30)
- `--http.log` - Enable HTTP log
- `--http.otel` - Enable OpenTelemetry
- `--http.response_trace_id` - Enable x-trace-id in response header

#### Speed Test Command

Run a speed test against DERP servers:

```bash
derperer speedtest [flags]
```

**Flags:**
- `--derp_map_url string` - DERP map URL (default "https://controlplane.tailscale.com/derpmap/default")
- `--derp_region_id int` - DERP region ID
- `--duration duration` - Test duration (default 30s)

### Global Flags

- `--generate-config.enable` - Generate config enable
- `--generate-config.format string` - Generate config format, one of json, yaml, toml, env (default "yaml")
- `--generate-config.path string` - Generate config path
- `--log.file string` - Log file path
- `--log.format string` - Log format, one of json, console, empty for default (console for dev, json for prod)
- `--log.level string` - Log level (default "info")
- `--log.levels.console string` - Log level for console, empty for same as log.level
- `--log.levels.file string` - Log level for file, empty for same as log.level
- `--log.rotate.enable` - Enable log rotation
- `--log.rotate.max_age int` - Max age of log file in days (default 28)
- `--log.rotate.max_backups int` - Max number of log file backups (default 3)
- `--log.rotate.max_size int` - Max size of log file in MB (default 500)

## Configuration

Derperer supports configuration through YAML, JSON, TOML files, or environment variables. The application will automatically search for configuration files in the following locations (in order of priority):

1. Current working directory: `./derperer.yaml`, `./derperer.json`, `./derperer.toml`
2. User config directory: `~/.config/derperer/derperer.yaml`
3. System config directory: `/etc/derperer/derperer.yaml`
4. Environment variables with `DERPERER_` prefix

You can also specify a custom config file using the `--config` flag or generate a sample configuration file:

```bash
# Generate sample configuration
derperer --generate-config.enable --generate-config.path derperer.yaml

# Use custom config file
derperer serve --config /path/to/your/config.yaml
```

Example configuration (`derperer.yaml`):

```yaml
derperer:
  check_duration: 10s
  recheck_interval: 10s
  refetch_interval: 10m0s
  check_concurrency: 10
  cn: false  # Set to true for China region only
  fetch_limit: 100

fofa:
  email: "your-email@example.com"
  key: "your-fofa-key"
  endpoint: "https://fofa.info/api/v1"

http:
  addr: ":8080"
  external_url: "http://127.0.0.1:8080"
  feature: 30
  log: false
  otel: false
  response_trace_id: false

log:
  level: "info"
  format: ""  # empty for default
  file: ""    # empty for stdout
  rotate:
    enable: false
    max_age: 28
    max_backups: 3
    max_size: 500
```

## API Documentation

When running the server, Swagger documentation is available at:
- Swagger UI: `http://localhost:8080/docs/`
- Swagger JSON: `http://localhost:8080/docs/swagger.json`
- Swagger YAML: `http://localhost:8080/docs/swagger.yaml`

## Examples

### Basic Server Start

```bash
# Start server with default settings
derperer serve

# Start server with custom port
derperer serve --http.addr :9090

# Start server with China region only
derperer serve --derperer.cn
```

### Speed Testing

```bash
# Test default region
derperer speedtest

# Test specific region
derperer speedtest --derp_region_id 5

# Test for 60 seconds
derperer speedtest --duration 60s

# Test with custom DERP map
derperer speedtest --derp_map_url https://example.com/derpmap
```

### Configuration Management

```bash
# Generate YAML config
derperer --generate-config.enable --generate-config.format yaml --generate-config.path config.yaml

# Generate JSON config
derperer --generate-config.enable --generate-config.format json --generate-config.path config.json

# Run with config file
derperer serve --config config.yaml
```

### Docker Compose Usage

The included `docker-compose.yml` provides a complete setup for running Derperer:

```bash
# Start the service
docker-compose up -d

# View logs
docker-compose logs -f

# Stop the service
docker-compose down

# Update to latest version
docker-compose pull && docker-compose up -d
```

**Configure FOFA credentials:**

1. Edit the `docker-compose.yml` file and uncomment the FOFA environment variables:
   ```yaml
   environment:
     - DERPERER_FOFA_EMAIL=your-email@example.com
     - DERPERER_FOFA_KEY=your-fofa-key
   ```

2. Or create a `.env` file in the same directory:
   ```env
   DERPERER_FOFA_EMAIL=your-email@example.com
   DERPERER_FOFA_KEY=your-fofa-key
   DERPERER_DERPERER_CN=false
   ```

**Alternative configuration with config file:**

Uncomment the volume mount in `docker-compose.yml` and create a local config file:

```yaml
volumes:
  - ./derperer.yaml:/app/derperer.yaml:ro
```

## Requirements

- Go 1.25.2 or later
- FOFA account (for server discovery features)

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the terms specified in the repository.

## Related Projects

- [Tailscale DERP](https://pkg.go.dev/tailscale.com/cmd/derper) - The original DERP server implementation