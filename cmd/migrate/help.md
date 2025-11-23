# Make script executable
chmod +x scripts/migrate.sh

# Full migration
./scripts/migrate.sh full --headscale-config /etc/headscale/config.yaml --output-dir ./migration

# Dry run to see what would happen
./scripts/migrate.sh --dry-run --verbose

# Only export nodes
./scripts/migrate.sh nodes --output-dir ./node-export

# Validate migration readiness
./scripts/migrate.sh validate

# Check status of existing migration
./scripts/migrate.sh status --output-dir ./migration

# Clean up migration artifacts
./scripts/migrate.sh cleanup --output-dir ./migration

func main() {
	// Check if migration command is requested
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		// Delegate to migration command
		os.Args = append([]string{os.Args[0]}, os.Args[2:]...)
		mainMigrate()
		return
	}

	// Normal server startup
	ctx := appcontext.NewContext()
	bankd.StartSSHServer(ctx, "/path/to/git/repo")
}

// mainMigrate is the entry point for migration commands
func mainMigrate() {
	// This would be in cmd/migrate/main.go
	// For now, we'll use the separate binary approach
	log.Println("Use './scripts/migrate.sh' for migration commands")
}
