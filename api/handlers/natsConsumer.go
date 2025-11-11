package consumer

import (
    "fmt"
    "log"
    "github.com/nats-io/nats.go"
    "mpesa" // Assuming this is the M-Pesa API integration package
)

func ListenForWithdrawals() {
    // Connect to NATS
    nc, err := nats.Connect(nats.DefaultURL)
    if err != nil {
        log.Fatal(err)
    }
    defer nc.Close()

    // Subscribe to the "withdrawals" channel
    _, err = nc.Subscribe("withdrawals", func(msg *nats.Msg) {
        fmt.Printf("Received withdrawal message: %s\n", string(msg.Data))
        
        // Process the withdrawal and interact with M-Pesa API
        processWithdrawal(string(msg.Data))
    })
    if err != nil {
        log.Fatal(err)
    }

    // Keep listening
    select {}
}

// Process the withdrawal message and call M-Pesa API to initiate payment
func processWithdrawal(message string) {
    // For the sake of this example, let's assume the message contains the amount and recipient info
    fmt.Println("Processing withdrawal:", message)

    // Call M-Pesa API to send money to the recipient
    // In reality, you'd extract the required details from the message and pass them to the M-Pesa API
    mpesaResponse, err := mpesa.InitiatePayment("5000", "recipientPhoneNumber")
    if err != nil {
        log.Fatalf("Failed to process M-Pesa payment: %v", err)
    }

    fmt.Printf("M-Pesa payment initiated successfully. Status: %s\n", mpesaResponse.StatusMessage)
}

