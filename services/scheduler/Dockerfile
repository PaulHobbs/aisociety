# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY protos/ ./protos/
COPY services/ ./services/

RUN CGO_ENABLED=0 GOOS=linux go build -o /scheduler-service ./services/scheduler/cmd/server.go

# Runtime stage
FROM scratch

COPY --from=builder /scheduler-service /scheduler-service

EXPOSE 50053

ENTRYPOINT ["/scheduler-service"]