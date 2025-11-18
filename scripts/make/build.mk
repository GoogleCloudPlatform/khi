# build.mk
# This file contains make tasks for building.

.PHONY: watch-web
watch-web: prepare-frontend ## Run frontend development server
	cd web && npx ng serve -c dev

.PHONY: build-web
build-web: prepare-frontend ## Build frontend for production
	cd web && npx ng build --output-path ../pkg/server/dist -c prod

.PHONY: build-go
build-go: generate-backend ## Build backend for production
	CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w -X github.com/GoogleCloudPlatform/khi/pkg/common/constants.VERSION=$(shell cat ./VERSION)" -o ./khi ./cmd/kubernetes-history-inspector/...

.PHONY: build-go-debug
build-go-debug: generate-backend ## Build backend for debugging
	CGO_ENABLED=0 go build -gcflags="all=-N -l" -ldflags="-X github.com/GoogleCloudPlatform/khi/pkg/common/constants.VERSION=$(shell cat ./VERSION)" -o ./khi-debug ./cmd/kubernetes-history-inspector/...

.PHONY: build
build: build-go build-web ## Build all source code

.PHONY: build-go-binaries
build-go-binaries: build-web generate-backend
	mkdir -p bin
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-s -w -X github.com/GoogleCloudPlatform/khi/pkg/common/constants.VERSION=$(shell cat ./VERSION)" -o ./bin/khi-$(shell cat ./VERSION)-amd64-windows.exe ./cmd/kubernetes-history-inspector/...
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X github.com/GoogleCloudPlatform/khi/pkg/common/constants.VERSION=$(shell cat ./VERSION)" -o ./bin/khi-$(shell cat ./VERSION)-amd64-linux ./cmd/kubernetes-history-inspector/...
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w -X github.com/GoogleCloudPlatform/khi/pkg/common/constants.VERSION=$(shell cat ./VERSION)" -o ./bin/khi-$(shell cat ./VERSION)-arm64-darwin ./cmd/kubernetes-history-inspector/...
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w -X github.com/GoogleCloudPlatform/khi/pkg/common/constants.VERSION=$(shell cat ./VERSION)" -o ./bin/khi-$(shell cat ./VERSION)-amd64-darwin ./cmd/kubernetes-history-inspector/...
