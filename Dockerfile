# Use official Go image
FROM golang:1.25-alpine

# Set working directory
WORKDIR /app

# Copy Go modules manifests
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the Go binary
RUN go build -o bot .

# Set environment variables (can be overridden at runtime)
ENV DISCORD_BOT_TOKEN=""
ENV DISCORD_GUILD_ID=""

# Run the bot
CMD ["./bot"]
