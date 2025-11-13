package mpesa

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "os"
    "time"
)

var (
    httpClient = &http.Client{Timeout: 30 * time.Second}
    baseURL    = os.Getenv("MPESA_DJANGO_API_URL") // Example: https://mpesa-api.yourapp.com
    apiToken   = os.Getenv("MPESA_API_TOKEN")      // Bearer token
)

type STKPushRequest struct {
    Phone         string `json:"phone_number"`
    Amount        int64  `json:"amount"`
    AccountRef    string `json:"account_reference"`
    TransactionID string `json:"transaction_id"` // idempotency
    CallbackURL   string `json:"callback_url,omitempty"`
}

type STKPushResponse struct {
    CheckoutRequestID string `json:"checkout_request_id"`
    ResponseCode      string `json:"response_code"`
    CustomerMessage   string `json:"customer_message"`
}

type B2CRequest struct {
    Phone         string `json:"phone_number"`
    Amount        int64  `json:"amount"`
    CommandID     string `json:"command_id"` // "BusinessPayment"
    Occasion      string `json:"occasion,omitempty"`
    Remarks       string `json:"remarks,omitempty"`
    TransactionID string `json:"transaction_id"`
}

type B2CResponse struct {
    ConversationID    string `json:"conversation_id"`
    OriginatorConvID  string `json:"originator_conversation_id"`
    ResponseCode      string `json:"response_code"`
}

// SendSTKPush - for Smart Deposit
func SendSTKPush(phone string, amountCents int64, idempotencyKey string) (*STKPushResponse, error) {
    if baseURL == "" {
        return nil, fmt.Errorf("MPESA_DJANGO_API_URL not set")
    }

    reqBody := STKPushRequest{
        Phone:         phone,
        Amount:        amountCents / 100, // API expects KES (shillings)
        AccountRef:    "BANKROLL_SMART_DEPOSIT",
        TransactionID: idempotencyKey,
        CallbackURL:   os.Getenv("MPESA_STK_CALLBACK_URL"),
    }

    data, _ := json.Marshal(reqBody)
    url := fmt.Sprintf("%s/lipanampesa/online/", baseURL)

    resp, err := doRequest("POST", url, data)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var result STKPushResponse
    json.NewDecoder(resp.Body).Decode(&result)
    return &result, nil
}

// SendB2C - for Smart Withdraw
func SendB2C(phone string, amountCents int64, idempotencyKey string) (*B2CResponse, error) {
    if baseURL == "" {
        return nil, fmt.Errorf("MPESA_DJANGO_API_URL not set")
    }

    reqBody := B2CRequest{
        Phone:         phone,
        Amount:        amountCents / 100, // API expects KES (shillings)
        CommandID:     "BusinessPayment",
        Remarks:       "BankRoll Smart Withdraw",
        TransactionID: idempotencyKey,
    }

    data, _ := json.Marshal(reqBody)
    url := fmt.Sprintf("%s/b2c/transaction/", baseURL)

    resp, err := doRequest("POST", url, data)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var result B2CResponse
    json.NewDecoder(resp.Body).Decode(&result)
    return &result, nil
}

// doRequest makes HTTP requests to the M-Pesa API with error handling
func doRequest(method, url string, body []byte) (*http.Response, error) {
    req, _ := http.NewRequest(method, url, bytes.NewBuffer(body))
    req.Header.Set("Content-Type", "application/json")
    if apiToken != "" {
        req.Header.Set("Authorization", "Bearer "+apiToken)
    }

    resp, err := httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    if resp.StatusCode >= 400 {
        b, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("MPesa API error %d: %s", resp.StatusCode, string(b))
    }
    return resp, nil
}

