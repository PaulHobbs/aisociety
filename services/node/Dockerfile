# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY protos/ ./protos/

COPY services/node/. ./services/node/

RUN CGO_ENABLED=0 GOOS=linux go build -o /node-service ./services/node/cmd/server.go

# Runtime stage
FROM scratch

COPY --from=builder /node-service /node-service

EXPOSE 50051

ENTRYPOINT ["/node-service"]