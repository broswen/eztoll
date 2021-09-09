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
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/broswen/eztoll/models"
)

var ddbClient *dynamodb.Client

func Handler(ctx context.Context, event events.SQSEvent) error {
	fmt.Printf("%+v\n", event)

	for _, record := range event.Records {
		var paymentRequest models.PaymentRequest
		if err := json.Unmarshal([]byte(record.Body), &paymentRequest); err != nil {
			log.Fatal(err)
		}

		for _, payment := range paymentRequest.Payments {
			fmt.Printf("%+v\n", payment)
			// update dynamodb item where plate_num and id match
			// set payment_id
			updateItemInput := dynamodb.UpdateItemInput{
				TableName: aws.String(os.Getenv("TOLLTABLE")),
				Key: map[string]types.AttributeValue{
					"PK": &types.AttributeValueMemberS{Value: payment.PlateNumber},
					"SK": &types.AttributeValueMemberS{Value: payment.Id},
				},
				UpdateExpression: "SET #p = :p",
				ExpressionAttributeNames: map[string]string{
					"#p": "payment_id",
				},
				ExpressionAttributeValues: map[string]types.AttributeValue{
					":p": &types.AttributeValueMemberS{Value: payment.PaymentId}
				},
			}
			_, err := ddbClient.UpdateItem(ctx, &updateItemInput)
			if err != nil {
				log.Fatal(err)
			}
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
}

func main() {
	lambda.Start(Handler)
}
