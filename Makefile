#!/usr/bin/make -f

export CGO_ENABLED=0
export GO111MODULE=on

OUTPUT=bootstrap

default: lint test build

# Run all lint checking with exit codes for CI.
lint:
	golint -set_exit_status `go list ./... | grep -v /vendor/`

# Run go fmt against code
fmt:
	go fmt ./...

# Run tests with coverage reporting.
test:
	go test -cover ./...

build:
	GOARCH=amd64 GOOS=linux go build -tags lambda.norpc -o ${OUTPUT} main.go

# https://github.com/aws/aws-lambda-go#building-your-function
package: build
	zip lambda-handler.zip bootstrap defaults.env