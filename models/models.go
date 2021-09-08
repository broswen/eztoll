package models

import (
	"strings"
	"time"
)

type GetTollsRequest struct {
	PlateNumber string `json:"plateNumber"`
}

type GetTollsResponse struct {
	Tolls []Toll `json:"tolls"`
}

type Toll struct {
	Id          string    `json:"id"`
	Timestamp   time.Time `json:"timestamp"`
	PlateNumber string    `json:"plateNumber"`
	TollId      string    `json:"tollId"`
	PaymentId   string    `json:"paymentId"`
	Cost        float64   `json:"cost"`
}

type Payment struct {
	PaymentId string `json:"paymentId"`
	TollId    string `json:"tollId"`
}

type PaymentRequest struct {
	Payments []Payment `json:"payments"`
}

func NormalizeLicensePlate(value string) string {
	// normalize by removing all spaces and dashes, then uppercase all characters
	value = strings.ReplaceAll(value, " ", "")
	value = strings.ReplaceAll(value, "-", "")
	return strings.ToUpper(value)
}
