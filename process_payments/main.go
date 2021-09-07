package main

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

var ddbClient *dynamodb.Client

func Handler(ctx context.Context, event events.SQSEvent) error {
	fmt.Printf("%+v\n", event)

	// extract payment id
	// extract list of license plate number + toll id + timestamp

	// post payment id into dynamodb table for toll event(s)

	return nil
}

func init() {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatal(err)
	}

	ddbClient = dynamodb.NewFromConfig(cfg)
}

func main() {
	lambda.Start(Handler)
}
