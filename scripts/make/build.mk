# build.mk
# This file contains make tasks for building.


.PHONY: watch-web
watch-web: $(GENERATE_FRONTEND_DUMMY) ## Run frontend development server
	cd web && npx ng serve -c dev

$(FRONTEND_ARTIFACT_FILES_DUMMY): $(GENERATE_FRONTEND_DUMMY) $(FRONTEND_SOURCE_FILES) $(FRONTEND_GENERATED_SRCS)## Build frontend for production
	cd web && npx ng build --output-path ../pkg/server/dist -c prod
	touch $(FRONTEND_ARTIFACT_FILES_DUMMY)

.PHONY: build-web
build-web: $(FRONTEND_ARTIFACT_FILES_DUMMY) ## Build frontend for production

.PHONY: watch-storybook
watch-storybook: $(GENERATE_FRONTEND_DUMMY) ## Run storybook development server
	cd web && npm run storybook

.PHONY: watch-karma
watch-karma: $(GENERATE_FRONTEND_DUMMY) ## Run karma test server
	cd web && npm run test

khi: $(GENERATE_BACKEND_DUMMY) $(FRONTEND_ARTIFACT_FILES_DUMMY) $(BACKEND_SRCS)
	CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w -X github.com/GoogleCloudPlatform/khi/pkg/common/constants.VERSION=$(shell cat ./VERSION)" -o ./khi ./cmd/kubernetes-history-inspector/...

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
