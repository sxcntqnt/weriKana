package db

import (
	"crypto/rand"
	"log"
	"os"
	"time"

	"weriKana/models"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite" // Import SQLite driver
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const MasterKeyPath = ".master_key"

var DB *gorm.DB

// Init initializes the database connection and runs all migrations/constraints
// Pass your DSN from main.go (e.g. from env var), or empty string for default
func Init(dsn string) (*gorm.DB, error) {
	// Use provided DSN, fallback to env, then to local default
	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
	}
	if dsn == "" {
		log.Println("DATABASE_URL not set, using local default")
		dsn = "host=localhost user=postgres password=postgres dbname=werikana port=5432 sslmode=disable TimeZone=Africa/Nairobi"
	}

	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Info,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: newLogger,
		// Optional: disable foreign key constraints when migrating (GORM default is off anyway)
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		return nil, err
	}

	// === 1. AutoMigrate all models ===
	if err := db.AutoMigrate(
		&models.SportsAccount{},
		&models.StockAccount{},
		&models.ForexAccount{},
		&models.CryptoAccount{},
		&models.Sharp{},
		&models.SharpAccount{},
		&models.Customer{},
		&models.Sender{},
		&models.AssetNexus{},
		&models.SportsManager{},
		&models.StockManager{},
		&models.ForexManager{},
		&models.CryptoManager{},
		&models.SharpProfile{},
		&models.Transaction{},
		&models.TransactionLog{},
	); err != nil {
		return nil, err
	}

	// === 2. Create account constraints ===
	if err := models.CreateSportsAccountConstraints(db); err != nil {
		return nil, err
	}
	if err := models.CreateStockAccountConstraints(db); err != nil {
		return nil, err
	}
	if err := models.CreateForexAccountConstraints(db); err != nil {
		return nil, err
	}
	if err := models.CreateCryptoAccountConstraints(db); err != nil {
		return nil, err
	}

	// === 3. Additional indexes & constraints ===
	rawSQL := []string{
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_bookie_customer ON sports_accounts (bookie_id, customer_id) WHERE deleted_at IS NULL;`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_customers_phone ON customers (phone) WHERE deleted_at IS NULL;`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_customers_email ON customers (email) WHERE deleted_at IS NULL;`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_senders_customer ON senders (customer_id) WHERE deleted_at IS NULL;`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_txn_ref ON transactions (reference) WHERE deleted_at IS NULL;`,
		`CREATE INDEX IF NOT EXISTS idx_txn_status ON transactions (status);`,
		`CREATE INDEX IF NOT EXISTS idx_txn_bookie ON transactions (bookie_account_id);`,
		`CREATE INDEX IF NOT EXISTS idx_bookie_real_balance ON sports_accounts (customer_id, bookie_id) WHERE real_balance_cents > 0;`,
		`CREATE INDEX IF NOT EXISTS idx_bookie_fake_balance ON sports_accounts (customer_id, bookie_id) WHERE fake_balance_cents > 0;`,
	}
	for _, sql := range rawSQL {
		if err := db.Exec(sql).Error; err != nil {
			log.Printf("Warning: index creation failed (might already exist): %v", err)
		}
	}

	// === 4. Add missing columns safely ===
	columns := map[string]string{
		"sports_accounts":      "real_balance_cents BIGINT DEFAULT 0",
		"fake_sports_accounts": "fake_balance_cents BIGINT DEFAULT 0",
		"master_key":           "encrypted_key TEXT",
		"transactions":         "is_real BOOLEAN DEFAULT FALSE",
		"bookies":              "mpesa_number TEXT",
	}
	for table, def := range columns {
		db.Exec(`ALTER TABLE ` + table + ` ADD COLUMN IF NOT EXISTS ` + def)
	}

	// === 5. Seed master encryption key if not exists ===
	LoadOrGenerateMasterKey()

	log.Println("Database initialized & migrated successfully")
	return db, nil
}

// LoadOrGenerateMasterKey loads from disk or generates + saves a new 32-byte AES key
// Returns the key bytes — never fails (panics on critical errors)
func LoadOrGenerateMasterKey() []byte {
	key, err := os.ReadFile(MasterKeyPath)
	if err != nil || len(key) != 32 {
		log.Println("No valid master key found — generating new one...")

		key = make([]byte, 32) // 256-bit AES key
		if _, err := rand.Read(key); err != nil {
			log.Fatalf("FATAL: Cannot generate master key: %v", err)
		}

		if err := os.WriteFile(MasterKeyPath, key, 0600); err != nil {
			log.Fatalf("FATAL: Cannot save master key to %s: %v", MasterKeyPath, err)
		}

		log.Println("New 32-byte master encryption key generated and saved to", MasterKeyPath)
		log.Println("BACK IT UP SECURELY NOW — losing this file = permanent data loss!")
	} else {
		log.Println("Master encryption key loaded from", MasterKeyPath)
	}

	return key
}

// SetupDatabase connects to SQLite DB (outside of InitDB function)
func SetupDatabase(dbPath string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return nil, err // Return error instead of log.Fatal
	}

	// Run migrations and create constraints for SQLite if necessary
	err = db.AutoMigrate(&models.SportsBank{}, &models.StockBank{})
	if err != nil {
		return nil, err
	}

	return db, nil
}
