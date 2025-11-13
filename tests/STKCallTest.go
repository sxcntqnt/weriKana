package tests

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    "weriKana/models"
    "gorm.io/gorm"
    "github.com/stretchr/testify/assert"
)

func TestSTKCallbackHandler_Success(t *testing.T) {
    db := setupTestDB()  // You should set up a test DB connection

    // Create a mock transaction
    txn := &models.Transaction{
        BookieAccountID: uuid.New(),
        CustomerID:      uuid.New(),
        AmountCents:     10000,
        Status:          models.StatusPending,
        Metadata: models.JSONMap{
            "third_party_ref": "test-checkout-id",
        },
    }
    db.Create(txn)

    // Mock the M-Pesa callback
    callback := STKCallback{
        Body: struct {
            StkCallback struct {
                CheckoutRequestID string `json:"CheckoutRequestID"`
                ResultCode        string `json:"ResultCode"`
                ResultDesc        string `json:"ResultDesc"`
                CallbackMetadata  struct {
                    Item []struct {
                        Name  string `json:"Name"`
                        Value any    `json:"Value"`
                    } `json:"Item"`
                } `json:"CallbackMetadata"`
            } `json:"Body"`
        }{
            StkCallback: struct {
                CheckoutRequestID string `json:"CheckoutRequestID"`
                ResultCode        string `json:"ResultCode"`
                ResultDesc        string `json:"ResultDesc"`
                CallbackMetadata  struct {
                    Item []struct {
                        Name  string `json:"Name"`
                        Value any    `json:"Value"`
                    } `json:"Item"`
                } `json:"CallbackMetadata"`
            }{
                CheckoutRequestID: "test-checkout-id",
                ResultCode:        "0",
                ResultDesc:        "Payment successful",
                CallbackMetadata: struct {
                    Item []struct {
                        Name  string `json:"Name"`
                        Value any    `json:"Value"`
                    } `json:"Item"`
                }{
                    Item: []struct {
                        Name  string `json:"Name"`
                        Value any    `json:"Value"`
                    }{
                        {Name: "MpesaReceiptNumber", Value: "1234567890"},
                    },
                },
            },
        },
    }

    // Serialize callback to JSON
    callbackJSON, _ := json.Marshal(callback)

    // Make a mock HTTP request to simulate the callback
    req := httptest.NewRequest("POST", "/stk/callback", bytes.NewBuffer(callbackJSON))
    resp := httptest.NewRecorder()

    // Call the callback handler
    STKCallbackHandler(db)(resp, req)

    // Assert the status is success
    assert.Equal(t, http.StatusOK, resp.Code)

    // Verify that the transaction's status was updated
    var txnUpdated models.Transaction
    db.First(&txnUpdated, txn.ID)
    assert.Equal(t, models.StatusSuccess, txnUpdated.Status)
    assert.Contains(t, txnUpdated.Metadata, "mpesa_receipt")
}

