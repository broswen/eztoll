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
	"github.com/aws/aws-sdk-go/aws"
	"github.com/broswen/eztoll/models"
)

var ddbClient *dynamodb.Client
var rekogClient *rekognition.Client

func Handler(ctx context.Context, event events.SQSEvent) error {

	for _, sqsRecord := range event.Records {
		var s3Event events.S3Event
		if err := json.Unmarshal([]byte(sqsRecord.Body), &s3Event); err != nil {
			log.Fatal(err)
		}
		for _, s3Record := range s3Event.Records {
			key, err := url.QueryUnescape(s3Record.S3.Object.Key)
			if err != nil {
				log.Fatal(err)
			}
			bucket, err := url.QueryUnescape(s3Record.S3.Bucket.Name)
			if err != nil {
				log.Fatal(err)
			}

			keyParts := strings.Split(key, "/")
			toll_id := keyParts[0]
			timestamp, err := time.Parse(time.RFC3339, strings.Split(keyParts[1], ".")[0])
			if err != nil {
				log.Fatal(err)
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
				log.Fatal(err)
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

			putItemInput := dynamodb.PutItemInput{
				TableName: aws.String(os.Getenv("TOLLTABLE")),
				Item: map[string]ddbtypes.AttributeValue{
					"PK":        &ddbtypes.AttributeValueMemberS{Value: normalizedPlate},
					"SK":        &ddbtypes.AttributeValueMemberS{Value: timestamp.Format(time.RFC3339)},
					"id":        &ddbtypes.AttributeValueMemberS{Value: fmt.Sprintf("%s#%s", timestamp.Format(time.RFC3339), normalizedPlate)},
					"timestamp": &ddbtypes.AttributeValueMemberS{Value: timestamp.Format(time.RFC3339)},
					"plate_num": &ddbtypes.AttributeValueMemberS{Value: normalizedPlate},
					"toll_id":   &ddbtypes.AttributeValueMemberS{Value: toll_id},
					"cost":      &ddbtypes.AttributeValueMemberN{Value: fmt.Sprintf("%.2f", cost)},
					"image_key": &ddbtypes.AttributeValueMemberS{Value: key},
				},
			}

			_, err = ddbClient.PutItem(ctx, &putItemInput)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(toll_id)
			fmt.Println(timestamp.Format(time.RFC3339))
			fmt.Println(key)
			fmt.Println(normalizedPlate)
		}
	}

	// extract toll booth id
	// extract timestamp and convert to time object
	// use s3 bucket + key with rekognition to get text
	// get largest text box and use as license plate number
	// mock current toll price api
	// unique id is license plate number + toll id + timestamp
	// post all info into dynamodb table
	// make sure toll id/license number/timestamp are unique (because non FIFO sqs queue)
	return nil
}

func init() {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatal(err)
	}

	rekogClient = rekognition.NewFromConfig(cfg)
	ddbClient = dynamodb.NewFromConfig(cfg)
}

func main() {
	lambda.Start(Handler)
}
