# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY protos/ ./protos/

COPY services/. ./services/

RUN CGO_ENABLED=0 GOOS=linux go build -o /workflow-service ./services/workflow/cmd_workflowserver/main.go

# Runtime stage
FROM scratch

COPY --from=builder /workflow-service /workflow-service

EXPOSE 50052

ENTRYPOINT ["/workflow-service"]