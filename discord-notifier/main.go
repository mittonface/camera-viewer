package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
)

type DiscordWebhookMessage struct {
	Content string         `json:"content,omitempty"`
	Embeds  []DiscordEmbed `json:"embeds,omitempty"`
}

type DiscordEmbed struct {
	Title       string         `json:"title,omitempty"`
	Description string         `json:"description,omitempty"`
	Color       int            `json:"color,omitempty"`
	Fields      []DiscordField `json:"fields,omitempty"`
	Timestamp   string         `json:"timestamp,omitempty"`
	Footer      *DiscordFooter `json:"footer,omitempty"`
}

type DiscordField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

type DiscordFooter struct {
	Text string `json:"text"`
}

func main() {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Get configuration from environment
	awsRegion := getEnv("AWS_REGION", "us-east-1")
	awsAccessKeyID := os.Getenv("AWS_ACCESS_KEY_ID")
	awsSecretAccessKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	bucketName := os.Getenv("BUCKET_NAME")
	discordWebhookURL := os.Getenv("DISCORD_WEBHOOK_URL")
	dbPath := getEnv("NOTIFIER_DB_PATH", "./discord-notifier.db")

	// Validate required configuration
	if bucketName == "" {
		log.Fatal("BUCKET_NAME environment variable is required")
	}
	if discordWebhookURL == "" {
		log.Fatal("DISCORD_WEBHOOK_URL environment variable is required")
	}

	// Initialize database
	db, err := initDatabase(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize AWS S3 client
	ctx := context.Background()
	var cfg aws.Config

	if awsAccessKeyID != "" && awsSecretAccessKey != "" {
		cfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(awsRegion),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				awsAccessKeyID,
				awsSecretAccessKey,
				"",
			)),
		)
	} else {
		cfg, err = config.LoadDefaultConfig(ctx, config.WithRegion(awsRegion))
	}

	if err != nil {
		log.Fatalf("Unable to load AWS config: %v", err)
	}

	s3Client := s3.NewFromConfig(cfg)

	// Check videos from the last 2 days
	log.Println("Checking for new videos...")
	now := time.Now()
	dates := []time.Time{now, now.AddDate(0, 0, -1)}

	newVideosFound := 0
	for _, date := range dates {
		prefix := fmt.Sprintf("%04d/%02d/%02d/", date.Year(), date.Month(), date.Day())
		
		result, err := s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket: aws.String(bucketName),
			Prefix: aws.String(prefix),
		})

		if err != nil {
			log.Printf("Error listing objects for prefix %s: %v", prefix, err)
			continue
		}

		for _, obj := range result.Contents {
			if obj.Key != nil && strings.HasSuffix(*obj.Key, ".mp4") {
				// Check if we've already posted about this video
				posted, err := isVideoPosted(db, *obj.Key)
				if err != nil {
					log.Printf("Error checking if video was posted: %v", err)
					continue
				}

				if !posted {
					// Send Discord notification
					var fileSize int64 = 0
					if obj.Size != nil {
						fileSize = *obj.Size
					}

					lastModified := time.Now()
					if obj.LastModified != nil {
						lastModified = *obj.LastModified
					}

					if err := sendDiscordNotification(discordWebhookURL, bucketName, *obj.Key, fileSize, lastModified); err != nil {
						log.Printf("Failed to send Discord notification for %s: %v", *obj.Key, err)
					} else {
						// Mark video as posted
						if err := markVideoPosted(db, *obj.Key); err != nil {
							log.Printf("Failed to mark video as posted: %v", err)
						} else {
							log.Printf("Successfully notified about new video: %s", *obj.Key)
							newVideosFound++
							// Sleep for 2 seconds to avoid Discord rate limits
							time.Sleep(2 * time.Second)
						}
					}
				}
			}
		}
	}

	if newVideosFound == 0 {
		log.Println("No new videos found")
	} else {
		log.Printf("Found and notified about %d new video(s)", newVideosFound)
	}

	// Clean up old entries (older than 7 days)
	if err := cleanupOldEntries(db); err != nil {
		log.Printf("Failed to cleanup old entries: %v", err)
	}
}

func initDatabase(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	// Create table if it doesn't exist
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS posted_videos (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		s3_key TEXT UNIQUE NOT NULL,
		posted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_posted_at ON posted_videos(posted_at);
	`

	if _, err := db.Exec(createTableSQL); err != nil {
		return nil, err
	}

	return db, nil
}

func isVideoPosted(db *sql.DB, s3Key string) (bool, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM posted_videos WHERE s3_key = ?", s3Key).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func markVideoPosted(db *sql.DB, s3Key string) error {
	_, err := db.Exec("INSERT INTO posted_videos (s3_key) VALUES (?)", s3Key)
	return err
}

func cleanupOldEntries(db *sql.DB) error {
	// Delete entries older than 7 days
	_, err := db.Exec("DELETE FROM posted_videos WHERE posted_at < datetime('now', '-7 days')")
	return err
}

func sendDiscordNotification(webhookURL, bucketName, videoKey string, fileSize int64, lastModified time.Time) error {
	// Extract date and filename from the key (format: YYYY/MM/DD/filename.mp4)
	date := "Unknown"
	filename := videoKey
	if len(videoKey) > 10 && videoKey[4] == '/' && videoKey[7] == '/' {
		date = fmt.Sprintf("%s-%s-%s", videoKey[0:4], videoKey[5:7], videoKey[8:10])
		if len(videoKey) > 11 {
			filename = videoKey[11:]
		}
	}

	// Format file size
	sizeStr := formatFileSize(fileSize)

	// Create embed message
	embed := DiscordEmbed{
		Title:       "📹 New Video Uploaded",
		Description: fmt.Sprintf("A new video has been uploaded to S3 bucket `%s`", bucketName),
		Color:       0x00ff00, // Green color
		Fields: []DiscordField{
			{
				Name:   "📅 Date",
				Value:  date,
				Inline: true,
			},
			{
				Name:   "📁 Filename",
				Value:  filename,
				Inline: true,
			},
			{
				Name:   "📊 Size",
				Value:  sizeStr,
				Inline: true,
			},
			{
				Name:   "🗂️ S3 Key",
				Value:  fmt.Sprintf("`%s`", videoKey),
				Inline: false,
			},
		},
		Timestamp: lastModified.Format(time.RFC3339),
		Footer: &DiscordFooter{
			Text: "Camera Viewer S3 Monitor",
		},
	}

	message := DiscordWebhookMessage{
		Embeds: []DiscordEmbed{embed},
	}

	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal discord message: %w", err)
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("POST", webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send discord notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("discord webhook returned status %d", resp.StatusCode)
	}

	return nil
}

func formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}