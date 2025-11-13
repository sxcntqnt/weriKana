package natsAnish

import (
    "encoding/json"
    "fmt"
    "log"
    "github.com/nats-io/nats.go"
    "weriKana/models"                    // Import models for database interaction
    "gorm.io/gorm"
)

// NATS connection variable
var nc *nats.Conn

// ListenForWithdrawals listens for withdrawal messages on the "withdrawals" subject in NATS
func ListenForWithdrawals(db *gorm.DB, nc *nats.Conn) {
    var err error

    // Connect to NATS
    nc, err = nats.Connect(nats.DefaultURL)
    if err != nil {
        log.Fatal("Failed to connect to NATS: ", err)
    }
    defer nc.Close()

    // Subscribe to the "withdrawals" subject
    _, err = nc.Subscribe("withdrawals", func(msg *nats.Msg) {
        log.Printf("Received withdrawal message: %s\n", string(msg.Data))

        // Parse the withdrawal request from the message
        var request map[string]interface{}
        if err := json.Unmarshal(msg.Data, &request); err != nil {
            log.Printf("Failed to parse withdrawal message: %v", err)
            return
        }

        // Send the message to another program or service for processing the withdrawal
        // For example, we could use an HTTP request, another NATS subject, or a different queue
        err := forwardToProcessingService(request)
        if err != nil {
            log.Printf("Failed to forward message for processing: %v", err)
        }

        // Acknowledge the message after forwarding
        ackMessage(msg)
    })

    if err != nil {
        log.Fatal("Failed to subscribe to 'withdrawals': ", err)
    }

    // Keep listening indefinitely
    select {}
}

// forwardToProcessingService sends the withdrawal data to another service for processing
func forwardToProcessingService(request map[string]interface{}) error {
    // Here we could forward the withdrawal to another NATS topic, HTTP endpoint, or any other service
    // For example, if you're using HTTP to forward:
    //  - You can send a POST request with the withdrawal data to another server.

    // In this example, we'll just log it as a placeholder for actual forwarding logic.
    log.Printf("Forwarding withdrawal request for processing: %v", request)

    // Example HTTP forwarding logic (for another service to process)
    // Replace this with the actual forwarding logic
    // Example (sending to HTTP endpoint)
    // resp, err := http.Post("http://another-service/withdrawals", "application/json", json.NewEncoder(request))
    // if err != nil {
    //     return fmt.Errorf("failed to forward withdrawal request: %w", err)
    // }
    // defer resp.Body.Close()

    return nil
}

// ackMessage acknowledges the NATS message after processing
func ackMessage(msg *nats.Msg) {
    // Acknowledge the message to remove it from the queue
    if err := msg.Ack(); err != nil {
        log.Printf("Failed to acknowledge message: %v", err)
    } else {
        fmt.Println("Message acknowledged.")
    }
}

