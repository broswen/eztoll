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
	ddbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/rekognition"
	"github.com/aws/aws-sdk-go-v2/service/rekognition/types"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/broswen/eztoll/models"
	"github.com/segmentio/ksuid"
)

var ddbClient *dynamodb.Client
var sqsClient *sqs.Client
var rekogClient *rekognition.Client

func Handler(ctx context.Context, event events.SQSEvent) error {

	failedRecords := make([]events.SQSMessage, 0)

	for _, sqsRecord := range event.Records {
		err := processSQSMessage(ctx, sqsRecord)
		if err != nil {
			failedRecords = append(failedRecords, sqsRecord)
		}
	}

	if len(failedRecords) == len(event.Records) {
		return fmt.Errorf("%d/%d records failed, failing entire batch", len(failedRecords), len(event.Records))
	} else if len(failedRecords) > 0 {
		fmt.Printf("%d/%d records failed", len(failedRecords), len(event.Records))

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

		detectTextInput := rekognition.DetectTextInput{
			Image: &types.Image{
				S3Object: &types.S3Object{
					Bucket: aws.String(bucket),
					Name:   aws.String(key),
				},
			},
			Filters: &types.DetectTextFilters{
				RegionsOfInterest: []types.RegionOfInterest{
					{
						BoundingBox: &types.BoundingBox{
							Height: aws.Float32(0.6),
							Width:  aws.Float32(1.0),
							Left:   aws.Float32(0),
							Top:    aws.Float32(0.25),
						},
					},
				},
				WordFilter: &types.DetectionFilter{
					MinConfidence:       aws.Float32(90),
					MinBoundingBoxWidth: aws.Float32(0.5),
				},
			},
		}

		detectTextResponse, err := rekogClient.DetectText(ctx, &detectTextInput)
		if err != nil {
			return fmt.Errorf("detect text: %v", err)
		}
		var textDetection types.TextDetection
		for _, text := range detectTextResponse.TextDetections {
			if text.Type != types.TextTypesLine {
				continue
			}
			if textDetection.Confidence == nil || *textDetection.Confidence < *text.Confidence {
				textDetection = text
			}
		}

		normalizedPlate := models.NormalizeLicensePlate(*textDetection.DetectedText)
		// mock static cost, should query toll prices api
		cost := 2.0

		id, err := ksuid.NewRandom()
		if err != nil {
			return fmt.Errorf("generate ksuid: %v", err)
		}

		putItemInput := dynamodb.PutItemInput{
			TableName: aws.String(os.Getenv("TOLLTABLE")),
			Item: map[string]ddbtypes.AttributeValue{
				"PK":        &ddbtypes.AttributeValueMemberS{Value: normalizedPlate},
				"SK":        &ddbtypes.AttributeValueMemberS{Value: id.String()},
				"id":        &ddbtypes.AttributeValueMemberS{Value: id.String()},
				"timestamp": &ddbtypes.AttributeValueMemberS{Value: timestamp.Format(time.RFC3339)},
				"plate_num": &ddbtypes.AttributeValueMemberS{Value: normalizedPlate},
				"toll_id":   &ddbtypes.AttributeValueMemberS{Value: toll_id},
				"cost":      &ddbtypes.AttributeValueMemberN{Value: fmt.Sprintf("%.2f", cost)},
				"image_key": &ddbtypes.AttributeValueMemberS{Value: key},
			},
		}

		_, err = ddbClient.PutItem(ctx, &putItemInput)
		if err != nil {
			return fmt.Errorf("PutItem: %v", err)
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
}

func main() {
	lambda.Start(Handler)
}
