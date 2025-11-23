# scripts/migrate.sh
#!/bin/bash

set -e

echo "Tailscale Migration Tool"
echo "========================"

# Build the migration tool
echo "Building migration tool..."
go build -o bin/tailscale-migrate ./cmd/migrate

# Run with provided arguments
./bin/tailscale-migrate "$@"
