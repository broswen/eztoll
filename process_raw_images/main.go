package main

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/rekognition"
)

var ddbClient *dynamodb.Client
var rekogClient *rekognition.Client

func Handler(ctx context.Context, event events.SQSEvent) error {
	fmt.Printf("%+v\n", event)

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
