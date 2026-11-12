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

COLOR ?= auto
MUTATION_BASE ?=
MUTATION_TARGET ?=
MUTATION_VCS ?=
MUTATION_FULL ?= false
MUTATION_MAX ?= 80
MUTATION_COLOR ?= $(COLOR)

MUTATION_DIFF_FLAGS := --color=$(MUTATION_COLOR) --max-mutants=$(MUTATION_MAX) $(if $(MUTATION_VCS),--vcs=$(MUTATION_VCS),) $(if $(MUTATION_BASE),--base=$(MUTATION_BASE),) $(if $(MUTATION_TARGET),--target=$(MUTATION_TARGET),) $(if $(filter true 1,$(MUTATION_FULL)),--full,)

.PHONY: coverage-go
coverage-go: $(GENERATE_BACKEND_DUMMY) $(BACKEND_TEST_SRCS) $(FRONTEND_ARTIFACT_FILES_DUMMY)## Run backend tests and generate coverage report
	KHI_SKIP_CLOUD_LOGGING=true go test -cover ./... -coverprofile=./go-cover.output
	go tool cover -html=./go-cover.output -o=go-cover.html

.PHONY: mutation-diff
ifneq ($(filter true 1,$(USE_CONTAINER)),)
mutation-diff: $(GENERATE_BACKEND_DUMMY) $(GENERATE_FRONTEND_DUMMY) build-builder ## Run unified Go and TypeScript differential coverage, mutation testing, and redundant test detection
	@if [ ! -f $(FRONTEND_ARTIFACT_FILES_DUMMY) ]; then \
		$(MAKE) build-web; \
	fi
	USE_CONTAINER=true CONTAINER_CMD="$(CONTAINER_CMD)" BUILDER_IMAGE="$(BUILDER_IMAGE)" go run ./scripts/mutation-diff --lang=all $(MUTATION_DIFF_FLAGS)
	@touch scripts/make/mutation-diff.done
else
mutation-diff: $(GENERATE_BACKEND_DUMMY) $(GENERATE_FRONTEND_DUMMY) ## Run unified Go and TypeScript differential coverage, mutation testing, and redundant test detection
	@if [ ! -f $(FRONTEND_ARTIFACT_FILES_DUMMY) ]; then \
		$(MAKE) build-web; \
	fi
	go run ./scripts/mutation-diff --lang=all $(MUTATION_DIFF_FLAGS)
	@touch scripts/make/mutation-diff.done
endif

.PHONY: mutation-diff-go
mutation-diff-go: $(GENERATE_BACKEND_DUMMY) ## Run unified Go differential coverage, mutation testing, and redundant test detection
	@if [ ! -f $(FRONTEND_ARTIFACT_FILES_DUMMY) ]; then \
		$(MAKE) build-web; \
	fi
	go run ./scripts/mutation-diff --lang=go $(MUTATION_DIFF_FLAGS)
	@touch scripts/make/mutation-diff.done

.PHONY: mutation-diff-web
ifneq ($(filter true 1,$(USE_CONTAINER)),)
mutation-diff-web: mutation-diff-web-container
else
mutation-diff-web: $(GENERATE_FRONTEND_DUMMY) ## Run unified TypeScript differential coverage, mutation testing, and redundant test detection
	go run ./scripts/mutation-diff --lang=ts $(MUTATION_DIFF_FLAGS)
	@touch scripts/make/mutation-diff.done
endif

.PHONY: mutation-diff-web-container
mutation-diff-web-container: $(GENERATE_FRONTEND_DUMMY) build-builder ## Run unified TypeScript differential coverage and mutation testing using container Karma
	USE_CONTAINER=true CONTAINER_CMD="$(CONTAINER_CMD)" BUILDER_IMAGE="$(BUILDER_IMAGE)" go run ./scripts/mutation-diff --lang=ts $(MUTATION_DIFF_FLAGS)
	@touch scripts/make/mutation-diff.done

FIXTURES_GCS_BUCKET ?= gs://khi-fixtures

.PHONY: download-fixtures
download-fixtures: ## Download benchmark test fixtures from GCS
	@echo "Downloading fixtures from $(FIXTURES_GCS_BUCKET)..."
	gcloud storage rsync -r $(FIXTURES_GCS_BUCKET) ./pkg --include-glob="**/testdata/fixtures/**.json"

.PHONY: upload-fixtures
upload-fixtures: ## Upload local benchmark test fixtures to GCS
	@echo "Uploading fixtures to $(FIXTURES_GCS_BUCKET)..."
	gcloud storage rsync -r ./pkg $(FIXTURES_GCS_BUCKET) --include-glob="**/testdata/fixtures/**.json"
