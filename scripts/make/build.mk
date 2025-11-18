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

define build_binary
	CGO_ENABLED=0 GOOS=$(1) GOARCH=$(2) go build -ldflags="-s -w -X github.com/GoogleCloudPlatform/khi/pkg/common/constants.VERSION=$(VERSION)" -o ./bin/khi-$(VERSION)-$(2)-$(1)$(3) ./cmd/kubernetes-history-inspector/...
endef

.PHONY: build-go-binaries
build-go-binaries: build-web generate-backend ## Build go binaries for multiple platforms
	mkdir -p bin
	$(call build_binary,windows,amd64,.exe)
	$(call build_binary,linux,amd64,)
	$(call build_binary,darwin,arm64,)
	$(call build_binary,darwin,amd64,)
