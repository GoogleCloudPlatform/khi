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
	go test -cover ./... -coverprofile=./go-cover.output
	go tool cover -html=./go-cover.output -o=go-cover.html

FIXTURES_GCS_BUCKET ?= gs://khi-fixtures

.PHONY: download-fixtures
download-fixtures: ## Download benchmark test fixtures from GCS
	@echo "Downloading fixtures from $(FIXTURES_GCS_BUCKET)..."
	gcloud storage rsync -r $(FIXTURES_GCS_BUCKET) ./pkg --include-glob="**/testdata/fixtures/**.json"

.PHONY: upload-fixtures
upload-fixtures: ## Upload local benchmark test fixtures to GCS
	@echo "Uploading fixtures to $(FIXTURES_GCS_BUCKET)..."
	gcloud storage rsync -r ./pkg $(FIXTURES_GCS_BUCKET) --include-glob="**/testdata/fixtures/**.json"
