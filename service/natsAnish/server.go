package natsAnish

import (
	"fmt"
	"log"

	"github.com/nats-io/nats.go"
	"gorm.io/gorm"
	"weriKana/models"
)

// ScrapingResult represents the outcome of a scraping operation
type ScrapingResult struct {
	Success bool
	Receipt string
	Error   string
}

// StartExecutionConsumer starts the NATS consumer for transaction processing
func StartExecutionConsumer(db *gorm.DB, nc *nats.Conn) {
	_, err := nc.Subscribe("bets.cashout.withdraw", func(m *nats.Msg) {
		msg, err := DecodeMsg(m.Data)
		if err != nil {
			log.Printf("Failed to decode message: %v", err)
			return
		}
		for _, leg := range msg.Transactions {
			go func(leg TransactionLeg) {
				key, err := decrypt(leg.EncryptedKey)
				if err != nil {
					log.Printf("Failed to decrypt key for tx %s: %v", leg.TransactionID, err)
					if err := updateTx(db, leg.TransactionID.String(), string(models.StatusFailed), err.Error()); err != nil {
						log.Printf("Failed to update transaction: %v", err)
					}
					return
				}
				result := executeScraping(key, leg.OTP, leg.AmountCents)
				if result.Success {
					if err := updateTx(db, leg.TransactionID.String(), string(models.StatusSuccess), result.Receipt); err != nil {
						log.Printf("Failed to update transaction: %v", err)
					}
				} else {
					if err := updateTx(db, leg.TransactionID.String(), string(models.StatusFailed), result.Error); err != nil {
						log.Printf("Failed to update transaction: %v", err)
					}
				}
			}(leg)
		}
		if err := m.Ack(); err != nil {
			log.Printf("Failed to acknowledge message: %v", err)
		}
	})
	if err != nil {
		log.Fatalf("Failed to subscribe to bets.cashout.withdraw: %v", err)
	}
}

// decrypt is a placeholder for the decryption logic
func decrypt(encryptedKey string) (string, error) {
	// Implement actual decryption logic here
	return encryptedKey, nil
}

// executeScraping is a placeholder for the web scraping logic
func executeScraping(key, otp string, amount int64) ScrapingResult {
	// Implement actual scraping logic here
	return ScrapingResult{
		Success: true,
		Receipt: "placeholder_receipt",
	}
}

// updateTx updates the transaction status and metadata in the database
func updateTx(db *gorm.DB, txID, status, details string) error {
	var tx models.Transaction
	if err := db.Where("id = ?", txID).First(&tx).Error; err != nil {
		return fmt.Errorf("failed to find transaction %s: %w", txID, err)
	}

	updates := map[string]interface{}{
		"status": status,
		"metadata": models.JSONMap{
			"details": details,
		},
	}

	if err := db.Model(&tx).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update transaction %s: %w", txID, err)
	}

	log.Printf("Updated transaction %s to status %s with details: %s", txID, status, details)
	return nil
}
