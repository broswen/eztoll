package main

import (
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func HandleRequest(event events.KinesisFirehoseEvent) (events.KinesisFirehoseResponse, error) {
	// create empty list of records to return
	records := make([]events.KinesisFirehoseResponseRecord, 0)
	for _, record := range event.Records {

		// populate record, append newline to bytes (already decoded from base64)
		responseRecord := events.KinesisFirehoseResponseRecord{
			RecordID: record.RecordID,
			Result:   events.KinesisFirehoseTransformedStateOk,
			Data:     append(record.Data, []byte("\n")...),
		}
		records = append(records, responseRecord)
	}

	return events.KinesisFirehoseResponse{
		Records: records,
	}, nil
}

func main() {
	lambda.Start(HandleRequest)
}
