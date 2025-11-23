// cmd/migrate/commands.go
package migrate

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"weriKana/internal/migration"
	"github.com/urfave/cli/v2"
)

// runFullMigration performs a complete migration
func runFullMigration(c *cli.Context) error {
	fmt.Println("🚀 Starting comprehensive Headscale to Tailscale migration...")
	fmt.Println("==========================================================")

	// Initialize migrator
	migrator := migration.NewConfigMigrator(
		c.String("headscale-config"),
		c.String("output-dir"),
	).WithDryRun(c.Bool("dry-run")).
		WithForce(c.Bool("force"))

	// Set up logging
	if c.Bool("verbose") {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}

	// Perform migration
	if err := migrator.Migrate(); err != nil {
		return fmt.Errorf("❌ Migration failed: %w", err)
	}

	// Print summary
	printMigrationSummary(migrator, c.String("output-dir"))

	return nil
}

// runConfigMigration migrates only configuration
func runConfigMigration(c *cli.Context) error {
	fmt.Println("⚙️  Migrating Headscale configuration to Tailscale...")

	migrator := migration.NewConfigMigrator(
		c.String("headscale-config"),
		c.String("output-dir"),
	).WithDryRun(c.Bool("dry-run"))

	// Load Headscale config
	headscaleConfig, err := migrator.LoadHeadscaleConfig()
	if err != nil {
		return fmt.Errorf("failed to load Headscale config: %w", err)
	}

	// Convert and save Tailscale config
	tailscaleConfig := migrator.ConvertToTailscaleConfig(headscaleConfig)
	if err := migrator.SaveTailscaleConfig(tailscaleConfig); err != nil {
		return fmt.Errorf("failed to save Tailscale config: %w", err)
	}

	fmt.Printf("✅ Configuration migrated to: %s\n", 
		filepath.Join(c.String("output-dir"), "tailscale-config.json"))

	return nil
}

// runNodesExport exports nodes from Headscale
func runNodesExport(c *cli.Context) error {
	fmt.Println("📋 Exporting nodes from Headscale...")

	migrator := migration.NewConfigMigrator(
		c.String("headscale-config"),
		c.String("output-dir"),
	).WithDryRun(c.Bool("dry-run"))

	// Load Headscale config
	headscaleConfig, err := migrator.LoadHeadscaleConfig()
	if err != nil {
		return fmt.Errorf("failed to load Headscale config: %w", err)
	}

	// Export nodes
	nodes, err := migrator.ExportNodes(headscaleConfig)
	if err != nil {
		return fmt.Errorf("failed to export nodes: %w", err)
	}

	// Generate auth keys instructions
	if err := migrator.GenerateTailscaleAuthKeys(nodes); err != nil {
		return fmt.Errorf("failed to generate auth keys instructions: %w", err)
	}

	fmt.Printf("✅ Exported %d nodes\n", len(nodes))
	fmt.Printf("📁 Node data: %s\n", 
		filepath.Join(c.String("output-dir"), "headscale-nodes-export.json"))
	fmt.Printf("🔑 Auth key instructions: %s\n", 
		filepath.Join(c.String("output-dir"), "tailscale-auth-keys-instructions.txt"))

	return nil
}

// runValidation validates migration readiness
func runValidation(c *cli.Context) error {
	fmt.Println("🔍 Validating migration readiness...")

	migrator := migration.NewConfigMigrator(
		c.String("headscale-config"),
		c.String("output-dir"),
	)

	// Check if Headscale config exists and is valid
	if _, err := migrator.LoadHeadscaleConfig(); err != nil {
		return fmt.Errorf("❌ Headscale config validation failed: %w", err)
	}
	fmt.Println("✅ Headscale configuration is valid")

	// Check output directory
	outputDir := c.String("output-dir")
	if _, err := os.Stat(outputDir); err == nil {
		fmt.Printf("⚠️  Output directory %s already exists\n", outputDir)
		if !c.Bool("force") {
			fmt.Printf("💡 Use --force to overwrite or choose a different --output-dir\n")
		}
	} else {
		fmt.Println("✅ Output directory is available")
	}

	// Check for required tools/access
	fmt.Println("✅ Basic validation passed")
	fmt.Println("\n📋 Next steps:")
	fmt.Println("   1. Ensure you have Tailscale admin access")
	fmt.Println("   2. Prepare auth keys in Tailscale admin console")
	fmt.Println("   3. Run 'tailscale-migrate full' to start migration")

	return nil
}

