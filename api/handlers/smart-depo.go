package deposit

import (
    "fmt"
    "log"
    "time"
    "models"
    "natsclient"
)

// DepositRequest represents the deposit amount and recipient info
type DepositRequest struct {
    Amount       float64
    RecipientID  string
    DepositTime  time.Time
}

// SmartDeposit calculates the proportional deposits and sends to NATS
func SmartDeposit(request DepositRequest) {
    // Get the total balance in the pot
    totalPotBalance := 0.0
    sportsbooks, err := getAllSportsbooks()
    if err != nil {
        log.Fatalf("Failed to fetch sportsbook accounts: %v", err)
    }

    // Get total balance of the pot (sum of all sportsbook balances)
    for _, sportsbook := range sportsbooks {
        totalPotBalance += sportsbook.Balance
    }

    if totalPotBalance == 0 {
        log.Fatalf("Total pot balance is zero")
        return
    }

    // Calculate proportional deposit for each sportsbook in the pot
    for _, sportsbook := range sportsbooks {
        proportion := sportsbook.Balance / totalPotBalance
        amountToDeposit := request.Amount * proportion

        // Send the deposit request to NATS
        message := fmt.Sprintf("Deposit: %f to sportsbook %s from the pot", amountToDeposit, sportsbook.ID)
        natsclient.SendMessage("deposits", []byte(message))
    }

    fmt.Printf("Smart deposit completed for amount: %f from the pot\n", request.Amount)
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

