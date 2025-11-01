# --------- Build stage ---------
FROM golang:1.25-alpine AS builder
WORKDIR /app

# Copy go modules files first (for caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of your code
COPY . .

# Copy .env into the container
# COPY cmd/grpcapi/.env .env

# Build your server binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o server ./cmd/grpcapi/server.go

# --------- Run stage ---------
FROM gcr.io/distroless/static
WORKDIR /app
COPY --from=builder /app/server .

# Expose ports
EXPOSE 50051
EXPOSE 8080

ENTRYPOINT ["/app/server"]
