# build.mk
# This file contains make tasks for building.

.PHONY: watch-web
watch-web: prepare-frontend ## Run frontend development server
	cd web && npx ng serve -c dev

.PHONY: build-web
build-web: prepare-frontend ## Build frontend for production
	cd web && npx ng build --output-path ../dist -c prod

.PHONY: build-go
build-go: generate-backend ## Build backend for production
	CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w -X github.com/GoogleCloudPlatform/khi/pkg/common/constants.VERSION=$(shell cat ./VERSION)" -o ./khi ./cmd/kubernetes-history-inspector/...

.PHONY: build-go-debug
build-go-debug: generate-backend ## Build backend for debugging
	CGO_ENABLED=0 go build -gcflags="all=-N -l" -ldflags="-X github.com/GoogleCloudPlatform/khi/pkg/common/constants.VERSION=$(shell cat ./VERSION)" -o ./khi-debug ./cmd/kubernetes-history-inspector/...

.PHONY: build
build: build-go build-web ## Build all source code
