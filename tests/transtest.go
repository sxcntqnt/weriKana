package tests

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "weriKana/services/mpesa"
    "github.com/google/uuid"
    "gorm.io/gorm"
)

func TestProcessMPesaWithdrawal(t *testing.T) {
    db := setupTestDB()  // Set up a test DB connection

    // Create a mock transaction with type "mobile"
    transaction := &Transaction{
        Type:               TransactionTypeWithdraw,
        AmountCents:        5000, // Amount in cents
        BookieAccountID:    uuid.New(),
        CustomerID:         uuid.New(),
        ExternalID:         "external-id",
        Metadata: JSONMap{
            "third_party_ref": "test-checkout-id", // mock third-party reference
        },
    }

    // Insert the transaction into the DB
    err := db.Create(transaction).Error
    if err != nil {
        t.Fatalf("Error creating mock transaction: %v", err)
    }

    // Mock M-Pesa API response
    mpesaResponse := &mpesa.B2CResponse{
        ResponseCode:      "0",
        OriginatorConvID:  "originator-conv-id",
        ConversationID:    "conversation-id",
    }

    // Mock M-Pesa SendB2C function to simulate withdrawal
    mpesa.SendB2C = func(phone string, amountCents int64, idempotencyKey string) (*mpesa.B2CResponse, error) {
        return mpesaResponse, nil
    }

    // Call the method to process M-Pesa withdrawal
    err = transaction.ProcessMPesaWithdrawal(db)

    assert.Nil(t, err)
    assert.Equal(t, "0", mpesaResponse.ResponseCode)

    // Verify transaction status is updated to 'success'
    var updatedTransaction Transaction
    err = db.First(&updatedTransaction, transaction.ID).Error
    assert.Nil(t, err)
    assert.Equal(t, StatusSuccess, updatedTransaction.Status)
}


