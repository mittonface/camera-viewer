# Build stage
FROM golang:1.21-alpine AS builder

# Set working directory
WORKDIR /app

# Install git and build dependencies
RUN apk add --no-cache git gcc musl-dev sqlite-dev

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application with CGO enabled for SQLite
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o discord-notifier .

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests, sqlite libs, and busybox-suid for cron
RUN apk --no-cache add ca-certificates sqlite-libs busybox-suid

# Create app directory
WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/discord-notifier .

# Create directory for database
RUN mkdir -p /data

# Add cron job to run every 5 minutes
RUN echo "*/5 * * * * cd /app && ./discord-notifier >> /var/log/discord-notifier.log 2>&1" > /etc/crontabs/root

# Create log file
RUN touch /var/log/discord-notifier.log

# Start cron in the foreground
CMD ["crond", "-f", "-d", "8"]