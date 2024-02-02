FROM golang:1.20-alpine

WORKDIR /xnode
COPY . ./
COPY resource-aggregation-poc /resource-aggregation-poc

CMD ["/resource-aggregation-poc"]
