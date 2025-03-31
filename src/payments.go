package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"
	"bytes"
)

type FlutterwavePaymentRequest struct {
	TxRef    string  `json:"tx_ref"`
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
	Email    string  `json:"email"`
	Phone    string  `json:"phone_number,omitempty"`
	RideID   string  `json:"ride_id"`
}

type FlutterwavePaymentResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    struct {
		Link string `json:"link"` // Payment link
	} `json:"data"`
}

func ProcessPayment(rideID string, amount float64, userEmail string) (string, error) {
	// Initialize Flutterwave config
	flutterwaveSecretKey := os.Getenv("FLUTTERWAVE_SECRET_KEY")
	if flutterwaveSecretKey == "" {
		return "", errors.New("flutterwave secret key not configured")
	}

	// Create payment request
	paymentReq := FlutterwavePaymentRequest{
		TxRef:    fmt.Sprintf("ride-%s-%d", rideID, time.Now().Unix()),
		Amount:   amount,
		Currency: "UGX", // Ugandan Shillings
		Email:    userEmail,
		RideID:   rideID,
	}

	// Convert to JSON
	reqBody, err := json.Marshal(paymentReq)
	if err != nil {
		return "", fmt.Errorf("failed to create payment request: %w", err)
	}

	// Create HTTP request
	client := &http.Client{}
	req, err := http.NewRequest("POST", "https://api.flutterwave.com/v3/payments", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+flutterwaveSecretKey)
	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send payment request: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var paymentResp FlutterwavePaymentResponse
	if err := json.NewDecoder(resp.Body).Decode(&paymentResp); err != nil {
		return "", fmt.Errorf("failed to decode payment response: %w", err)
	}

	if paymentResp.Status != "success" {
		return "", fmt.Errorf("payment failed: %s", paymentResp.Message)
	}

	return paymentResp.Data.Link, nil
}

func VerifyPayment(txRef string) (bool, error) {
	flutterwaveSecretKey := os.Getenv("FLUTTERWAVE_SECRET_KEY")
	url := fmt.Sprintf("https://api.flutterwave.com/v3/transactions/verify_by_reference?tx_ref=%s", txRef)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, err
	}

	req.Header.Set("Authorization", "Bearer "+flutterwaveSecretKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	var verificationResponse struct {
		Status string `json:"status"`
		Data   struct {
			Status string `json:"status"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&verificationResponse); err != nil {
		return false, err
	}

	return verificationResponse.Data.Status == "successful", nil
}

func ProcessMTNPayment(phone string, amount float64) (string, error) {
    return "", errors.New("MTN payment not implemented")
}

func ProcessAirtelPayment(phone string, amount float64) (string, error) {
    return "", errors.New("Airtel payment not implemented")
}

func ProcessChipperPayment(email string, amount float64) (string, error) {
    return "", errors.New("Chipper payment not implemented")
}
