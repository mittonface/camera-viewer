# Discord Notifier for Camera Viewer

This is a standalone Go script that monitors an S3 bucket for new video files and sends notifications to a Discord channel via webhook.

## Features

- Polls S3 bucket for new MP4 videos from the last 2 days
- Sends formatted Discord webhook notifications for each new video
- Uses SQLite database to track already-notified videos (prevents duplicates)
- Automatically cleans up old database entries after 7 days
- Can be run via cron for periodic checking

## Setup

1. Copy `.env.example` to `.env` and configure your settings:

   ```bash
   cp .env.example .env
   ```

2. Edit `.env` with your configuration:

   - `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY`: Your AWS credentials
   - `BUCKET_NAME`: The S3 bucket to monitor
   - `DISCORD_WEBHOOK_URL`: Your Discord webhook URL

3. Install dependencies:

   ```bash
   go mod download
   ```

4. Build the binary:
   ```bash
   go build -o discord-notifier
   ```

## Usage

### Manual Run

```bash
./discord-notifier
```

### Cron Setup

To run every 5 minutes, add this to your crontab (`crontab -e`):

```cron
*/5 * * * * cd /path/to/discord-notifier && ./discord-notifier >> notifier.log 2>&1
```

Or with environment variables directly in cron:

```cron
*/5 * * * * cd /path/to/discord-notifier && AWS_ACCESS_KEY_ID=xxx AWS_SECRET_ACCESS_KEY=yyy BUCKET_NAME=zzz DISCORD_WEBHOOK_URL=www ./discord-notifier >> notifier.log 2>&1
```

## Docker Usage

You can also run this in Docker alongside the main camera-viewer application:

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY discord-notifier/go.mod discord-notifier/go.sum ./
RUN go mod download
COPY discord-notifier/main.go ./
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o discord-notifier .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/discord-notifier .
CMD ["./discord-notifier"]
```

## Environment Variables

| Variable                | Description             | Default                 |
| ----------------------- | ----------------------- | ----------------------- |
| `AWS_REGION`            | AWS region for S3       | `us-east-1`             |
| `AWS_ACCESS_KEY_ID`     | AWS access key          | (required)              |
| `AWS_SECRET_ACCESS_KEY` | AWS secret key          | (required)              |
| `BUCKET_NAME`           | S3 bucket name          | (required)              |
| `DISCORD_WEBHOOK_URL`   | Discord webhook URL     | (required)              |
| `NOTIFIER_DB_PATH`      | Path to SQLite database | `./discord-notifier.db` |

## How It Works

1. The script checks the S3 bucket for MP4 files in today's and yesterday's date folders (format: `YYYY/MM/DD/`)
2. For each video found, it checks the SQLite database to see if a notification was already sent
3. If not already notified, it sends a Discord webhook message with video details
4. The video is then marked as notified in the database
5. Old database entries (>7 days) are automatically cleaned up

## Discord Webhook Format

The notification includes:

- Video upload date
- Filename
- File size
- Full S3 key
- Upload timestamp
