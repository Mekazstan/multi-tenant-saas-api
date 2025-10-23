package payment

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type PaystackProvider struct {
	secretKey     string
	webhookSecret string
	baseURL       string
}

func NewPaystackProvider(secretKey, webhookSecret string) *PaystackProvider {
	return &PaystackProvider{
		secretKey:     secretKey,
		webhookSecret: webhookSecret,
		baseURL:       "https://api.paystack.co",
	}
}

type PaystackInitializeParams struct {
	Email       string            `json:"email"`
	Amount      int64             `json:"amount"`
	Currency    string            `json:"currency,omitempty"`
	Reference   string            `json:"reference"`
	CallbackURL string            `json:"callback_url"`
	Metadata    map[string]string `json:"metadata"`
}

type PaystackInitializeResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    struct {
		AuthorizationURL string `json:"authorization_url"`
		AccessCode       string `json:"access_code"`
		Reference        string `json:"reference"`
	} `json:"data"`
}

func (p *PaystackProvider) InitializeTransaction(params PaystackInitializeParams) (*PaystackInitializeResponse, error) {
	url := fmt.Sprintf("%s/transaction/initialize", p.baseURL)

	jsonData, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal params: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.secretKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result PaystackInitializeResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !result.Status {
		return nil, fmt.Errorf("paystack error: %s", result.Message)
	}

	return &result, nil
}

func (p *PaystackProvider) VerifyWebhookSignature(payload []byte, signature string) bool {
	mac := hmac.New(sha512.New, []byte(p.webhookSecret))
	mac.Write(payload)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

type PaystackWebhookEvent struct {
	Event string                 `json:"event"`
	Data  map[string]interface{} `json:"data"`
}

func (p *PaystackProvider) ParseWebhookEvent(payload []byte) (*PaystackWebhookEvent, error) {
	var event PaystackWebhookEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, fmt.Errorf("failed to parse webhook event: %w", err)
	}
	return &event, nil
}
