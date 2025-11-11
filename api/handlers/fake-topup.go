// api/handlers/fake-topup.go
func FakeTopup(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			BookieAccountID uuid.UUID `json:"bookie_account_id"`
			AmountCents     int64     `json:"amount_cents"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		var acct models.BookieAccount
		if err := db.First(&acct, "id = ?", req.BookieAccountID).Error; err != nil {
			http.Error(w, "account not found", http.StatusNotFound)
			return
		}

		db.Model(&acct).Update("fake_balance_cents", gorm.Expr("fake_balance_cents + ?", req.AmountCents))
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"status":        "fake_credited",
			"new_balance":   acct.FakeBalanceCents + req.AmountCents,
		})
	}
}
