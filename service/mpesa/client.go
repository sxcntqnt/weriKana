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


// SendSTKPush — uses the config set by Init()
func SendSTKPush(phone string, amountCents int64, idempotencyKey string) (*STKPushResponse, error) {
    reqBody := STKPushRequest{
        Phone:         phone,
        Amount:        amountCents / 100,
        AccountRef:    "BANKROLL_SMART_DEPOSIT",
        TransactionID: idempotencyKey,
        CallbackURL:   config.STKCallbackURL, // ← uses internal config
    }

    data, _ := json.Marshal(reqBody)
    url := fmt.Sprintf("%s/lipanampesa/online/", config.BaseURL)

    resp, err := doRequest("POST", url, data)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var result STKPushResponse
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    return &result, nil
}

// SendB2C
func SendB2C(phone string, amountCents int64, idempotencyKey string) (*B2CResponse, error) {
    reqBody := B2CRequest{
        Phone:         phone,
        Amount:        amountCents / 100,
        CommandID:     "BusinessPayment",
        Remarks:       "BankRoll Smart Withdraw",
        TransactionID: idempotencyKey,
    }

    data, _ := json.Marshal(reqBody)
    url := fmt.Sprintf("%s/b2c/transaction/", config.BaseURL)

    resp, err := doRequest("POST", url, data)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var result B2CResponse
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    return &result, nil
}

// doRequest — uses config.HTTPClient and config.APIToken
func doRequest(method, url string, body []byte) (*http.Response, error) {
    req, _ := http.NewRequest(method, url, bytes.NewBuffer(body))
    req.Header.Set("Content-Type", "application/json")
    if config.APIToken != "" {
        req.Header.Set("Authorization", "Bearer "+config.APIToken)
    }

    resp, err := config.HTTPClient.Do(req)
    if err != nil {
        return nil, err
    }
    if resp.StatusCode >= 400 {
        b, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("M-Pesa API error %d: %s", resp.StatusCode, string(b))
    }
    return resp, nil
}
