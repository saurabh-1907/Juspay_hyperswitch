package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type PaymentIntentRequest struct {
	Amount                   int    `json:"amount"`
	Currency                 string `json:"currency"`
	Confirm                  bool   `json:"confirm"`
	MerchantOrderReferenceID string `json:"merchant_order_reference_id"`
}

type PaymentIntentResponse struct {
	PaymentID    string `json:"payment_id"`
	ClientSecret string `json:"client_secret"`
	Status       string `json:"status"`
}

func getAPIKey() string {
	return os.Getenv("HYPERSWITCH_API_KEY")
}

func getAPIURL() string {
	url := os.Getenv("HYPERSWITCH_API_URL")
	if url == "" {
		return "https://sandbox.hyperswitch.io"
	}
	return url
}

func CreatePaymentIntent(reqData PaymentIntentRequest) (*PaymentIntentResponse, error) {
	reqData.Confirm = false
	
	jsonData, err := json.Marshal(reqData)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/payments", getAPIURL())
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", getAPIKey())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("hyperswitch error: %s", string(body))
	}

	var paymentResp PaymentIntentResponse
	if err := json.Unmarshal(body, &paymentResp); err != nil {
		return nil, err
	}
	
	return &paymentResp, nil
}

func RetrievePaymentIntent(paymentID string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/payments/%s", getAPIURL(), paymentID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", getAPIKey())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("hyperswitch error: %s", string(body))
	}

	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	return data, nil
}
