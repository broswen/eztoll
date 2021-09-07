.PHONY: build clean deploy

build: clean 
	export GO111MODULE=on
	env CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -ldflags="-s -w" -o bin/process_raw_images process_raw_images/main.go
	env CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -ldflags="-s -w" -o bin/process_events process_events/main.go
	env CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -ldflags="-s -w" -o bin/process_payments process_payments/main.go
	env CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -ldflags="-s -w" -o bin/get_tolls get_tolls/main.go
	env CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -ldflags="-s -w" -o bin/post_payment post_payment/main.go

clean:
	rm -rf ./bin 

deploy: clean build
	sls deploy --verbose
