package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/broswen/eztoll/toll"
)

var ddbClient *dynamodb.Client
var sqsClient *sqs.Client
var tollClient *toll.TollClient

func Handler(ctx context.Context, event events.SQSEvent) error {

	failedRecords := make([]events.SQSMessage, 0)
	for _, record := range event.Records {
		log.Printf("MessageId: %s\n", record.MessageId)

		var paymentRequest toll.PaymentRequest

		if err := json.Unmarshal([]byte(record.Body), &paymentRequest); err != nil {
			log.Printf("unmarshall body: %v\n", err)
			failedRecords = append(failedRecords, record)
			continue
		}

		for _, payment := range paymentRequest.Payments {

			err := tollClient.SubmitPayment(ctx, payment)

			if err != nil {
				log.Printf("UpdateItem: %v\n", err)
				failedRecords = append(failedRecords, record)
				continue
			}
		}
	}

	if len(failedRecords) == len(event.Records) {
		return fmt.Errorf("%d/%d records failed, failing entire batch\n", len(failedRecords), len(event.Records))
	} else if len(failedRecords) > 0 {
		fmt.Printf("%d/%d records failed\n", len(failedRecords), len(event.Records))

		entries := make([]sqstypes.SendMessageBatchRequestEntry, 0)

		// for every failed record, add to send message batch input
		// max of 10 in each event, safe to add all to request
		for _, record := range failedRecords {
			entry := sqstypes.SendMessageBatchRequestEntry{
				Id:          aws.String(record.MessageId),
				MessageBody: aws.String(record.Body),
			}

			entries = append(entries, entry)
		}
		sendMessageBatchInput := sqs.SendMessageBatchInput{
			QueueUrl: aws.String(os.Getenv("PAYMENTDLQ")),
			Entries:  entries,
		}

		_, err := sqsClient.SendMessageBatch(ctx, &sendMessageBatchInput)
		if err != nil {
			// error while sending failed records to DLQ
			// safe to fail lambda, updating payments is idempotent
			return fmt.Errorf("send failed records to DLQ: %v", err)
		}
	}

	return nil
}

func init() {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatal(err)
	}

	ddbClient = dynamodb.NewFromConfig(cfg)
	sqsClient = sqs.NewFromConfig(cfg)
	tollClient = toll.NewClientFromDynamoDB(ddbClient)
}

func main() {
	lambda.Start(Handler)
}
