package db

import (
	"log"
	"os"
	"strings"
	"time"

	"weriKana/models"

	"github.com/spf13/viper"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const MasterKeyPath = ".master_key"

var DB *gorm.DB

// Init initializes the database connection and runs all migrations/constraints.
func Init() (*gorm.DB, error) {
	// Determine environment (Viper first, fallback OS)
	env := viper.GetString("ENVIRONMENT")
	if env == "" {
		env = os.Getenv("ENVIRONMENT")
	}
	if env == "" {
		env = "dev"
		log.Println("ENVIRONMENT not set, defaulting to 'dev'")
	}
	var dbURL string
	var db *gorm.DB
	var err error
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Info,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)
	// ===========================
	// PRODUCTION — POSTGRES
	// ===========================
	if env == "production" {
		dbURL = viper.GetString("DATABASE_URL")
		if dbURL == "" {
			dbURL = os.Getenv("DATABASE_URL")
		}
		if dbURL == "" {
			dbURL = "host=localhost user=postgres password=postgres dbname=werikana port=5432 sslmode=disable TimeZone=Africa/Nairobi"
			log.Println("No database URL provided, using default PostgreSQL connection")
		}
		db, err = gorm.Open(postgres.Open(dbURL), &gorm.Config{
			Logger: newLogger,
			DisableForeignKeyConstraintWhenMigrating: true,
		})
		if err != nil {
			return nil, err
		}
		log.Println("Connected to PostgreSQL database (production)")
	} else {
		// ===========================
		// DEVELOPMENT — SQLITE
		// ===========================
		dbURL = viper.GetString("SQLITE_DB_PATH")
		if dbURL == "" {
			dbURL = os.Getenv("SQLITE_DB_PATH")
		}
		if dbURL == "" {
			dbURL = "dev.db"
			log.Println("No database URL provided, using default SQLite database: dev.db")
		}
		db, err = gorm.Open(sqlite.Open(dbURL), &gorm.Config{
			Logger: newLogger,
		})
		if err != nil {
			return nil, err
		}
		log.Printf("Connected to SQLite database: %s (development)", dbURL)
	}
	// === 1. AutoMigrate all models ===
	modelsList := []interface{}{
		&models.SportsAccount{},
		&models.StockAccount{},
		&models.ForexAccount{},
		&models.CryptoAccount{},
		&models.Sharp{},
		&models.SharpAccount{},
		&models.Sender{},
		&models.AssetNexus{},
		&models.SportsWallet{},
		&models.StockWallet{},
		&models.ForexWallet{},
		&models.CryptoWallet{},
		&models.SportsManager{},
		&models.StockManager{},
		&models.ForexManager{},
		&models.CryptoManager{},
		&models.SharpProfile{},
		&models.Transaction{},
		&models.TransactionLog{},
		&models.Bookie{},
		&models.RealLedgerEntry{},
		&models.FakeLedgerEntry{},
	}
	if err := db.AutoMigrate(modelsList...); err != nil {
		log.Printf("Warning: AutoMigrate partial failure (fix models): %v", err)
	}
	// === 2. Account constraints ===
	constraintFuncs := []func(*gorm.DB) error{
		models.CreateSportsAccountConstraints,
		models.CreateStockAccountConstraints,
		models.CreateForexAccountConstraints,
		models.CreateCryptoAccountConstraints,
	}
	for _, fn := range constraintFuncs {
		if err := fn(db); err != nil {
			log.Printf("Warning: Constraint creation failed: %v", err)
		}
	}
	// === 3. Additional indexes & constraints ===
	if env == "production" {
		rawSQL := []string{
			`CREATE UNIQUE INDEX IF NOT EXISTS idx_bookie_sharp ON sports_accounts (bookie_id, sharp_id) WHERE deleted_at IS NULL;`, // Updated: sharp_id replaces customer_id
			`CREATE UNIQUE INDEX IF NOT EXISTS idx_sharps_email ON sharps (email) WHERE deleted_at IS NULL;`, // Updated: sharps replaces customers; skip phone (JSON)
			`CREATE UNIQUE INDEX IF NOT EXISTS idx_senders_sharp ON senders (sharp_id) WHERE deleted_at IS NULL;`, // Updated: sharp_id replaces customer_id
			`CREATE UNIQUE INDEX IF NOT EXISTS idx_txn_ref ON transactions (reference) WHERE deleted_at IS NULL;`,
			`CREATE INDEX IF NOT EXISTS idx_txn_status ON transactions (status);`,
			`CREATE INDEX IF NOT EXISTS idx_txn_bookie ON transactions (bookie_account_id);`,
			`CREATE INDEX IF NOT EXISTS idx_bookie_real_balance ON sports_accounts (sharp_id, bookie_id) WHERE real_balance_cents > 0;`, // Updated: sharp_id
			`CREATE INDEX IF NOT EXISTS idx_bookie_fake_balance ON sports_accounts (sharp_id, bookie_id) WHERE fake_balance_cents > 0;`, // Updated: sharp_id
		}
		for _, sql := range rawSQL {
			if err := db.Exec(sql).Error; err != nil {
				log.Printf("Warning: PostgreSQL index creation failed: %v", err)
			}
		}
		columns := map[string]string{
			"sports_accounts": "real_balance_cents BIGINT DEFAULT 0",
			"fake_sports_accounts": "fake_balance_cents BIGINT DEFAULT 0",
			"master_key": "encrypted_key TEXT",
			"transactions": "is_real BOOLEAN DEFAULT FALSE",
			"bookies": "mpesa_number TEXT",
		}
		for table, def := range columns {
			parts := strings.SplitN(def, " ", 3)
			if len(parts) < 2 {
				continue
			}
			col := parts[0]
			rest := strings.Join(parts[1:], " ")
			if err := db.Exec("ALTER TABLE " + table + " ADD COLUMN IF NOT EXISTS " + col + " " + rest).Error; err != nil {
				log.Printf("Warning: Failed to add column to %s: %v", table, err)
			}
		}
	} else {
		rawSQL := []string{
			`CREATE UNIQUE INDEX IF NOT EXISTS idx_bookie_sharp ON sports_accounts (bookie_id, sharp_id);`, // Updated: sharp_id replaces customer_id
			`CREATE UNIQUE INDEX IF NOT EXISTS idx_sharps_email ON sharps (email);`, // Updated: sharps replaces customers; skip phone (JSON)
			`CREATE UNIQUE INDEX IF NOT EXISTS idx_senders_sharp ON senders (sharp_id);`, // Updated: sharp_id replaces customer_id
			`CREATE UNIQUE INDEX IF NOT EXISTS idx_txn_ref ON transactions (reference);`,
			`CREATE INDEX IF NOT EXISTS idx_txn_status ON transactions (status);`,
			`CREATE INDEX IF NOT EXISTS idx_txn_bookie ON transactions (bookie_account_id);`,
			`CREATE INDEX IF NOT EXISTS idx_bookie_real_balance ON sports_accounts (sharp_id, bookie_id);`, // Updated: sharp_id
			`CREATE INDEX IF NOT EXISTS idx_bookie_fake_balance ON sports_accounts (sharp_id, bookie_id);`, // Updated: sharp_id
		}
		for _, sql := range rawSQL {
			if err := db.Exec(sql).Error; err != nil {
				log.Printf("Warning: SQLite index creation failed: %v", err)
			}
		}
		log.Println("SQLite database indexes created")
	}
	// === 4. Create master_key table ===
	if err := db.Exec(`
		CREATE TABLE IF NOT EXISTS master_key (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			encrypted_key TEXT
		);
	`).Error; err != nil {
		log.Printf("Warning: master_key table creation failed: %v", err)
	}
	// === 5. Seed master encryption key ===
	log.Println("Database initialized & migrated successfully")
	DB = db
	return db, nil
}
