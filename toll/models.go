package toll

import (
	"errors"
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
	Id          string    `json:"id" dynamodbav:"id"`
	Timestamp   time.Time `json:"timestamp" dynamodbav:"timestamp"`
	PlateNumber string    `json:"plateNumber" dynamodbav:"plate_num"`
	TollId      string    `json:"tollId" dynamodbav:"toll_id"`
	PaymentId   string    `json:"paymentId" dynamodbav:"payment_id"`
	Cost        float64   `json:"cost" dynamodbav:"cost"`
	ImageKey    string    `json:"imageKey" dynamodbav:"image_key"`
}

type Payment struct {
	PaymentId   string `json:"paymentId"`
	PlateNumber string `json:"plateNumber"`
	Id          string `json:"id"`
}

func (p Payment) Validate() error {
	if p.PaymentId == "" {
		return errors.New("payment id is invalid")
	}
	if p.PlateNumber == "" {
		return errors.New("plate number id is invalid")
	}
	if p.Id == "" {
		return errors.New("id is invalid")
	}
	return nil
}

type PaymentRequest struct {
	Payments []Payment `json:"payments"`
}

func (pp PaymentRequest) Validate() error {
	for _, p := range pp.Payments {
		if err := p.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func NormalizeLicensePlate(value string) string {
	// normalize by removing all spaces and dashes, then uppercase all characters
	value = strings.ReplaceAll(value, " ", "")
	value = strings.ReplaceAll(value, "-", "")
	return strings.ToUpper(value)
}
