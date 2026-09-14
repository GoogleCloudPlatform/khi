# build.mk
# This file contains make tasks for building.


WEB_HOST ?= localhost
WEB_PORT ?= 4200
WEB_ALLOWED_HOSTS_FLAG ?= $(if $(filter 0.0.0.0,$(WEB_HOST)),--allowed-hosts,)
STORYBOOK_HOST ?= localhost
STORYBOOK_PORT ?= 6006
KARMA_PORT ?= 9876
BACKEND_PORT ?= $(or $(PORT),8080)
BACKEND_HOST ?= $(if $(filter 0.0.0.0,$(HOST)),127.0.0.1,$(or $(HOST),127.0.0.1))

.PHONY: watch-web
watch-web: $(GENERATE_FRONTEND_DUMMY) ## Run frontend development server
	cd web && BACKEND_PORT=$(BACKEND_PORT) BACKEND_HOST=$(BACKEND_HOST) npx ng serve -c dev --host $(WEB_HOST) --port $(WEB_PORT) $(WEB_ALLOWED_HOSTS_FLAG)

$(FRONTEND_ARTIFACT_FILES_DUMMY): $(GENERATE_FRONTEND_DUMMY) $(FRONTEND_SOURCE_FILES) $(FRONTEND_GENERATED_SRCS)## Build frontend for production
	cd web && npx ng build --output-path ../pkg/server/dist -c prod
	touch $(FRONTEND_ARTIFACT_FILES_DUMMY)

.PHONY: build-web
build-web: $(FRONTEND_ARTIFACT_FILES_DUMMY) ## Build frontend for production

.PHONY: watch-storybook
watch-storybook: $(GENERATE_FRONTEND_DUMMY) ## Run storybook development server
	cd web && npm run storybook -- --host $(STORYBOOK_HOST) --port $(STORYBOOK_PORT)

.PHONY: build-storybook
build-storybook: $(GENERATE_FRONTEND_DUMMY) ## Build storybook
	cd web && npm run build-storybook

.PHONY: watch-karma
watch-karma: $(GENERATE_FRONTEND_DUMMY) ## Run karma test server
	cd web && KARMA_PORT=$(KARMA_PORT) npm run test

khi: $(GENERATE_BACKEND_DUMMY) $(FRONTEND_ARTIFACT_FILES_DUMMY) $(BACKEND_SRCS)
	CGO_ENABLED=0 go build -ldflags="-s -w -X github.com/GoogleCloudPlatform/khi/pkg/common/constants.VERSION=$(shell cat ./VERSION)" -o ./khi ./cmd/kubernetes-history-inspector/...

.PHONY: build-go
build-go: khi ## Build backend for production

khi-debug: $(GENERATE_BACKEND_DUMMY) $(FRONTEND_ARTIFACT_FILES_DUMMY) $(BACKEND_SRCS)
	CGO_ENABLED=0 go build -gcflags="all=-N -l" -ldflags="-X github.com/GoogleCloudPlatform/khi/pkg/common/constants.VERSION=$(shell cat ./VERSION)" -o ./khi-debug ./cmd/kubernetes-history-inspector/...	

.PHONY: build-go-debug
build-go-debug: khi-debug ## Build backend for debugging

.PHONY: build
build: build-go

define build_binary
	CGO_ENABLED=0 GOOS=$(1) GOARCH=$(2) go build -ldflags="-s -w -X github.com/GoogleCloudPlatform/khi/pkg/common/constants.VERSION=$(shell cat ./VERSION)" -o ./bin/khi-$(1)-$(2)$(3) ./cmd/kubernetes-history-inspector/...
endef

.PHONY: build-go-binaries
build-go-binaries: $(GENERATE_BACKEND_DUMMY) $(BACKEND_SRCS) $(FRONTEND_ARTIFACT_FILES_DUMMY) ## Build go binaries for multiple platforms
	mkdir -p bin
	$(call build_binary,windows,amd64,.exe)
	$(call build_binary,linux,amd64,)
	$(call build_binary,darwin,arm64,)
	$(call build_binary,darwin,amd64,)

BUILDER_IMAGE ?= gcr.io/kubernetes-history-inspector/builder:latest

.PHONY: build-builder
build-builder: ## Build CI builder docker image
	docker build -t $(BUILDER_IMAGE) ./scripts/builder

.PHONY: push-builder
push-builder: build-builder ## Push CI builder docker image
	docker push $(BUILDER_IMAGE)
