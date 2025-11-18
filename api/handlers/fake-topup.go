package handlers

import (
    "encoding/json"          // Add this import for JSON handling
    "net/http"
    "github.com/google/uuid"
    "gorm.io/gorm"
    "weriKana/models"         // Import the models package to use models.BookieAccount
)

// FakeTopup handles the fake top-up of a bookie account's balance
func FakeTopup(db *gorm.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var req struct {
            BookieAccountID uuid.UUID `json:"bookie_account_id"`
            AmountCents     int64     `json:"amount_cents"`
        }

        // Decode the incoming JSON request body
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            http.Error(w, "Invalid request body", http.StatusBadRequest)
            return
        }

        // Find the bookie account in the database
        var acct models.SportsAccount
        if err := db.First(&acct, "id = ?", req.BookieAccountID).Error; err != nil {
            http.Error(w, "Account not found", http.StatusNotFound)
            return
        }

        // Update the fake balance for the account
        db.Model(&acct).Update("fake_balance_cents", gorm.Expr("fake_balance_cents + ?", req.AmountCents))

        // Respond with the updated balance
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(map[string]any{
            "status":      "fake_credited",
            "new_balance": acct.FakeBalanceCents + req.AmountCents,
        })
    }
}

