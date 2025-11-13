package mpesa

import (
    "encoding/json"
    "log"
    "net/http"
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

// STKCallbackHandler handles M-Pesa callback and processes transaction updates
func STKCallbackHandler(db *gorm.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var cb STKCallback
        err := json.NewDecoder(r.Body).Decode(&cb)
        if err != nil {
            http.Error(w, "Invalid request body", http.StatusBadRequest)
            log.Printf("Failed to decode callback: %v", err)
            return
        }

        // Extract CheckoutRequestID and ResultCode from the callback
        chkID := cb.Body.StkCallback.CheckoutRequestID
        resultCode := cb.Body.StkCallback.ResultCode

        // Find the corresponding transaction
        var tx models.Transaction
        if err := db.Where("metadata->>'third_party_ref' = ?", chkID).First(&tx).Error; err != nil {
            http.Error(w, "Transaction not found", http.StatusNotFound)
            log.Printf("Transaction not found for CheckoutRequestID: %s", chkID)
            return
        }

        // Process success or failure based on ResultCode
        if resultCode == "0" { // Success
            var acct models.SportsAccount // Replace with the correct account type (e.g., SportsAccount)
            if err := db.First(&acct, tx.SportsAccountID).Error; err != nil {
                http.Error(w, "Sports account not found", http.StatusNotFound)
                log.Printf("Sports account not found for transaction ID: %s", tx.ID)
                return
            }

            // Update the account balance and transaction status
            db.Model(&acct).Update("real_balance_cents", gorm.Expr("real_balance_cents + ?", tx.AmountCents))
            db.Model(&tx).Updates(map[string]any{
                "status": models.StatusSuccess,
                "metadata": models.JSONMap{
                    "mpesa_receipt": extractReceipt(cb),
                    "final_status":  "credited",
                },
            })
            log.Printf("Transaction %s successfully credited.", tx.ID)
        } else { // Failure
            db.Model(&tx).Update("status", models.StatusFailed)
            log.Printf("Transaction %s failed with ResultCode: %s", tx.ID, resultCode)
        }

        // Respond back to M-Pesa
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(`{"status":"received"}`))
    }
}

// extractReceipt looks for the MpesaReceiptNumber in the callback and returns it
func extractReceipt(cb STKCallback) string {
    for _, item := range cb.Body.StkCallback.CallbackMetadata.Item {
        if item.Name == "MpesaReceiptNumber" {
            return item.Value.(string)
        }
    }
    return ""
}
