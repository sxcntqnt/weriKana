package mpesa

import (
    "log"
    "github.com/gofiber/fiber/v2" // Import the Fiber package
    "weriKana/models"
    "gorm.io/gorm"
)

// STKCallback structure to match the expected response from M-Pesa
type STKCallback struct {
    Body struct {
        StkCallback struct {
            CheckoutRequestID string `json:"CheckoutRequestID"`
            ResultCode        string `json:"ResultCode"`
            ResultDesc        string `json:"ResultDesc"`
            CallbackMetadata  struct {
                Item []struct {
                    Name  string `json:"Name"`
                    Value any    `json:"Value"`
                } `json:"Item"`
            } `json:"CallbackMetadata"`
        } `json:"Body"`
    } `json:"Body"`
}

// STKCallbackHandler — internal handler (adapted for GoFiber)
func STKCallbackHandler(db *gorm.DB) fiber.Handler {
    return func(c *fiber.Ctx) error {
        var cb STKCallback
        if err := c.BodyParser(&cb); err != nil {
            log.Printf("M-Pesa callback decode error: %v", err)
            return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
                "error": "Invalid JSON",
            })
        }

        chkID := cb.Body.StkCallback.CheckoutRequestID
        resultCode := cb.Body.StkCallback.ResultCode

        var tx models.Transaction
        if err := db.Where("metadata->>'third_party_ref' = ?", chkID).First(&tx).Error; err != nil {
            log.Printf("No tx for CheckoutRequestID: %s", chkID)
            return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
                "error": "Tx not found",
            })
        }

        if resultCode == "0" {
            var acct models.SportsAccount
            if err := db.First(&acct, tx.SportsAccountID).Error; err != nil {
                log.Printf("Account not found: %v", err)
                return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
                    "error": "Account error",
                })
            }

            db.Model(&acct).Update("real_balance_cents", gorm.Expr("real_balance_cents + ?", tx.AmountCents))
            db.Model(&tx).Updates(map[string]any{
                "status": models.StatusSuccess,
                "metadata": models.JSONMap{
                    "mpesa_receipt": extractReceipt(cb),
                    "final_status":  "credited",
                },
            })
            log.Printf("Deposit successful → TxID: %s | Amount: %d", tx.ID, tx.AmountCents)
        } else {
            db.Model(&tx).Update("status", models.StatusFailed)
            log.Printf("Deposit failed → TxID: %s | Code: %s", tx.ID, resultCode)
        }

        return c.Status(fiber.StatusOK).JSON(fiber.Map{
            "status": "received",
        })
    }
}

// Helper function to extract the MpesaReceiptNumber from the callback metadata
func extractReceipt(cb STKCallback) string {
    for _, item := range cb.Body.StkCallback.CallbackMetadata.Item {
        if item.Name == "MpesaReceiptNumber" {
            if s, ok := item.Value.(string); ok {
                return s
            }
        }
    }
    return ""
}

