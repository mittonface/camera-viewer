# Deployment Guide

This guide explains how to set up automatic deployments from GitHub to your server using GitHub Actions.

## Overview

The deployment system uses GitHub Actions to:

1. Build a Docker image when you push to the main branch
2. Transfer the image and configuration files to your server via SSH
3. Deploy the application using Docker Compose
4. Perform health checks to ensure successful deployment

## Setup Instructions

### 1. Server Preparation

**Install Docker and Docker Compose on your server:**

```bash
# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
sudo usermod -aG docker $USER

# Install Docker Compose
sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose

# Log out and back in for group changes to take effect
```

**Create the application directory:**

```bash
mkdir -p ~/camera-viewer
cd ~/camera-viewer
```

**Create the .env file:**

```bash
cp .env.example .env
# Edit .env with your actual values
nano .env
```

### 2. SSH Key Setup

**Generate SSH key pair (if you don't have one):**

```bash
ssh-keygen -t ed25519 -C "github-actions"
```

**Add the public key to your server:**

```bash
# On your server
cat ~/.ssh/id_ed25519.pub >> ~/.ssh/authorized_keys
chmod 600 ~/.ssh/authorized_keys
```

**Copy the private key for GitHub secrets:**

```bash
cat ~/.ssh/id_ed25519
# Copy this entire output for GitHub secrets
```

### 3. GitHub Repository Secrets

Go to your GitHub repository → Settings → Secrets and variables → Actions

Add these repository secrets:

| Secret Name      | Description                          | Example                                  |
| ---------------- | ------------------------------------ | ---------------------------------------- |
| `SERVER_HOST`    | Your server's IP address or hostname | `192.168.1.100`                          |
| `SERVER_USER`    | Username on your server              | `ubuntu`                                 |
| `SERVER_SSH_KEY` | Private SSH key (entire content)     | `-----BEGIN OPENSSH PRIVATE KEY-----...` |

### 4. GitHub Actions Workflow

The workflow file `.github/workflows/deploy.yml` is already configured and will:

- Trigger on pushes to the `main` branch
- Build the Docker image
- Transfer files to your server
- Deploy the application
- Perform health checks

### 5. First Deployment

**Option A: Automatic (Recommended)**

1. Push your code to the main branch
2. GitHub Actions will automatically deploy
3. Monitor the deployment in the Actions tab

**Option B: Manual**

1. Copy files to your server manually
2. Run the deployment script:
   ```bash
   # On your server
   cd ~/camera-viewer
   ./deploy.sh
   ```

## Deployment Process

### What happens during deployment:

1. **Build Phase** (GitHub Actions):

   - Checkout code
   - Build Docker image
   - Save image as compressed file

2. **Transfer Phase**:

   - Copy image, docker-compose.yml, and Makefile to server
   - Use SCP over SSH

3. **Deploy Phase** (On Server):
   - Load new Docker image
   - Stop existing containers
   - Start new containers
   - Clean up old images
   - Perform health check

### Deployment Commands

**On your local machine:**

```bash
# Trigger deployment by pushing to main
git push origin main

# Manual trigger via GitHub web interface
# Go to Actions tab → Deploy to Server → Run workflow
```

**On your server:**

```bash
# Manual deployment
make deploy-server

# Check status
make status

# View logs
make logs

# Restart if needed
make restart
```

## Monitoring and Troubleshooting

### Check Deployment Status

**GitHub Actions:**

- Go to your repository's Actions tab
- Click on the latest workflow run
- Check each step for errors

**On Server:**

```bash
# Check if containers are running
docker-compose ps

# View application logs
docker-compose logs -f camera-viewer

# Check system resources
docker stats

# Health check
curl http://localhost:8080/list-years
```

### Common Issues

**Deployment fails with SSH errors:**

- Verify SERVER_HOST, SERVER_USER, and SERVER_SSH_KEY secrets
- Test SSH connection manually: `ssh user@server`
- Check server's SSH configuration

**Docker build fails:**

- Check Dockerfile syntax
- Verify all required files are present
- Check GitHub Actions logs for specific errors

**Application won't start:**

- Verify .env file exists on server with correct values
- Check container logs: `docker-compose logs`
- Ensure ports aren't already in use

**Health check fails:**

- Wait longer for application to start
- Check if authentication is blocking the health check
- Verify network connectivity

### Rollback

If a deployment fails, you can rollback:

```bash
# On server
cd ~/camera-viewer

# Stop current deployment
docker-compose down

# List available images
docker images camera-viewer

# Use a previous image
docker tag camera-viewer:previous camera-viewer:latest
docker-compose up -d
```

## Security Considerations

- Use strong passwords in .env file
- Regularly rotate SSH keys
- Consider using a VPN for server access
- Enable firewall rules to restrict access
- Keep Docker and system packages updated
- Use HTTPS in production (add reverse proxy)

## Customization

### Modify Deployment Branch

Edit `.github/workflows/deploy.yml`:

```yaml
on:
  push:
    branches: [your-branch-name]
```

### Add Environment-Specific Deployments

Create separate workflow files for different environments:

- `.github/workflows/deploy-staging.yml`
- `.github/workflows/deploy-production.yml`

### Use Docker Registry

Uncomment the Docker Hub login section in the workflow and push images to a registry instead of transferring files.

## Alternative: Self-Hosted Runner

For better performance and security, consider using a self-hosted GitHub Actions runner on your server:

1. Go to repository Settings → Actions → Runners
2. Click "New self-hosted runner"
3. Follow the setup instructions for your server
4. Modify the workflow to use: `runs-on: self-hosted`

This eliminates the need for SSH and file transfers.
