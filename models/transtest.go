package transaction

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "mpesa"
)

func TestProcessM-PesaWithdrawal(t *testing.T) {
    // Create a mock transaction with type "mobile"
    transaction := &Transaction{
        Type:               TransactionTypeMobile,
        OutputAmountInCents: "5000",  // Assuming the amount is in cents
        RecipientID:        "recipient-id",  // Mock recipient ID
    }

    // Mock a successful M-Pesa response
    mpesaResponse := &mpesa.PaymentResponse{
        StatusCode:    "200",
        StatusMessage: "Payment successful",
    }

    // Mock the M-Pesa API call
    mpesa.InitiatePayment = func(amount string, phoneNumber string) (*mpesa.PaymentResponse, error) {
        return mpesaResponse, nil
    }

    // Call the method to process M-Pesa withdrawal
    err := transaction.ProcessM-PesaWithdrawal()

    assert.Nil(t, err)
    assert.Equal(t, "200", mpesaResponse.StatusCode)
    assert.Equal(t, "Payment successful", mpesaResponse.StatusMessage)
}

