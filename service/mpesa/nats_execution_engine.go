// services/nats_execution_consumer.go
func StartExecutionConsumer(db *gorm.DB, nc *nats.Conn) {
	nc.Subscribe("bets.cashout.withdraw", func(m *nats.Msg) {
		var payload map[string]any
		json.Unmarshal(m.Data, &payload)

		withdrawals := payload["withdrawals"].([]any)
		for _, w := range withdrawals {
			wd := w.(map[string]any)
			bookieAcctID := wd["bookie_account_id"].(string)
			amount := wd["amount_cents"].(float64)
			encKey := wd["encrypted_key"].(string)
			otp := wd["otp"].(string)
			txID := wd["transaction_id"].(string)

			// Decrypt key
			key, _ := decrypt(encKey)

			// Call Execution Engine (web scraping)
			go func() {
				result := scrapeAndWithdraw(bookieAcctID, key, otp, int64(amount))
				if result.Success {
					updateTx(db, txID, models.StatusSuccess, result.Receipt)
				} else {
					updateTx(db, txID, models.StatusFailed, result.Error)
				}
			}()
		}

		m.Ack()
	})
}

type ScrapingResult struct {
	Success bool
	Receipt string
	Error   string
}

// services/nats_execution.go
func StartExecutionEngine(db *gorm.DB, nc *nats.Conn) {
	nc.Subscribe("bets.cashout.withdraw", func(m *nats.Msg) {
		var payload map[string]any
		json.Unmarshal(m.Data, &payload)

		withdrawals := payload["withdrawals"].([]any)
		for _, w := range withdrawals {
			wd := w.(map[string]any)
			encKey := wd["encrypted_key"].(string)
			otp := wd["otp"].(string)
			amount := int64(wd["amount_cents"].(float64))
			txID := wd["transaction_id"].(string)

			key, _ := decrypt(encKey)

			go func() {
				result := executeScraping(key, otp, amount)
				if result.Success {
					updateTx(db, txID, models.StatusSuccess, result.Receipt)
				} else {
					updateTx(db, txID, models.StatusFailed, result.Error)
				}
			}()
		}
		m.Ack()
	})
}
