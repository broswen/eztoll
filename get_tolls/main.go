package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/broswen/eztoll/toll"
)

var ddbClient *dynamodb.Client
var tollClient *toll.TollClient

type Response events.APIGatewayProxyResponse

func Handler(ctx context.Context, event events.APIGatewayProxyRequest) (Response, error) {

	request := toll.GetTollsRequest{
		PlateNumber: event.PathParameters["id"],
	}

	if request.PlateNumber == "" {
		return Response{
			StatusCode: http.StatusBadRequest,
			Body:       "invalid plate number",
		}, nil
	}

	normalizedPlate := toll.NormalizeLicensePlate(request.PlateNumber)

	tolls, err := tollClient.GetByPlate(ctx, normalizedPlate)
	if err != nil {
		log.Printf("GetByPlate: %v\n", err)
		return Response{
			StatusCode: http.StatusInternalServerError,
			Body:       "error getting tolls",
		}, nil
	}

	response := toll.GetTollsResponse{
		Tolls: tolls,
	}

	j, err := json.Marshal(response)
	if err != nil {
		log.Println(err)
		return Response{
			StatusCode: http.StatusInternalServerError,
			Body:       "error marshalling response",
		}, nil
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
	tollClient = toll.NewClientFromDynamoDB(ddbClient)
}

func main() {
	lambda.Start(Handler)
}
