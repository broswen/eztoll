package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/broswen/eztoll/models"
)

var ddbClient *dynamodb.Client

type Response events.APIGatewayProxyResponse

func Handler(ctx context.Context, event events.APIGatewayProxyRequest) (Response, error) {

	request := models.GetTollsRequest{
		PlateNumber: event.PathParameters["id"],
	}

	if request.PlateNumber == "" {
		return Response{
			StatusCode: 400,
			Body:       "invalid plate number",
		}, nil
	}

	normalizePlate := models.NormalizeLicensePlate(request.PlateNumber)

	queryInput := dynamodb.QueryInput{
		TableName:              aws.String(os.Getenv("TOLLTABLE")),
		KeyConditionExpression: aws.String("PK = :p"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":p": &types.AttributeValueMemberS{Value: normalizePlate},
		},
	}

	queryResponse, err := ddbClient.Query(ctx, &queryInput)
	if err != nil {
		log.Fatal(err)
	}

	tolls := make([]models.Toll, 0)

	for _, v := range queryResponse.Items {
		cost, err := strconv.ParseFloat(v["cost"].(*types.AttributeValueMemberN).Value, 64)
		if err != nil {
			log.Println(err.Error())
			continue
		}
		timestamp, err := time.Parse(time.RFC3339, v["timestamp"].(*types.AttributeValueMemberS).Value)
		if err != nil {
			log.Println(err.Error())
			continue
		}
		toll := models.Toll{
			Id:          v["id"].(*types.AttributeValueMemberS).Value,
			Timestamp:   timestamp,
			PlateNumber: v["plate_num"].(*types.AttributeValueMemberS).Value,
			TollId:      v["toll_id"].(*types.AttributeValueMemberS).Value,
			Cost:        cost,
		}
		if value, ok := v["payment_id"]; ok {
			toll.PaymentId = value.(*types.AttributeValueMemberS).Value
		}
		tolls = append(tolls, toll)
	}

	response := models.GetTollsResponse{
		Tolls: tolls,
	}

	j, err := json.Marshal(response)
	if err != nil {
		log.Fatal(err)
	}

	resp := Response{
		StatusCode:      200,
		IsBase64Encoded: false,
		Body:            string(j),
	}

	return resp, nil
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
