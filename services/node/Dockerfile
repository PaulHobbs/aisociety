# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY protos/ ./protos/

COPY services/node/. ./services/node/

RUN CGO_ENABLED=0 GOOS=linux go build -o /node ./services/node/cmd/server.go

# Runtime stage
FROM alpine:latest

# Copy the built binary into the final image
COPY --from=builder /node /node

EXPOSE 50051
EXPOSE 60051

# Copy the entrypoint script into the image
COPY services/node/entrypoint.sh /app/entrypoint.sh

# Make the entrypoint script executable
RUN chmod +x /app/entrypoint.sh

# Set the entrypoint to the script, which will then run the node application
ENTRYPOINT ["/app/entrypoint.sh", "/node"]