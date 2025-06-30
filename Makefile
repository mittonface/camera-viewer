# Variables
APP_NAME := camera-viewer
IMAGE_NAME := $(APP_NAME)
CONTAINER_NAME := $(APP_NAME)
PORT := 8080

# Default target
.PHONY: help
help: ## Show this help message
	@echo "Camera Viewer Docker Commands:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Environment setup
.PHONY: env
env: ## Create .env file from .env.example
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo "Created .env file from .env.example"; \
		echo "Please edit .env with your actual values"; \
	else \
		echo ".env file already exists"; \
	fi

# Docker build commands
.PHONY: build
build: ## Build the Docker image
	docker build -t $(IMAGE_NAME) .

.PHONY: build-no-cache
build-no-cache: ## Build the Docker image without cache
	docker build --no-cache -t $(IMAGE_NAME) .

# Docker run commands
.PHONY: run
run: ## Run the container with docker run
	docker run -d \
		--name $(CONTAINER_NAME) \
		-p $(PORT):$(PORT) \
		--env-file .env \
		$(IMAGE_NAME)

.PHONY: run-interactive
run-interactive: ## Run the container interactively
	docker run -it \
		--name $(CONTAINER_NAME)-interactive \
		-p $(PORT):$(PORT) \
		--env-file .env \
		$(IMAGE_NAME)

# Docker Compose commands
.PHONY: up
up: ## Start services with docker-compose
	docker-compose up -d

.PHONY: up-build
up-build: ## Start services and rebuild images
	docker-compose up -d --build

.PHONY: up-logs
up-logs: ## Start services and show logs
	docker-compose up

.PHONY: down
down: ## Stop and remove containers
	docker-compose down

.PHONY: restart
restart: down up ## Restart services

# Container management
.PHONY: stop
stop: ## Stop the running container
	docker stop $(CONTAINER_NAME) || true

.PHONY: start
start: ## Start the stopped container
	docker start $(CONTAINER_NAME)

.PHONY: remove
remove: stop ## Remove the container
	docker rm $(CONTAINER_NAME) || true

# Logs and monitoring
.PHONY: logs
logs: ## Show container logs
	docker-compose logs -f

.PHONY: logs-app
logs-app: ## Show application logs only
	docker-compose logs -f $(APP_NAME)

.PHONY: status
status: ## Show container status
	docker-compose ps

.PHONY: health
health: ## Check container health
	docker inspect --format='{{.State.Health.Status}}' $(CONTAINER_NAME) || echo "Container not running or no health check"

# Shell access
.PHONY: shell
shell: ## Get shell access to running container
	docker-compose exec $(APP_NAME) sh

.PHONY: shell-root
shell-root: ## Get root shell access to running container
	docker-compose exec --user root $(APP_NAME) sh

# Development commands
.PHONY: dev
dev: env up-build logs ## Full development setup (env + build + run + logs)

.PHONY: clean
clean: ## Clean up containers and images
	docker-compose down -v
	docker rm $(CONTAINER_NAME) || true
	docker rmi $(IMAGE_NAME) || true

.PHONY: clean-all
clean-all: ## Clean up everything (containers, images, volumes)
	docker-compose down -v --rmi all
	docker system prune -f

# Production commands
.PHONY: deploy
deploy: build up ## Build and deploy for production

.PHONY: update
update: down build up ## Update deployment (down, build, up)

.PHONY: deploy-server
deploy-server: ## Deploy to server (run this on server)
	./deploy.sh

# Utility commands
.PHONY: check-env
check-env: ## Check if required environment variables are set
	@echo "Checking environment variables..."
	@if [ ! -f .env ]; then echo "ERROR: .env file not found. Run 'make env' first."; exit 1; fi
	@grep -q "BUCKET_NAME=" .env || { echo "ERROR: BUCKET_NAME not set in .env"; exit 1; }
	@grep -q "AWS_ACCESS_KEY_ID=" .env || { echo "ERROR: AWS_ACCESS_KEY_ID not set in .env"; exit 1; }
	@grep -q "USERNAME=" .env || { echo "ERROR: USERNAME not set in .env"; exit 1; }
	@grep -q "PASSWORD=" .env || { echo "ERROR: PASSWORD not set in .env"; exit 1; }
	@echo "✓ All required environment variables are set"

.PHONY: test-build
test-build: ## Test if the application builds successfully
	docker build -t $(IMAGE_NAME)-test .
	docker rmi $(IMAGE_NAME)-test

.PHONY: open
open: ## Open the application in browser
	@echo "Opening http://localhost:$(PORT)"
	@which open >/dev/null && open http://localhost:$(PORT) || \
	 which xdg-open >/dev/null && xdg-open http://localhost:$(PORT) || \
	 echo "Please open http://localhost:$(PORT) in your browser"