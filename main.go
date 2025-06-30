package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/joho/godotenv"
	"path/filepath"
	"strings"
	"time"
)

// basicAuth middleware to protect endpoints
func basicAuth(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := os.Getenv("USERNAME")
		password := os.Getenv("PASSWORD")
		
		// If no credentials are set, allow access (for backward compatibility)
		if username == "" || password == "" {
			handler(w, r)
			return
		}
		
		// Get the Basic Authentication credentials
		user, pass, ok := r.BasicAuth()
		
		if !ok || user != username || pass != password {
			// Set WWW-Authenticate header to prompt for credentials
			w.Header().Set("WWW-Authenticate", `Basic realm="Camera Viewer"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		
		handler(w, r)
	}
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	ctx := context.Background()
	var cfg aws.Config
	var err error

	if os.Getenv("AWS_ACCESS_KEY_ID") != "" && os.Getenv("AWS_SECRET_ACCESS_KEY") != "" {
		cfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(os.Getenv("AWS_REGION")),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				os.Getenv("AWS_ACCESS_KEY_ID"),
				os.Getenv("AWS_SECRET_ACCESS_KEY"),
				"",
			)),
		)
	} else {
		cfg, err = config.LoadDefaultConfig(ctx)
	}

	if err != nil {
		log.Fatal("Unable to load AWS config:", err)
	}

	s3Client := s3.NewFromConfig(cfg)

	http.HandleFunc("/list-bucket", basicAuth(func(w http.ResponseWriter, r *http.Request) {
		bucketName := os.Getenv("BUCKET_NAME")
		if bucketName == "" {
			http.Error(w, "BUCKET_NAME environment variable is not set", http.StatusInternalServerError)
			return
		}

		result, err := s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket: aws.String(bucketName),
		})

		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to list objects: %v", err), http.StatusInternalServerError)
			return
		}

		var objects []string
		for _, obj := range result.Contents {
			objects = append(objects, *obj.Key)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"bucket": bucketName,
			"objects": objects,
			"count": len(objects),
		})
	}))

	http.HandleFunc("/list-files-by-date", basicAuth(func(w http.ResponseWriter, r *http.Request) {
		bucketName := os.Getenv("BUCKET_NAME")
		if bucketName == "" {
			http.Error(w, "BUCKET_NAME environment variable is not set", http.StatusInternalServerError)
			return
		}

		year := r.URL.Query().Get("year")
		month := r.URL.Query().Get("month")
		day := r.URL.Query().Get("day")

		if year == "" || month == "" || day == "" {
			http.Error(w, "year, month, and day query parameters are required", http.StatusBadRequest)
			return
		}

		prefix := fmt.Sprintf("%s/%s/%s/", year, month, day)

		result, err := s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket: aws.String(bucketName),
			Prefix: aws.String(prefix),
		})

		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to list objects: %v", err), http.StatusInternalServerError)
			return
		}

		var files []map[string]interface{}
		for _, obj := range result.Contents {
			if strings.HasSuffix(*obj.Key, ".mp4") {
				fileInfo := map[string]interface{}{
					"key":          *obj.Key,
					"filename":     filepath.Base(*obj.Key),
					"size":         *obj.Size,
					"lastModified": obj.LastModified,
				}
				
				// Add storage class information
				fileInfo["storageClass"] = string(obj.StorageClass)
				
				files = append(files, fileInfo)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"date":  fmt.Sprintf("%s-%s-%s", year, month, day),
			"files": files,
			"count": len(files),
		})
	}))

	http.HandleFunc("/list-years", basicAuth(func(w http.ResponseWriter, r *http.Request) {
		bucketName := os.Getenv("BUCKET_NAME")
		if bucketName == "" {
			http.Error(w, "BUCKET_NAME environment variable is not set", http.StatusInternalServerError)
			return
		}

		result, err := s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket:    aws.String(bucketName),
			Delimiter: aws.String("/"),
		})

		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to list objects: %v", err), http.StatusInternalServerError)
			return
		}

		yearMap := make(map[string]bool)
		for _, prefix := range result.CommonPrefixes {
			year := strings.TrimSuffix(*prefix.Prefix, "/")
			yearMap[year] = true
		}

		var years []string
		for year := range yearMap {
			years = append(years, year)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"years": years,
		})
	}))

	http.HandleFunc("/list-months", basicAuth(func(w http.ResponseWriter, r *http.Request) {
		bucketName := os.Getenv("BUCKET_NAME")
		if bucketName == "" {
			http.Error(w, "BUCKET_NAME environment variable is not set", http.StatusInternalServerError)
			return
		}

		year := r.URL.Query().Get("year")
		if year == "" {
			http.Error(w, "year query parameter is required", http.StatusBadRequest)
			return
		}

		prefix := year + "/"
		result, err := s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket:    aws.String(bucketName),
			Prefix:    aws.String(prefix),
			Delimiter: aws.String("/"),
		})

		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to list objects: %v", err), http.StatusInternalServerError)
			return
		}

		monthMap := make(map[string]bool)
		for _, prefix := range result.CommonPrefixes {
			path := strings.TrimPrefix(*prefix.Prefix, year+"/")
			month := strings.TrimSuffix(path, "/")
			monthMap[month] = true
		}

		var months []string
		for month := range monthMap {
			months = append(months, month)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"year":   year,
			"months": months,
		})
	}))

	http.HandleFunc("/list-days", basicAuth(func(w http.ResponseWriter, r *http.Request) {
		bucketName := os.Getenv("BUCKET_NAME")
		if bucketName == "" {
			http.Error(w, "BUCKET_NAME environment variable is not set", http.StatusInternalServerError)
			return
		}

		year := r.URL.Query().Get("year")
		month := r.URL.Query().Get("month")
		if year == "" || month == "" {
			http.Error(w, "year and month query parameters are required", http.StatusBadRequest)
			return
		}

		prefix := fmt.Sprintf("%s/%s/", year, month)
		result, err := s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket:    aws.String(bucketName),
			Prefix:    aws.String(prefix),
			Delimiter: aws.String("/"),
		})

		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to list objects: %v", err), http.StatusInternalServerError)
			return
		}

		dayMap := make(map[string]bool)
		for _, prefix := range result.CommonPrefixes {
			path := strings.TrimPrefix(*prefix.Prefix, fmt.Sprintf("%s/%s/", year, month))
			day := strings.TrimSuffix(path, "/")
			dayMap[day] = true
		}

		var days []string
		for day := range dayMap {
			days = append(days, day)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"year":  year,
			"month": month,
			"days":  days,
		})
	}))

	http.HandleFunc("/get-video-url", basicAuth(func(w http.ResponseWriter, r *http.Request) {
		bucketName := os.Getenv("BUCKET_NAME")
		if bucketName == "" {
			http.Error(w, "BUCKET_NAME environment variable is not set", http.StatusInternalServerError)
			return
		}

		key := r.URL.Query().Get("key")
		if key == "" {
			http.Error(w, "key query parameter is required", http.StatusBadRequest)
			return
		}

		// Create a presigned URL for the video
		presignClient := s3.NewPresignClient(s3Client)
		request, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(key),
		}, func(opts *s3.PresignOptions) {
			opts.Expires = time.Duration(3600 * time.Second) // 1 hour expiration
		})

		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to create presigned URL: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"url": request.URL,
		})
	}))

	http.HandleFunc("/latest-video", basicAuth(func(w http.ResponseWriter, r *http.Request) {
		bucketName := os.Getenv("BUCKET_NAME")
		if bucketName == "" {
			http.Error(w, "BUCKET_NAME environment variable is not set", http.StatusInternalServerError)
			return
		}

		// Get current date in YYYY/MM/DD format
		now := time.Now()
		year := fmt.Sprintf("%04d", now.Year())
		month := fmt.Sprintf("%02d", now.Month())
		day := fmt.Sprintf("%02d", now.Day())
		prefix := fmt.Sprintf("%s/%s/%s/", year, month, day)

		result, err := s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket: aws.String(bucketName),
			Prefix: aws.String(prefix),
		})

		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to list objects: %v", err), http.StatusInternalServerError)
			return
		}

		var latestVideo map[string]interface{}
		var latestKey string

		// Find the latest video by filename (lexicographically last)
		for _, obj := range result.Contents {
			if strings.HasSuffix(*obj.Key, ".mp4") {
				if latestKey == "" || *obj.Key > latestKey {
					latestKey = *obj.Key
					latestVideo = map[string]interface{}{
						"key":          *obj.Key,
						"filename":     filepath.Base(*obj.Key),
						"size":         *obj.Size,
						"lastModified": obj.LastModified,
						"storageClass": string(obj.StorageClass),
					}
				}
			}
		}

		w.Header().Set("Content-Type", "application/json")
		if latestVideo != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"date":        fmt.Sprintf("%s-%s-%s", year, month, day),
				"latestVideo": latestVideo,
				"found":       true,
			})
		} else {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"date":  fmt.Sprintf("%s-%s-%s", year, month, day),
				"found": false,
			})
		}
	}))

	http.HandleFunc("/stats", basicAuth(func(w http.ResponseWriter, r *http.Request) {
		bucketName := os.Getenv("BUCKET_NAME")
		if bucketName == "" {
			http.Error(w, "BUCKET_NAME environment variable is not set", http.StatusInternalServerError)
			return
		}

		// Get date range from query parameters
		startDate := r.URL.Query().Get("start_date")
		endDate := r.URL.Query().Get("end_date")
		
		// Default to last 30 days if no range specified
		if startDate == "" || endDate == "" {
			now := time.Now()
			endDate = now.Format("2006-01-02")
			startDate = now.AddDate(0, 0, -30).Format("2006-01-02")
		}

		// Parse dates
		startTime, err := time.Parse("2006-01-02", startDate)
		if err != nil {
			http.Error(w, "Invalid start_date format (use YYYY-MM-DD)", http.StatusBadRequest)
			return
		}
		
		endTime, err := time.Parse("2006-01-02", endDate)
		if err != nil {
			http.Error(w, "Invalid end_date format (use YYYY-MM-DD)", http.StatusBadRequest)
			return
		}

		// Collect statistics
		dailyStats := make(map[string]map[string]interface{})
		totalVideos := 0
		var totalSize int64 = 0
		storageClassCounts := make(map[string]int)
		
		// Iterate through each day in the range
		for d := startTime; !d.After(endTime); d = d.AddDate(0, 0, 1) {
			dateStr := d.Format("2006-01-02")
			year := fmt.Sprintf("%04d", d.Year())
			month := fmt.Sprintf("%02d", d.Month())
			day := fmt.Sprintf("%02d", d.Day())
			prefix := fmt.Sprintf("%s/%s/%s/", year, month, day)

			result, err := s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
				Bucket: aws.String(bucketName),
				Prefix: aws.String(prefix),
			})

			if err != nil {
				// Log error but continue processing other dates
				log.Printf("Error listing objects for %s: %v", dateStr, err)
				continue
			}

			dayVideos := 0
			var daySize int64 = 0
			dayStorageClasses := make(map[string]int)

			for _, obj := range result.Contents {
				if strings.HasSuffix(*obj.Key, ".mp4") {
					dayVideos++
					totalVideos++
					daySize += *obj.Size
					totalSize += *obj.Size
					
					storageClass := string(obj.StorageClass)
					if storageClass == "" {
						storageClass = "STANDARD"
					}
					dayStorageClasses[storageClass]++
					storageClassCounts[storageClass]++
				}
			}

			// Only include days with videos
			if dayVideos > 0 {
				dailyStats[dateStr] = map[string]interface{}{
					"videos": dayVideos,
					"size_bytes": daySize,
					"size_mb": float64(daySize) / (1024 * 1024),
					"storage_classes": dayStorageClasses,
				}
			}
		}

		// Calculate summary statistics
		daysWithVideos := len(dailyStats)
		avgVideosPerDay := 0.0
		if daysWithVideos > 0 {
			avgVideosPerDay = float64(totalVideos) / float64(daysWithVideos)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"period": map[string]string{
				"start_date": startDate,
				"end_date": endDate,
			},
			"summary": map[string]interface{}{
				"total_videos": totalVideos,
				"total_size_bytes": totalSize,
				"total_size_mb": float64(totalSize) / (1024 * 1024),
				"days_with_videos": daysWithVideos,
				"avg_videos_per_day": avgVideosPerDay,
				"storage_class_distribution": storageClassCounts,
			},
			"daily_stats": dailyStats,
		})
	}))

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "healthy",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"service": "camera-viewer",
		})
	})

	http.HandleFunc("/", basicAuth(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	}))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Server starting on port %s...\n", port)
	
	username := os.Getenv("USERNAME")
	password := os.Getenv("PASSWORD")
	if username != "" && password != "" {
		fmt.Println("Authentication enabled - USERNAME and PASSWORD required")
	} else {
		fmt.Println("Warning: No authentication configured (set USERNAME and PASSWORD env vars)")
	}
	
	fmt.Println("Available endpoints:")
	fmt.Printf("  - http://localhost:%s/ (Web UI)\n", port)
	fmt.Printf("  - http://localhost:%s/list-bucket\n", port)
	fmt.Printf("  - http://localhost:%s/list-years\n", port)
	fmt.Printf("  - http://localhost:%s/list-months?year=2024\n", port)
	fmt.Printf("  - http://localhost:%s/list-days?year=2024&month=01\n", port)
	fmt.Printf("  - http://localhost:%s/list-files-by-date?year=2024&month=01&day=15\n", port)
	fmt.Printf("  - http://localhost:%s/stats (with optional start_date and end_date params)\n", port)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}