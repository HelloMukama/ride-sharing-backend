package main

import (
	"errors"
)

func ProcessPayment(rideID string, amount float64) (string, error) {
	// Stub implementation for interview demo
	if amount <= 0 {
		return "", errors.New("invalid amount")
	}
	return "pmt_" + rideID[:8], nil
}
