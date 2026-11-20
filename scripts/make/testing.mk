# testing.mk
# This file contains make tasks related to testing.

KARMA_PORT ?= 9876
# Set USE_CONTAINER=true or USE_CONTAINER=1 to run frontend tests inside the container.
USE_CONTAINER ?= false

.PHONY: test-web
ifneq ($(filter true 1,$(USE_CONTAINER)),)
test-web: test-web-container
else
test-web: $(GENERATE_FRONTEND_DUMMY) $(FRONTEND_SOURCE_FILES)## Run frontend tests
	cd web && npm run test:scripts
	cd web && KARMA_PORT=$(KARMA_PORT) npx ng test --browsers ChromeHeadlessNoSandbox --watch=false
endif

.PHONY: test-web-container
test-web-container: build-builder ## Run frontend tests inside container
	@if [ -z "$(CONTAINER_CMD)" ]; then \
		echo "test-web-container requires docker or podman, but neither was found." >&2; exit 1; \
	fi
	$(CONTAINER_CMD) run --rm \
		--shm-size=1g \
		-u $(shell id -u):$(shell id -g) \
		-v $(CURDIR):/workspace \
		-w /workspace \
		-e HOME=/tmp \
		-e KARMA_PORT=$(KARMA_PORT) \
		$(BUILDER_IMAGE) \
		make test-web USE_CONTAINER=false

.PHONY: watch-test-web
watch-test-web: $(GENERATE_FRONTEND_DUMMY) ## Run frontend tests in watch mode
	cd web && KARMA_PORT=$(KARMA_PORT) npx ng test

.PHONY: test-go
test-go: $(GENERATE_BACKEND_DUMMY) $(BACKEND_TEST_SRCS) $(FRONTEND_ARTIFACT_FILES_DUMMY) ## Run backend tests
	go test ./...

.PHONY: test-go-short
test-go-short: $(GENERATE_BACKEND_DUMMY) $(BACKEND_TEST_SRCS) $(FRONTEND_ARTIFACT_FILES_DUMMY) ## Run backend tests in short mode (scaled-down stress iterations)
	go test -short ./...

.PHONY: coverage-web
ifneq ($(filter true 1,$(USE_CONTAINER)),)
coverage-web: coverage-web-container
else
coverage-web: $(GENERATE_FRONTEND_DUMMY) $(FRONTEND_SOURCE_FILES)## Run frontend tests and generate coverage report
	cd web && KARMA_PORT=$(KARMA_PORT) npx ng test --code-coverage --browsers ChromeHeadlessNoSandbox --watch false --progress false
endif

.PHONY: coverage-web-container
coverage-web-container: build-builder ## Run frontend tests and generate coverage report inside container
	@if [ -z "$(CONTAINER_CMD)" ]; then \
		echo "coverage-web-container requires docker or podman, but neither was found." >&2; exit 1; \
	fi
	$(CONTAINER_CMD) run --rm \
		--shm-size=1g \
		-u $(shell id -u):$(shell id -g) \
		-v $(CURDIR):/workspace \
		-w /workspace \
		-e HOME=/tmp \
		-e KARMA_PORT=$(KARMA_PORT) \
		$(BUILDER_IMAGE) \
		make coverage-web USE_CONTAINER=false

.PHONY: coverage-go
coverage-go: $(GENERATE_BACKEND_DUMMY) $(BACKEND_TEST_SRCS) $(FRONTEND_ARTIFACT_FILES_DUMMY)## Run backend tests and generate coverage report
	KHI_SKIP_CLOUD_LOGGING=true go test -cover ./... -coverprofile=./go-cover.output
	go tool cover -html=./go-cover.output -o=go-cover.html

.PHONY: codequality-ci
codequality-ci: ## Run unified differential code quality checks for Cloud Build CI
	/bin/bash ./scripts/run-codequality-ci.sh

.PHONY: post-codequality-comment
post-codequality-comment: ## Post or update sticky GitHub PR comment from completed Cloud Build Check Run
	/bin/bash ./scripts/post-codequality-pr-comment.sh

