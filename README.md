# Camera Viewer

A web-based application for viewing camera footage that's automatically recorded and uploaded to Amazon S3. This application provides an easy way to browse and watch your security camera recordings organized by date.

## Overview

Camera Viewer is designed for scenarios where security cameras automatically record footage and upload it to S3 storage in a structured folder format (`YYYY/MM/DD/*.mp4`). The application provides:

- **Date-based navigation** - Browse recordings by year, month, and day
- **Latest video display** - Automatically shows the most recent video on the main page
- **Storage class awareness** - Displays S3 storage class (Standard, Glacier, Deep Archive) for each video
- **Direct video playback** - Stream videos directly in the browser for Standard storage files
- **Basic authentication** - Secure access with username/password protection
- **Responsive design** - Works on desktop and mobile devices

## Features

- 📅 **Hierarchical browsing** - Navigate through years → months → days
- 🎥 **Video streaming** - Watch videos directly in the browser (Standard storage only)
- 🔄 **Auto-refresh** - Latest video updates every 5 minutes
- 🏷️ **Storage indicators** - Visual badges showing S3 storage class
- 🔒 **Authentication** - Basic HTTP auth protection
- 📱 **Mobile friendly** - Responsive web interface
- 🐳 **Containerized** - Docker and Docker Compose ready

## Quick Start

### Prerequisites

- Docker and Docker Compose
- AWS S3 bucket with video files organized as `YYYY/MM/DD/*.mp4`
- AWS credentials with S3 read access

### Setup

1. **Clone and setup environment:**

   ```bash
   git clone <your-repo>
   cd camera-viewer
   make env  # Creates .env from .env.example
   ```

2. **Configure environment variables:**
   Edit `.env` file with your settings:

   ```bash
   # AWS Configuration
   AWS_REGION=us-east-1
   AWS_ACCESS_KEY_ID=your-access-key-id
   AWS_SECRET_ACCESS_KEY=your-secret-access-key
   BUCKET_NAME=your-s3-bucket-name

   # Authentication
   USERNAME=admin
   PASSWORD=your-secure-password
   ```

3. **Start the application:**

   ```bash
   make dev  # Builds, runs, and shows logs
   ```

4. **Access the application:**
   ```bash
   make open  # Opens http://localhost:8080 in browser
   ```

## Make Commands

The Makefile provides convenient commands for common operations:

### Quick Commands

- `make help` - Show all available commands
- `make dev` - Full development setup (env + build + run + logs)
- `make env` - Create .env file from .env.example

### Build Commands

- `make build` - Build the Docker image
- `make build-no-cache` - Build without using cache
- `make test-build` - Test build without saving image

### Run Commands

- `make up` - Start services with docker-compose
- `make up-build` - Start services and rebuild images
- `make up-logs` - Start services and show logs
- `make down` - Stop and remove containers
- `make restart` - Restart services (down + up)

### Development Commands

- `make logs` - Show container logs
- `make status` - Show container status
- `make shell` - Get shell access to container
- `make health` - Check container health status

### Utility Commands

- `make check-env` - Verify environment variables are set
- `make clean` - Clean up containers and images
- `make clean-all` - Clean everything (containers, images, volumes)
- `make open` - Open application in browser

### Production Commands

- `make deploy` - Build and deploy for production
- `make update` - Update deployment (down, build, up)

## File Structure

```
camera-viewer/
├── main.go              # Go application source
├── index.html           # Web interface
├── go.mod              # Go module definition
├── go.sum              # Go dependency checksums
├── Dockerfile          # Docker image definition
├── docker-compose.yml  # Docker Compose configuration
├── Makefile           # Build automation
├── .env.example       # Environment template
├── .env               # Your environment (create from .env.example)
├── .dockerignore      # Docker build exclusions
└── README.md          # This file
```

## S3 Bucket Structure

The application expects your S3 bucket to contain video files organized in this structure:

```
your-bucket/
├── 2024/
│   ├── 01/
│   │   ├── 01/
│   │   │   ├── camera_20240101_120000.mp4
│   │   │   ├── camera_20240101_130000.mp4
│   │   │   └── ...
│   │   ├── 02/
│   │   └── ...
│   ├── 02/
│   └── ...
└── 2025/
    └── ...
```

## Storage Classes

The application handles different S3 storage classes:

- **🟢 Standard** - Videos are immediately playable
- **🔵 Glacier** - Videos need restoration before viewing
- **⚫ Deep Archive** - Videos need restoration (longer process)

Videos in Glacier or Deep Archive storage will show an informational message instead of a video player.

## Authentication

Basic HTTP authentication protects all endpoints. Configure credentials in your `.env` file:

```bash
USERNAME=your_username
PASSWORD=your_secure_password
```

If no credentials are set, the application will run without authentication (not recommended for production).

## Development

### Local Development (without Docker)

1. **Install Go 1.21+**
2. **Set environment variables:**
   ```bash
   export AWS_REGION=us-east-1
   export AWS_ACCESS_KEY_ID=your-key
   export AWS_SECRET_ACCESS_KEY=your-secret
   export BUCKET_NAME=your-bucket
   export USERNAME=admin
   export PASSWORD=password
   ```
3. **Run the application:**
   ```bash
   go mod download
   go run main.go
   ```

### Building for Production

```bash
# Build optimized image
make build

# Deploy to production
make deploy
```

## Troubleshooting

### Common Issues

1. **"BUCKET_NAME environment variable is not set"**

   - Ensure your `.env` file is properly configured
   - Run `make check-env` to verify all variables

2. **"Authentication failed"**

   - Check USERNAME and PASSWORD in `.env`
   - Clear browser cache/credentials

3. **"No videos found"**

   - Verify S3 bucket structure matches `YYYY/MM/DD/*.mp4`
   - Check AWS credentials have S3 read permissions
   - Ensure bucket name is correct

4. **Videos won't play**
   - Check if videos are in Glacier/Deep Archive storage
   - Verify AWS credentials allow GetObject operations

### Logs and Debugging

```bash
# View application logs
make logs

# Check container status
make status

# Access container shell for debugging
make shell

# Verify environment variables
make check-env
```

## Security Considerations

- Always use HTTPS in production
- Use strong passwords for authentication
- Consider IAM roles instead of access keys in AWS environments
- Regularly rotate AWS credentials
- Keep the Docker image updated

## License

[Add your license here]

## Contributing

[Add contribution guidelines here]
