package withdraw

import (
    "fmt"
    "log"
    "models"
    "natsclient"
)

// WithdrawRequest represents the withdrawal amount and sender info
type WithdrawRequest struct {
    Amount       float64
    SenderID     string
    WithdrawTime time.Time
}

// SmartWithdraw calculates proportional withdrawals and sends to NATS
func SmartWithdraw(request WithdrawRequest) {
    // Get all sportsbook accounts from the pot
    sportsbooks, err := getAllSportsbooks()
    if err != nil {
        log.Fatalf("Failed to fetch sportsbook accounts: %v", err)
    }

    // Calculate the total balance in the pot
    totalPotBalance := 0.0
    for _, sportsbook := range sportsbooks {
        totalPotBalance += sportsbook.Balance
    }

    if totalPotBalance == 0 {
        log.Fatalf("Total pot balance is zero")
        return
    }

    // Calculate proportional withdrawal for each sportsbook in the pot
    for _, sportsbook := range sportsbooks {
        proportion := sportsbook.Balance / totalPotBalance
        amountToWithdraw := request.Amount * proportion

        // Send withdrawal request to NATS
        message := fmt.Sprintf("Withdraw: %f from sportsbook %s in the pot", amountToWithdraw, sportsbook.ID)
        natsclient.SendMessage("withdrawals", []byte(message))
    }

    fmt.Printf("Smart withdrawal completed for amount: %f from the pot\n", request.Amount)
}

// Helper function to fetch all sportsbook accounts (mock)
func getAllSportsbooks() ([]models.Bank, error) {
    // In reality, you would fetch from the database
    sportsbooks := []models.Bank{
        {ID: "sportsbook1", Balance: 1000.0},
        {ID: "sportsbook2", Balance: 1500.0},
    }
    return sportsbooks, nil
}

