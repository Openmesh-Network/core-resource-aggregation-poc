#!/bin/sh
CGO_ENABLED=0 GOOS=linux go build -o resource-aggregation-poc && docker build -t xnode:latest .
