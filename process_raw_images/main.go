package main

import (
	"context"
	"os"
	"strings"
	"time"

	"encoding/json"
	"fmt"
	"log"
	"net/url"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/rekognition"
	rekogtypes "github.com/aws/aws-sdk-go-v2/service/rekognition/types"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/broswen/eztoll/toll"
	"github.com/segmentio/ksuid"
)

var ddbClient *dynamodb.Client
var sqsClient *sqs.Client
var rekogClient *rekognition.Client
var tollClient *toll.TollClient

func Handler(ctx context.Context, event events.SQSEvent) error {

	failedRecords := make([]events.SQSMessage, 0)

	for _, sqsRecord := range event.Records {
		err := processSQSMessage(ctx, sqsRecord)
		if err != nil {
			fmt.Println(err)
			failedRecords = append(failedRecords, sqsRecord)
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
			QueueUrl: aws.String(os.Getenv("RAWIMAGEDLQ")),
			Entries:  entries,
		}

		_, err := sqsClient.SendMessageBatch(ctx, &sendMessageBatchInput)
		if err != nil {
			// error while sending failed records to DLQ
			// safe to fail lambda, updating tolls is idempotent
			return fmt.Errorf("send failed records to DLQ: %v", err)
		}
	}

	return nil
}

func processSQSMessage(ctx context.Context, message events.SQSMessage) error {
	var s3Event events.S3Event
	if err := json.Unmarshal([]byte(message.Body), &s3Event); err != nil {
		return fmt.Errorf("unmarshall body: %v", err)
	}
	for _, s3Record := range s3Event.Records {
		key, err := url.QueryUnescape(s3Record.S3.Object.Key)
		if err != nil {
			return fmt.Errorf("unescape object key: %v", err)
		}
		bucket, err := url.QueryUnescape(s3Record.S3.Bucket.Name)
		if err != nil {
			return fmt.Errorf("unescape object key: %v", err)
		}

		keyParts := strings.Split(key, "/")
		toll_id := keyParts[0]
		timestamp, err := time.Parse(time.RFC3339, strings.Split(keyParts[1], ".")[0])
		if err != nil {
			return fmt.Errorf("parse timestamp: %v", err)
		}

		image := &rekogtypes.Image{
			S3Object: &rekogtypes.S3Object{
				Bucket: aws.String(bucket),
				Name:   aws.String(key),
			},
		}

		detectedPlate, err := tollClient.DetectText(ctx, rekogClient, image)
		if err != nil {
			return fmt.Errorf("detect plate: %v", err)
		}

		normalizedPlate := toll.NormalizeLicensePlate(detectedPlate)
		// mock static cost, should query toll prices api
		cost := 2.0

		id, err := ksuid.NewRandom()
		if err != nil {
			return fmt.Errorf("generate ksuid: %v", err)
		}

		newToll := toll.Toll{
			Id:          id.String(),
			Timestamp:   timestamp,
			PlateNumber: normalizedPlate,
			TollId:      toll_id,
			Cost:        cost,
			ImageKey:    key,
		}

		err = tollClient.SubmitToll(ctx, newToll)
		if err != nil {
			return fmt.Errorf("submit toll: %v", err)
		}

	}
	return nil
}

func init() {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatal(err)
	}

	rekogClient = rekognition.NewFromConfig(cfg)
	ddbClient = dynamodb.NewFromConfig(cfg)
	sqsClient = sqs.NewFromConfig(cfg)
	tollClient = toll.NewClientFromDynamoDB(ddbClient)
}

func main() {
	lambda.Start(Handler)
}
