package transaction

import (
    "fmt"
    "log"
    "mpesa" // Import your M-Pesa API logic
)

func (t *Transaction) ProcessM-PesaWithdrawal() {
    if t.Type == TransactionTypeMobile {
        // Get the recipient's phone number and mobile provider
        recipient, err := GetRecipientByID(t.RecipientID)
        if err != nil {
            log.Fatalf("Recipient not found: %v", err)
        }

        if recipient.MobileProvider != "M-Pesa" {
            log.Fatalf("Invalid mobile provider for transaction: %v", recipient.MobileProvider)
        }

        // Initiate the M-Pesa payment
        mpesaResponse, err := mpesa.InitiatePayment(t.OutputAmountInCents, recipient.PhoneNumber)
        if err != nil {
            log.Fatalf("Failed to process M-Pesa payment: %v", err)
        }

        fmt.Printf("M-Pesa payment initiated successfully. Status: %s\n", mpesaResponse.StatusMessage)
    }
}