// runReport generates a migration report
func runReport(c *cli.Context) error {
	fmt.Println("📊 Generating migration report...")

	migrator := migration.NewConfigMigrator(
		c.String("headscale-config"),
		c.String("output-dir"),
	)

	if err := migrator.GenerateMigrationReport(); err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}

	reportPath := filepath.Join(c.String("output-dir"), "migration-report.json")
	fmt.Printf("✅ Migration report generated: %s\n", reportPath)
	fmt.Println("\n📋 Open the report to see:")
	fmt.Println("   - Migration statistics")
	fmt.Println("   - Next steps")
	fmt.Println("   - Generated files")
	fmt.Println("   - Warnings and recommendations")

	return nil
}

// runCleanup cleans up migration artifacts
func runCleanup(c *cli.Context) error {
	fmt.Println("🧹 Cleaning up migration artifacts...")

	migrator := migration.NewConfigMigrator(
		c.String("headscale-config"),
		c.String("output-dir"),
	).WithDryRun(c.Bool("dry-run"))

	if c.Bool("dry-run") {
		fmt.Printf("[DRY RUN] Would clean up: %s\n", c.String("output-dir"))
		return nil
	}

	if err := migrator.Cleanup(); err != nil {
		return fmt.Errorf("failed to clean up: %w", err)
	}

	fmt.Printf("✅ Cleaned up: %s\n", c.String("output-dir"))
	return nil
}

// runStatus shows migration status
func runStatus(c *cli.Context) error {
	fmt.Println("📈 Migration Status")
	fmt.Println("==================")

	outputDir := c.String("output-dir")
	
	files := []struct {
		name string
		desc string
	}{
		{"tailscale-config.json", "Tailscale configuration"},
		{"headscale-nodes-export.json", "Exported nodes data"},
		{"tailscale-auth-keys-instructions.txt", "Auth key setup instructions"},
		{"tailscale-acl-migration.md", "ACL migration guide"},
		{"migration-report.json", "Comprehensive migration report"},
	}

	allExist := true
	for _, file := range files {
		path := filepath.Join(outputDir, file.name)
		if _, err := os.Stat(path); err == nil {
			fmt.Printf("✅ %s: %s\n", file.desc, path)
		} else {
			fmt.Printf("❌ %s: MISSING\n", file.desc)
			allExist = false
		}
	}

	fmt.Println()
	if allExist {
		fmt.Println("🎉 Migration appears complete!")
		fmt.Println("📋 Next: Set up Tailscale with the generated configuration")
	} else {
		fmt.Println("🚧 Migration incomplete")
		fmt.Println("💡 Run 'tailscale-migrate full' to complete migration")
	}

	return nil
}

// printMigrationSummary prints a beautiful migration summary
func printMigrationSummary(migrator *migration.ConfigMigrator, outputDir string) {
	stats := migrator.Stats
	
	fmt.Println()
	fmt.Println("🎉 MIGRATION COMPLETE!")
	fmt.Println("======================")
	fmt.Printf("📊 Statistics:\n")
	fmt.Printf("   • Total nodes processed: %d\n", stats.TotalNodes)
	fmt.Printf("   • Configuration migrated: %t\n", stats.ConfigMigrated)
	fmt.Printf("   • Nodes exported: %t\n", stats.NodesExported)
	fmt.Printf("   • ACL migration guide: %t\n", stats.ACLsMigrated)
	fmt.Printf("   • Duration: %s\n", stats.Duration)
	
	fmt.Printf("\n📁 Generated Files:\n")
	files, _ := filepath.Glob(filepath.Join(outputDir, "*"))
	for _, file := range files {
		info, err := os.Stat(file)
		if err == nil {
			fmt.Printf("   • %s (%d bytes)\n", filepath.Base(file), info.Size())
		}
	}

	fmt.Printf("\n🚀 Next Steps:\n")
	fmt.Printf("   1. Review: %s/migration-report.json\n", outputDir)
	fmt.Printf("   2. Create auth keys using: %s/tailscale-auth-keys-instructions.txt\n", outputDir)
	fmt.Printf("   3. Migrate ACLs using: %s/tailscale-acl-migration.md\n", outputDir)
	fmt.Printf("   4. Configure Tailscale with: %s/tailscale-config.json\n", outputDir)
	fmt.Printf("   5. Test connectivity and decommission Headscale\n")
	
	fmt.Printf("\n💡 Pro Tip: Set environment variables for easy Tailscale setup:\n")
	fmt.Printf("   export TAILSCALE_STATE_DIR=%s/tailscale-state\n", outputDir)
	fmt.Printf("   export USE_TAILSCALE=true\n")
}
