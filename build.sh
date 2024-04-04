#!/bin/sh

# Generate the sources first!
go run util/generate-sources.go

# Compile and turn to docker image.
CGO_ENABLED=0 GOOS=linux go build -o resource-aggregation-poc && docker build -t xnode:latest .
