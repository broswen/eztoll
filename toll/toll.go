package toll

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	ddbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/rekognition"
	"github.com/aws/aws-sdk-go-v2/service/rekognition/types"
	rekogtypes "github.com/aws/aws-sdk-go-v2/service/rekognition/types"
	"github.com/aws/aws-sdk-go/aws"
)

type TollClient struct {
	ddbClient *dynamodb.Client
}

func NewClientFromDynamoDB(ddbClient *dynamodb.Client) *TollClient {
	return &TollClient{
		ddbClient: ddbClient,
	}
}

func (tc TollClient) GetByPlate(ctx context.Context, plateNumber string) ([]Toll, error) {
	queryInput := dynamodb.QueryInput{
		TableName:              aws.String(os.Getenv("TOLLTABLE")),
		KeyConditionExpression: aws.String("PK = :p"),
		ExpressionAttributeValues: map[string]ddbtypes.AttributeValue{
			":p": &ddbtypes.AttributeValueMemberS{Value: plateNumber},
		},
	}

	queryResponse, err := tc.ddbClient.Query(ctx, &queryInput)
	if err != nil {
		return nil, err
	}

	responseTolls := make([]Toll, 0)

	for _, v := range queryResponse.Items {
		var toll Toll
		err = attributevalue.UnmarshalMap(v, &toll)
		if err != nil {
			log.Printf("unmarshal map: %v\n", err)
			continue
		}
		responseTolls = append(responseTolls, toll)
	}
	return responseTolls, nil
}

func (tc TollClient) DetectText(ctx context.Context, rekogClient *rekognition.Client, image *rekogtypes.Image) (string, error) {
	detectTextInput := rekognition.DetectTextInput{
		Image: image,
		Filters: &rekogtypes.DetectTextFilters{
			RegionsOfInterest: []types.RegionOfInterest{
				{
					BoundingBox: &rekogtypes.BoundingBox{
						Height: aws.Float32(0.6),
						Width:  aws.Float32(1.0),
						Left:   aws.Float32(0),
						Top:    aws.Float32(0.25),
					},
				},
			},
			WordFilter: &rekogtypes.DetectionFilter{
				MinConfidence:       aws.Float32(90),
				MinBoundingBoxWidth: aws.Float32(0.5),
			},
		},
	}

	detectTextResponse, err := rekogClient.DetectText(ctx, &detectTextInput)
	if err != nil {
		return "", fmt.Errorf("detect text: %v", err)
	}

	if len(detectTextResponse.TextDetections) == 0 {
		return "", fmt.Errorf("no text detected: %s/%s", *image.S3Object.Bucket, *image.S3Object.Name)
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

	return *textDetection.DetectedText, nil
}

func (tc TollClient) SubmitToll(ctx context.Context, toll Toll) error {
	item, err := attributevalue.MarshalMap(toll)
	if err != nil {
		return err
	}

	putItemInput := dynamodb.PutItemInput{
		TableName: aws.String(os.Getenv("TOLLTABLE")),
		Item:      item,
	}

	_, err = tc.ddbClient.PutItem(ctx, &putItemInput)
	if err != nil {
		return fmt.Errorf("PutItem: %v", err)
	}
	return nil
}

func (tc TollClient) SubmitPayment(ctx context.Context, payment Payment) error {
	updateItemInput := dynamodb.UpdateItemInput{
		TableName: aws.String(os.Getenv("TOLLTABLE")),
		Key: map[string]ddbtypes.AttributeValue{
			"PK": &ddbtypes.AttributeValueMemberS{Value: payment.PlateNumber},
			"SK": &ddbtypes.AttributeValueMemberS{Value: payment.Id},
		},
		UpdateExpression: aws.String("SET #p = :p"),
		ExpressionAttributeNames: map[string]string{
			"#p": "payment_id",
		},
		ExpressionAttributeValues: map[string]ddbtypes.AttributeValue{
			":p": &ddbtypes.AttributeValueMemberS{Value: payment.PaymentId},
		},
		ConditionExpression: aws.String("attribute_exists(PK) AND attribute_exists(SK) AND attribute_not_exists(payment_id)"),
	}
	_, err := tc.ddbClient.UpdateItem(ctx, &updateItemInput)
	return err
}
