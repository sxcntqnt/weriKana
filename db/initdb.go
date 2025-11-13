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

var DB *gorm.DB

// InitDB initializes the database with GORM + all constraints
func InitDB() {
    dsn := os.Getenv("DATABASE_URL")
    if dsn == "" {
        // Fallback: local Postgres
        dsn = "host=localhost user=postgres password=postgres dbname=bankroll port=5432 sslmode=disable"
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

    var err error
    DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
        Logger: newLogger,
    })
    if err != nil {
        log.Fatal("Failed to connect to the database:", err)
    }

    // === 1. AutoMigrate All Models ===
    err = DB.AutoMigrate(
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
    )
    if err != nil {
        log.Fatal("Migration failed:", err)
    }

    // === 2. Create Constraints for Accounts ===
    err = models.CreateSportsAccountConstraints(DB)
    if err != nil {
        log.Fatal("Failed to create sports account constraints:", err)
    }
    err = models.CreateStockAccountConstraints(DB)
    if err != nil {
        log.Fatal("Failed to create stock account constraints:", err)
    }
    err = models.CreateForexAccountConstraints(DB)
    if err != nil {
        log.Fatal("Failed to create forex account constraints:", err)
    }
    err = models.CreateCryptoAccountConstraints(DB)
    if err != nil {
        log.Fatal("Failed to create crypto account constraints:", err)
    }

    // === 3. Create Additional Constraints & Indexes ===
    DB.Exec(`
        CREATE UNIQUE INDEX IF NOT EXISTS idx_bookie_customer
        ON sports_accounts (bookie_id, customer_id)
        WHERE deleted_at IS NULL;
    `)

    // Unique: Customer phone & email
    DB.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_customers_phone ON customers (phone) WHERE deleted_at IS NULL;`)
    DB.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_customers_email ON customers (email) WHERE deleted_at IS NULL;`)

    // Unique: Sender per Customer
    DB.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_senders_customer ON senders (customer_id) WHERE deleted_at IS NULL;`)

    // Unique: Transaction reference (MPESA receipt, OTP ref)
    DB.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_txn_ref ON transactions (reference) WHERE deleted_at IS NULL;`)

    // Performance: Status + BookieAccount
    DB.Exec(`CREATE INDEX IF NOT EXISTS idx_txn_status ON transactions (status);`)
    DB.Exec(`CREATE INDEX IF NOT EXISTS idx_txn_bookie ON transactions (bookie_account_id);`)

    // After AutoMigrate
    DB.AutoMigrate(&models.Transaction{})

    // === 4. Add Dual Balance Columns ===
    DB.Exec("ALTER TABLE sports_accounts ADD COLUMN IF NOT EXISTS real_balance_cents BIGINT DEFAULT 0")
    DB.Exec("ALTER TABLE sports_accounts ADD COLUMN IF NOT EXISTS fake_balance_cents BIGINT DEFAULT 0")

    // Add IsReal to transactions
    DB.Exec("ALTER TABLE transactions ADD COLUMN IF NOT EXISTS is_real BOOLEAN DEFAULT FALSE")

    // Index for fast balance queries
    DB.Exec("CREATE INDEX IF NOT EXISTS idx_bookie_real_balance ON sports_accounts (customer_id, bookie_id) WHERE real_balance_cents > 0")
    DB.Exec("CREATE INDEX IF NOT EXISTS idx_bookie_fake_balance ON sports_accounts (customer_id, bookie_id) WHERE fake_balance_cents > 0")
    DB.Exec("ALTER TABLE bookies ADD COLUMN IF NOT EXISTS mpesa_number TEXT")
    DB.Exec("ALTER TABLE sports_accounts ADD COLUMN IF NOT EXISTS encrypted_key TEXT")

    // === 5. Seed Master Encryption Key (if not exists) ===
    seedMasterKey()

    log.Println("Database initialized successfully with all constraints")
}

// seedMasterKey generates a 32-byte AES key and stores it in .env or KMS
func seedMasterKey() {
    keyPath := ".master_key"
    if _, err := os.Stat(keyPath); os.IsNotExist(err) {
        key := make([]byte, 32)
        if _, err := rand.Read(key); err != nil {
            log.Fatal("Failed to generate master key:", err)
        }
        if err := os.WriteFile(keyPath, key, 0600); err != nil {
            log.Fatal("Failed to save master key:", err)
        }
        log.Println("Generated new master encryption key")
    } else {
        log.Println("Master key already exists")
    }
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