FIXTURES_GCS_BUCKET ?= gs://khi-fixtures

.PHONY: download-fixtures
download-fixtures: ## Download benchmark test fixtures from GCS
	@echo "Downloading fixtures from $(FIXTURES_GCS_BUCKET)..."
	gcloud storage rsync -r $(FIXTURES_GCS_BUCKET) ./pkg --include-glob="**/testdata/fixtures/**.json"

.PHONY: upload-fixtures
upload-fixtures: ## Upload local benchmark test fixtures to GCS
	@echo "Uploading fixtures to $(FIXTURES_GCS_BUCKET)..."
	gcloud storage rsync -r ./pkg $(FIXTURES_GCS_BUCKET) --include-glob="**/testdata/fixtures/**.json"

VRT_FILTER ?=
VRT_WORKERS ?= 0
VRT_OUT_DIR ?= vrt-output/snapshot
VRT_BASE_DIR ?=
VRT_REPORT_DIR ?= vrt-output/report
VRT_STORYBOOK_DIR ?= web/storybook-static

.PHONY: vrt
ifneq ($(filter true 1,$(USE_CONTAINER)),)
vrt: vrt-container
else
vrt: ## Capture Storybook screenshots locally (set VRT_FILTER=<regex> to narrow stories, VRT_BASE_DIR=<dir> to compare)
	@if [ ! -f "$(VRT_STORYBOOK_DIR)/index.json" ]; then \
		$(MAKE) build-storybook; \
	fi
	@if [ -n "$(VRT_BASE_DIR)" ]; then \
		go run ./scripts/vrt --mode=compare --storybook-dir="$(VRT_STORYBOOK_DIR)" --base-dir="$(VRT_BASE_DIR)" --head-dir="$(VRT_OUT_DIR)" --report-dir="$(VRT_REPORT_DIR)" --filter="$(VRT_FILTER)" --workers="$(VRT_WORKERS)"; \
	else \
		go run ./scripts/vrt --mode=capture --storybook-dir="$(VRT_STORYBOOK_DIR)" --out-dir="$(VRT_OUT_DIR)" --filter="$(VRT_FILTER)" --workers="$(VRT_WORKERS)"; \
	fi
endif

.PHONY: vrt-container
vrt-container: build-builder ## Run Storybook visual regression capture/comparison inside the CI builder container
	@if [ -z "$(CONTAINER_CMD)" ]; then \
		echo "vrt-container requires docker or podman, but neither was found." >&2; exit 1; \
	fi
	$(CONTAINER_CMD) run --rm \
		--shm-size=1g \
		-u $(shell id -u):$(shell id -g) \
		-v $(CURDIR):/workspace \
		-v $(shell go env GOMODCACHE):/tmp/gomodcache:ro \
		-w /workspace \
		-e HOME=/tmp \
		-e GOPATH=/tmp/go \
		-e GOMODCACHE=/tmp/gomodcache \
		-e GOCACHE=/tmp/gocache \
		-e VRT_FILTER="$(VRT_FILTER)" \
		-e VRT_WORKERS="$(VRT_WORKERS)" \
		-e VRT_OUT_DIR="$(VRT_OUT_DIR)" \
		-e VRT_BASE_DIR="$(VRT_BASE_DIR)" \
		-e VRT_REPORT_DIR="$(VRT_REPORT_DIR)" \
		-e VRT_STORYBOOK_DIR="$(VRT_STORYBOOK_DIR)" \
		$(BUILDER_IMAGE) \
		make vrt USE_CONTAINER=false

.PHONY: vrt-ci
vrt-ci: ## Run Storybook visual regression test pipeline for Cloud Build CI
	/bin/bash ./scripts/run-vrt-ci.sh

.PHONY: post-vrt-comment
post-vrt-comment: ## Post or update sticky GitHub PR comment from completed VRT Cloud Build Check Run
	/bin/bash ./scripts/post-vrt-pr-comment.sh
