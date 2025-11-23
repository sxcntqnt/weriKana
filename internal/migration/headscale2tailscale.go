package migration

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
	"tailscale.com/types/key"
)

// HeadscaleConfig represents the Headscale configuration structure
type HeadscaleConfig struct {
	ServerURL          string   `yaml:"server_url" json:"server_url"`
	ListenAddr         string   `yaml:"listen_addr" json:"listen_addr"`
	MetricsAddr        string   `yaml:"metrics_addr" json:"metrics_addr"`
	GrpcListenAddr     string   `yaml:"grpc_listen_addr" json:"grpc_listen_addr"`
	PrivateKeyPath     string   `yaml:"private_key_path" json:"private_key_path"`
	EphemeralNodeInactivityTimeout string `yaml:"ephemeral_node_inactivity_timeout" json:"ephemeral_node_inactivity_timeout"`
	DBType             string   `yaml:"db_type" json:"db_type"`
	DBPath             string   `yaml:"db_path" json:"db_path"`
	IPPrefixes         []string `yaml:"ip_prefixes" json:"ip_prefixes"`
	NoisePrivateKeyPath string  `yaml:"noise_private_key_path" json:"noise_private_key_path"`
	DERP                struct {
		Server struct {
			Enabled    bool   `yaml:"enabled" json:"enabled"`
			RegionID   int    `yaml:"region_id" json:"region_id"`
			RegionCode string `yaml:"region_code" json:"region_code"`
			RegionName string `yaml:"region_name" json:"region_name"`
		} `yaml:"server" json:"server"`
		URLs []string `yaml:"urls" json:"urls"`
	} `yaml:"derp" json:"derp"`
}

// HeadscaleNode represents a node from Headscale
type HeadscaleNode struct {
	ID          uint64 `json:"id"`
	MachineKey  string `json:"machine_key"`
	NodeKey     string `json:"node_key"`
	DiscoKey    string `json:"disco_key"`
	IPAddresses []string `json:"ip_addresses"`
	Name        string `json:"name"`
	User        string `json:"user"`
	LastSeen    string `json:"last_seen"`
	Expiry      string `json:"expiry"`
	CreatedAt   string `json:"created_at"`
}

// TailscaleConfig represents Tailscale configuration
type TailscaleConfig struct {
	AuthKey     string   `json:"authKey,omitempty"`
	ControlURL  string   `json:"controlUrl,omitempty"`
	Ephemeral   bool     `json:"ephemeral,omitempty"`
	Hostname    string   `json:"hostname,omitempty"`
	StateDir    string   `json:"stateDir,omitempty"`
	Socket      string   `json:"socket,omitempty"`
}

// MigrationStats tracks migration statistics
type MigrationStats struct {
	TotalNodes    int `json:"total_nodes"`
	MigratedNodes int `json:"migrated_nodes"`
	FailedNodes   int `json:"failed_nodes"`
}

// ConfigMigrator handles configuration migration
type ConfigMigrator struct {
	HeadscaleConfigPath string
	TailscaleConfigPath string
	OutputDir          string
	Stats              MigrationStats
}

// NewConfigMigrator creates a new migrator instance
func NewConfigMigrator(headscaleConfigPath, outputDir string) *ConfigMigrator {
	return &ConfigMigrator{
		HeadscaleConfigPath: headscaleConfigPath,
		OutputDir:          outputDir,
		TailscaleConfigPath: filepath.Join(outputDir, "tailscale-config.json"),
	}
}

// LoadHeadscaleConfig loads Headscale configuration from YAML file
func (m *ConfigMigrator) LoadHeadscaleConfig() (*HeadscaleConfig, error) {
	data, err := os.ReadFile(m.HeadscaleConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read headscale config: %w", err)
	}

	var config HeadscaleConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse headscale config: %w", err)
	}

	return &config, nil
}

