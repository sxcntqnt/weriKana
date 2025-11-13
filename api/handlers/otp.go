package handlers

// api/handlers/otp.go
func RequestWithdrawOTP(db *gorm.DB, otpSvc *otp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			CustomerID uuid.UUID `json:"customer_id"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		var customer models.Customer
		if err := db.First(&customer, "id = ?", req.CustomerID).Error; err != nil {
			http.Error(w, "customer not found", http.StatusNotFound)
			return
		}

		code := otpSvc.Send(customer.ID, customer.Phone)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "otp_sent",
			"hint":   "Check SMS",
		})
	}
}
