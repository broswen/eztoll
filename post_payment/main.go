package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/broswen/eztoll/toll"
)

var sqsClient *sqs.Client

// Response is of type APIGatewayProxyResponse since we're leveraging the
// AWS Lambda Proxy Request functionality (default behavior)
//
// https://serverless.com/framework/docs/providers/aws/events/apigateway/#lambda-proxy-integration
type Response events.APIGatewayProxyResponse

// Handler is our lambda handler invoked by the `lambda.Start` function call
func Handler(ctx context.Context, event events.APIGatewayProxyRequest) (Response, error) {
	var paymentRequest toll.PaymentRequest
	if err := json.Unmarshal([]byte(event.Body), &paymentRequest); err != nil {
		log.Println(err)
		return Response{
			StatusCode: http.StatusBadRequest,
			Body:       "error unmarshalling request",
		}, nil
	}

	if err := paymentRequest.Validate(); err != nil {
		log.Println(err)
		return Response{
			StatusCode: http.StatusBadRequest,
			Body:       err.Error(),
		}, nil
	}

	j, err := json.Marshal(paymentRequest)
	if err != nil {
		log.Println(err)
		return Response{
			StatusCode: http.StatusInternalServerError,
			Body:       "error marshalling response",
		}, nil
	}

	sendMessageInput := sqs.SendMessageInput{
		QueueUrl:    aws.String(os.Getenv("PAYMENTQUEUE")),
		MessageBody: aws.String(string(j)),
	}

	_, err = sqsClient.SendMessage(ctx, &sendMessageInput)
	if err != nil {
		log.Println(err)
		return Response{
			StatusCode: http.StatusInternalServerError,
			Body:       "error submitting request",
		}, nil
	}

	return Response{
		StatusCode: 200,
	}, nil
}

func init() {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatal(err)
	}

	sqsClient = sqs.NewFromConfig(cfg)
}

func main() {
	lambda.Start(Handler)
}
