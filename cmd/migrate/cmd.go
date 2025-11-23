// cmd/migrate/main.go
package migrate

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"weriKana/internal/migration"
	"github.com/urfave/cli/v2"
)

var (
	version = "1.0.0"
	commit  = "dev"
)

func main() {
	app := &cli.App{
		Name:    "tailscale-migrate",
		Version: version,
		Usage:   "Migrate from Headscale to Tailscale",
		Description: `Comprehensive migration tool to transition from Headscale to Tailscale.
This tool helps migrate configuration, nodes, and settings while providing
detailed reports and guidance for the migration process.`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "headscale-config",
				Aliases: []string{"c"},
				Value:   "/etc/headscale/config.yaml",
				Usage:   "Path to Headscale configuration file",
				EnvVars: []string{"HEADSCALE_CONFIG_PATH"},
			},
			&cli.StringFlag{
				Name:    "output-dir",
				Aliases: []string{"o"},
				Value:   "./tailscale-migration",
				Usage:   "Output directory for migration artifacts",
				EnvVars: []string{"MIGRATION_OUTPUT_DIR"},
			},
			&cli.BoolFlag{
				Name:    "dry-run",
				Aliases: []string{"n"},
				Value:   false,
				Usage:   "Perform a dry run without making changes",
			},
			&cli.BoolFlag{
				Name:    "force",
				Aliases: []string{"f"},
				Value:   false,
				Usage:   "Force migration even if validation fails",
			},
			&cli.BoolFlag{
				Name:  "verbose",
				Aliases: []string{"v"},
				Value: false,
				Usage: "Enable verbose logging",
			},
		},
		Commands: []*cli.Command{
			{
				Name:    "full",
				Aliases: []string{"f"},
				Usage:   "Perform full migration (config, nodes, ACLs)",
				Action:  runFullMigration,
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "export-nodes",
						Value: true,
						Usage: "Export nodes from Headscale",
					},
					&cli.BoolFlag{
						Name:  "generate-acls",
						Value: true,
						Usage: "Generate ACL migration guide",
					},
					&cli.BoolFlag{
						Name:  "validate",
						Value: true,
						Usage: "Validate migration results",
					},
				},
			},
			{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "Migrate only configuration",
				Action:  runConfigMigration,
			},
			{
				Name:    "nodes",
				Aliases: []string{"n"},
				Usage:   "Export nodes only",
				Action:  runNodesExport,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "format",
						Value: "json",
						Usage: "Output format (json, yaml, csv)",
					},
					&cli.StringFlag{
						Name:  "database",
						Usage: "Headscale database path (if different from config)",
					},
				},
			},
			{
				Name:    "validate",
				Aliases: []string{"v"},
				Usage:   "Validate migration readiness",
				Action:  runValidation,
			},
			{
				Name:    "report",
				Aliases: []string{"r"},
				Usage:   "Generate migration report from existing data",
				Action:  runReport,
			},
			{
				Name:    "cleanup",
				Aliases: []string{"x"},
				Usage:   "Clean up migration artifacts",
				Action:  runCleanup,
			},
			{
				Name:    "status",
				Aliases: []string{"s"},
				Usage:   "Show migration status",
				Action:  runStatus,
			},
		},
		Action: runFullMigration, // Default action
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
