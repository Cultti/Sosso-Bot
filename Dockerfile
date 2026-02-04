# Stage 1: Build the Go binary
FROM golang:1.25-alpine AS builder

# Set working directory
WORKDIR /app

# Copy Go modules manifests
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the Go binary with optimizations
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bot .

# Stage 2: Create minimal runtime image
FROM alpine:3.20

# Set working directory
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/bot .

# Declare data volume for persistence
VOLUME /app/data

# Set environment variables (can be overridden at runtime)
ENV DISCORD_BOT_TOKEN=""
ENV WEBHOOK_AUTH_TOKEN=""
ENV FACEIT_API_KEY=""
ENV DISCORD_MATCH_CHANNEL=""

# Expose the port for the webhook server
EXPOSE 8080

# Run the bot
CMD ["./bot"]