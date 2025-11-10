var db *sql.DB

func initDB() {
	var err error
	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal(err)
	}

	// Create tables
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL
		);
		CREATE TABLE IF NOT EXISTS accounts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			bookie_name TEXT NOT NULL,
			encrypted_real_bal BLOB NOT NULL,
			fake_bal REAL NOT NULL DEFAULT 0,
			FOREIGN KEY (user_id) REFERENCES users (id)
		);
		CREATE TABLE IF NOT EXISTS ledger (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			bookie_name TEXT NOT NULL,
			tx_type TEXT NOT NULL,
			amount REAL NOT NULL,
			is_real BOOLEAN NOT NULL,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
			prev_hash TEXT NOT NULL,
			signature TEXT NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users (id)
		);
	`)
	if err != nil {
		log.Fatal(err)
	}
}