// ConvertToTailscaleConfig converts Headscale config to Tailscale compatible format
func (m *ConfigMigrator) ConvertToTailscaleConfig(headscaleConfig *HeadscaleConfig) *TailscaleConfig {
	tailscaleConfig := &TailscaleConfig{
		// Use the Headscale server URL as control URL
		ControlURL: headscaleConfig.ServerURL,
		// Set state directory
		StateDir: filepath.Join(m.OutputDir, "tailscale-state"),
		// Set socket path
		Socket: filepath.Join(m.OutputDir, "tailscale.sock"),
	}

	return tailscaleConfig
}

// SaveTailscaleConfig saves Tailscale configuration to file
func (m *ConfigMigrator) SaveTailscaleConfig(config *TailscaleConfig) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tailscale config: %w", err)
	}

	if err := os.WriteFile(m.TailscaleConfigPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write tailscale config: %w", err)
	}

	log.Printf("Tailscale configuration saved to: %s", m.TailscaleConfigPath)
	return nil
}

// GenerateMigrationReport generates a migration report
func (m *ConfigMigrator) GenerateMigrationReport() error {
	report := struct {
		MigrationStats MigrationStats `json:"migration_stats"`
		OutputFiles    []string       `json:"output_files"`
		NextSteps      []string       `json:"next_steps"`
	}{
		MigrationStats: m.Stats,
		OutputFiles: []string{
			m.TailscaleConfigPath,
		},
		NextSteps: []string{
			"1. Install Tailscale on your nodes",
			"2. Use the generated configuration with Tailscale",
			"3. Update your DNS/DERP settings if needed",
			"4. Test connectivity between nodes",
		},
	}

	reportPath := filepath.Join(m.OutputDir, "migration-report.json")
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal migration report: %w", err)
	}

	if err := os.WriteFile(reportPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write migration report: %w", err)
	}

	log.Printf("Migration report saved to: %s", reportPath)
	return nil
}

// ExportNodes exports node information from Headscale (simplified version)
func (m *ConfigMigrator) ExportNodes(headscaleConfig *HeadscaleConfig) ([]HeadscaleNode, error) {
	// This is a simplified implementation. In a real scenario, you would:
	// 1. Connect to Headscale database
	// 2. Query nodes table
	// 3. Export node information
	
	log.Println("Note: Node export requires database access. Please export nodes manually from Headscale.")
	
	// Placeholder for actual implementation
	nodes := []HeadscaleNode{}
	
	// Export nodes to JSON file for reference
	nodesPath := filepath.Join(m.OutputDir, "headscale-nodes-export.json")
	nodesData, _ := json.MarshalIndent(nodes, "", "  ")
	os.WriteFile(nodesPath, nodesData, 0644)
	
	return nodes, nil
}

// GenerateTailscaleAuthKeys generates Tailscale auth keys configuration
func (m *ConfigMigrator) GenerateTailscaleAuthKeys(nodes []HeadscaleNode) error {
	// This would generate pre-auth keys for Tailscale
	// In practice, you'd create these via Tailscale admin console or API
	
	authKeysPath := filepath.Join(m.OutputDir, "tailscale-auth-keys.txt")
	instructions := []string{
		"# Tailscale Auth Keys Setup",
		"# ==========================",
		"",
		"1. Go to Tailscale Admin Console: https://login.tailscale.com/admin/authkeys",
		"2. Create new auth keys for your nodes",
		"3. Use ephemeral keys if you had ephemeral nodes in Headscale",
		"4. Configure appropriate expiry times",
		"5. Use the generated keys in your Tailscale configuration",
		"",
		"Note: You cannot directly migrate Headscale keys to Tailscale.",
	}

	content := strings.Join(instructions, "\n")
	if err := os.WriteFile(authKeysPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write auth keys instructions: %w", err)
	}

	log.Printf("Auth keys instructions saved to: %s", authKeysPath)
	return nil
}

