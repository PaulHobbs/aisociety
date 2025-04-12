# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY protos/ ./protos/

COPY services/node/. ./services/node/

RUN CGO_ENABLED=0 GOOS=linux go build -o /node ./services/node/cmd/server.go

# Runtime stage
FROM scratch

# Copy the built binary into the final image
COPY --from=builder /node /node

EXPOSE 50051
EXPOSE 60051

# ENTRYPOINT remains unchanged for container startup
ENTRYPOINT ["/node"]