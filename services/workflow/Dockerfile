# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY protos/ ./protos/

COPY services/. ./services/

RUN CGO_ENABLED=0 GOOS=linux go build -o /workflow ./services/workflow/cmd/main.go

# Runtime stage
FROM scratch

COPY --from=builder /workflow /workflow

EXPOSE 50052
# For tests:
EXPOSE 60052

# Set the WORKFLOW_API_TOKENS environment variable at runtime to configure authentication tokens.
# Example: docker run -e WORKFLOW_API_TOKENS="admin:supersecrettoken,user:othertoken" ...

ENTRYPOINT ["/workflow"]