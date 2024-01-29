FROM golang:1.20-alpine

WORKDIR /xnode
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /resource-aggregation-poc

CMD ["/resource-aggregation-poc"]