// Migrate performs the complete migration process
func (m *ConfigMigrator) Migrate() error {
	log.Println("Starting migration from Headscale to Tailscale...")

	// Load Headscale configuration
	headscaleConfig, err := m.LoadHeadscaleConfig()
	if err != nil {
		return fmt.Errorf("failed to load headscale config: %w", err)
	}

	// Create output directory
	if err := os.MkdirAll(m.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Convert configuration
	tailscaleConfig := m.ConvertToTailscaleConfig(headscaleConfig)

	// Save Tailscale configuration
	if err := m.SaveTailscaleConfig(tailscaleConfig); err != nil {
		return fmt.Errorf("failed to save tailscale config: %w", err)
	}

	// Export nodes (placeholder - requires database access)
	nodes, _ := m.ExportNodes(headscaleConfig)
	m.Stats.TotalNodes = len(nodes)

	// Generate auth keys instructions
	if err := m.GenerateTailscaleAuthKeys(nodes); err != nil {
		return fmt.Errorf("failed to generate auth keys instructions: %w", err)
	}

	// Generate migration report
	if err := m.GenerateMigrationReport(); err != nil {
		return fmt.Errorf("failed to generate migration report: %w", err)
	}

	log.Println("Migration completed successfully!")
	log.Printf("Total nodes processed: %d", m.Stats.TotalNodes)
	log.Printf("Output directory: %s", m.OutputDir)

	return nil
}

// Utility functions

// ValidateHeadscaleConfig validates Headscale configuration file
func ValidateHeadscaleConfig(configPath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var config HeadscaleConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("invalid headscale config format: %w", err)
	}

	// Basic validation
	if config.ServerURL == "" {
		return fmt.Errorf("server_url is required in headscale config")
	}

	if config.DBPath == "" && config.DBType == "sqlite3" {
		return fmt.Errorf("db_path is required for sqlite3 database")
	}

	return nil
}

// CleanupOldState cleans up old Headscale state if needed
func CleanupOldState(stateDir string) error {
	if _, err := os.Stat(stateDir); os.IsNotExist(err) {
		return nil // Directory doesn't exist
	}

	// Create backup of old state
	backupDir := stateDir + ".backup"
	if err := os.Rename(stateDir, backupDir); err != nil {
		return fmt.Errorf("failed to backup old state: %w", err)
	}

	log.Printf("Old state backed up to: %s", backupDir)
	return nil
}

// GenerateSystemdService generates a systemd service file for Tailscale
func GenerateSystemdService(outputDir, user string) error {
	serviceContent := fmt.Sprintf(`[Unit]
Description=Tailscale client
After=network.target

[Service]
Type=notify
User=%s
ExecStart=/usr/bin/tailscaled --state=%s/tailscale-state
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
`, user, outputDir)

	servicePath := filepath.Join(outputDir, "tailscale.service")
	if err := os.WriteFile(servicePath, []byte(serviceContent), 0644); err != nil {
		return fmt.Errorf("failed to write systemd service file: %w", err)
	}

	log.Printf("Systemd service file generated: %s", servicePath)
	return nil
}

// PrintMigrationInstructions prints helpful migration instructions
func PrintMigrationInstructions() {
	instructions := `
Migration from Headscale to Tailscale - Next Steps:

1. Setup Tailscale:
   - Create a Tailscale account at https://tailscale.com
   - Set up your organization and team

2. Configure Nodes:
   - Install Tailscale on all your nodes
   - Use the generated auth keys or OAuth for authentication

3. Network Configuration:
   - Update your DNS settings if needed
   - Configure subnet routers if you had specific routes
   - Set up ACLs in Tailscale admin console

4. DERP Servers:
   - If you used custom DERP servers, configure them in Tailscale
   - Or use Tailscale's global DERP infrastructure

5. Testing:
   - Test connectivity between nodes
   - Verify DNS resolution
   - Check access controls

Important Notes:
- Node keys cannot be migrated directly between Headscale and Tailscale
- You'll need to re-authenticate all nodes
- Review and update your network policies in Tailscale ACLs
	`

	fmt.Println(instructions)
}
