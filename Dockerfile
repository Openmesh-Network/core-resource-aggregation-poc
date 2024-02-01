FROM golang:1.20-alpine

WORKDIR /xnode
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
COPY resource-aggregation-poc /resource-aggregation-poc

CMD ["/resource-aggregation-poc"]
