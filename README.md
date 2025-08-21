# Elasticsearch Log Trimmer

I got tired of manually cleaning up old Elasticsearch indexes, so I built this tool to automate the process. It analyzes your cluster, figures out which indexes are taking up space or getting old, and can delete them safely.

The tool has nice colored output and structured JSON logging, so it works well both for interactive use and automated deployments. By default it just shows you what it would delete (dry run mode), so you won't accidentally nuke anything important.

## Installation

Clone this repo and build it with Go 1.21+:

```bash
git clone <this-repo>
cd log-trimmer
make build
```

Or if you want to build for multiple platforms:

```bash
make build-all
```

The binary ends up in `./build/log-trimmer`.

## Quick Start

First, do a dry run to see what would happen:

```bash
./build/log-trimmer --host https://localhost:9200 --max-age 7d --pattern "logs-*"
```

If the results look good, add the `--delete-indexes` flag to actually delete stuff:

```bash
./build/log-trimmer --host https://localhost:9200 --max-age 7d --pattern "logs-*" --delete-indexes
```

You can also limit by total size instead of age:

```bash
./build/log-trimmer --host https://localhost:9200 --max-size 100GB --pattern "logs-*" --delete-indexes
```

## Configuration

The tool accepts both command line flags and environment variables. Environment variables are handy for containerized deployments.

### Command Line Options

- `--host` - Elasticsearch URL (required)
- `--username` - Username for auth (optional)
- `--password` - Password for auth (optional)
- `--max-age` - Keep indexes newer than this (e.g., `7d`, `24h`, `30d`)
- `--max-size` - Keep total size under this limit (e.g., `50GB`, `1TB`)
- `--pattern` - Index pattern to match (default: `vector-*`)
- `--delete-indexes` - Actually delete stuff (default: false)
- `--verbose` - More output
- `--skip-tls` - Skip TLS verification (default: true)
- `--log-level` - Set log level (`debug`, `info`, `warn`, `error`)
- `--log-format` - Output format (`console` or `json`)
- `--log-file` - Write logs to file

### Environment Variables

Set these instead of (or in addition to) command line flags:

- `ES_HOST` - Elasticsearch URL
- `ES_USERNAME` - Username
- `ES_PASSWORD` - Password
- `MAX_AGE` - Maximum age
- `MAX_SIZE` - Maximum total size
- `INDEX_PATTERN` - Index pattern
- `DELETE_INDEXES` - Set to `true` to actually delete
- `LOG_LEVEL` - Log level
- `LOG_FORMAT` - Log format
- `LOG_FILE` - Log file path

Example with environment variables:

```bash
export ES_HOST="https://elasticsearch.company.com:9200"
export ES_USERNAME="admin"
export MAX_AGE="30d"
export INDEX_PATTERN="application-logs-*"
./build/log-trimmer --delete-indexes
```

## How Deletion Works

The tool uses a simple strategy: it sorts indexes by creation date (oldest first) and applies your retention rules.

If you specify `--max-age`, any index older than that gets marked for deletion.

If you specify `--max-size`, it calculates the total size of all matching indexes. If that's over your limit, it marks the oldest indexes for deletion until the total would be under the limit.

You can use both rules together. The tool will delete anything that violates either rule.

## Logging

I added structured logging because it's useful for production deployments. You get two output modes:

Console mode gives you nice colored output for interactive use:

```
[INFO] [elasticsearch] connect: Connecting to Elasticsearch cluster
[SUCCESS] [configuration] validate: Configuration validated successfully
[WARN] [analysis] deletion_plan: DELETION PLAN: 3 indexes selected for deletion
```

JSON mode gives you machine-readable logs:

```json
{
  "timestamp": "2025-08-21T12:00:01-06:00",
  "level": "info",
  "message": "Starting Elasticsearch Log Trimmer",
  "component": "application",
  "operation": "startup",
  "service": "log-trimmer",
  "version": "1.0.0"
}
```

Set `--log-format json` for JSON output, or use `--log-file` to write structured logs to disk while keeping console output readable.

## Docker Usage

There's a Dockerfile included. Build the image:

```bash
make docker-build
```

Run it with environment variables:

```bash
docker run --rm \
  -e ES_HOST="https://elasticsearch:9200" \
  -e MAX_AGE="7d" \
  -e INDEX_PATTERN="logs-*" \
  company/log-trimmer:latest --delete-indexes
```

## Development

The project follows standard Go conventions with a few extra directories:

- `cmd/log-trimmer/` - Main application
- `internal/config/` - Configuration handling
- `internal/elasticsearch/` - Elasticsearch client
- `internal/logger/` - Structured logging
- `pkg/utils/` - Utility functions

Use the Makefile for common tasks:

```bash
make help          # Show available commands
make build         # Build the app
make test          # Run tests
make lint          # Run linter
make clean         # Clean up build artifacts
```

For development, run `make dev-setup` to install useful tools like `golangci-lint`.

## Safety Features

The tool defaults to dry-run mode, so you have to explicitly pass `--delete-indexes` to actually delete anything.

It shows you exactly what it plans to delete before doing anything, including the reason (age limit, size limit, or both).

If individual deletions fail, it continues with the remaining indexes and gives you a summary of what worked and what didn't.

## Size and Age Formats

Sizes use standard units: `500MB`, `50GB`, `1TB`, etc.

Ages use simple time units: `7d` (days), `24h` (hours), `30m` (minutes), `1w` (weeks).

The tool is pretty flexible about formats, so `1.5GB` and `90m` work fine.

## Examples

Clean up old application logs:

```bash
./build/log-trimmer \
  --host https://elk.company.com:9200 \
  --username elastic \
  --pattern "application-logs-*" \
  --max-age 90d \
  --delete-indexes
```

Limit total log storage to 500GB:

```bash
./build/log-trimmer \
  --host https://elasticsearch:9200 \
  --pattern "logs-*" \
  --max-size 500GB \
  --delete-indexes
```

Use in a cron job with JSON logging:

```bash
./build/log-trimmer \
  --host https://elasticsearch:9200 \
  --max-age 30d \
  --log-format json \
  --log-file /var/log/log-trimmer.log \
  --delete-indexes
```

## License

MIT - do whatever you want with it.