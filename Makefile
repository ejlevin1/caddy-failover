# Makefile for Caddy Failover Plugin

.PHONY: build test clean docker-build docker-push xcaddy-build help

# Variables
IMAGE_NAME ?= caddy-failover
REGISTRY ?= ghcr.io
ORG ?= ejlevin1
VERSION ?= latest
FULL_IMAGE = $(REGISTRY)/$(ORG)/$(IMAGE_NAME):$(VERSION)

help: ## Show this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Build the plugin locally (requires Go)
	go build -v ./...

test: ## Run Go tests
	go test -v -race -cover ./...

xcaddy-build: ## Build Caddy with this plugin using xcaddy
	@which xcaddy > /dev/null || (echo "Installing xcaddy..." && go install github.com/caddyserver/xcaddy/cmd/xcaddy@latest)
	xcaddy build --with github.com/ejlevin1/caddy-failover=.

xcaddy-run: xcaddy-build ## Build and run Caddy with example config
	./caddy run --config examples/basic-caddyfile --adapter caddyfile

docker-build: ## Build Docker image
	docker build -t $(IMAGE_NAME):$(VERSION) .

docker-push: docker-build ## Push Docker image to registry
	docker tag $(IMAGE_NAME):$(VERSION) $(FULL_IMAGE)
	docker push $(FULL_IMAGE)

docker-test: docker-build ## Run integration tests with Docker
	chmod +x test/test.sh
	./test/test.sh

clean: ## Clean build artifacts and test files
	rm -f caddy caddy-* test-caddyfile test.Caddyfile
	docker rmi $(IMAGE_NAME):$(VERSION) 2>/dev/null || true
	docker rmi $(FULL_IMAGE) 2>/dev/null || true
	go clean -cache

run-example: docker-build ## Run example configuration with Docker
	docker run --rm \
		-v $(PWD)/examples/basic-caddyfile:/etc/caddy/Caddyfile \
		-p 80:80 -p 443:443 -p 2019:2019 \
		$(IMAGE_NAME):$(VERSION)

lint: ## Run Go linter (requires golangci-lint)
	@which golangci-lint > /dev/null || (echo "Please install golangci-lint: https://golangci-lint.run/usage/install/" && exit 1)
	golangci-lint run

fmt: ## Format Go code
	go fmt ./...
	gofmt -s -w .

mod-tidy: ## Tidy Go modules
	go mod tidy

verify: fmt mod-tidy ## Verify code formatting and dependencies
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "Working directory is not clean. Please commit your changes."; \
		git status --porcelain; \
		exit 1; \
	fi

release: ## Create a new release (requires VERSION parameter)
	@if [ -z "$(VERSION)" ]; then echo "VERSION is required"; exit 1; fi
	git tag -a v$(VERSION) -m "Release v$(VERSION)"
	git push origin v$(VERSION)
