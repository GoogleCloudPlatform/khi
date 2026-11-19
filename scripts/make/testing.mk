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

COLOR ?= auto
CODEQUALITY_BASE ?=
CODEQUALITY_TARGET ?=
CODEQUALITY_VCS ?=
CODEQUALITY_FULL ?= false
CODEQUALITY_WORKERS ?= 0
CODEQUALITY_MAX ?= 80
CODEQUALITY_TIMEOUT ?= 3m
-include $(HOME)/.config/khi/codequality.mk
CODEQUALITY_SIMILARITY ?= false
CODEQUALITY_SIMILARITY_PROJECT ?=
CODEQUALITY_DECISIONS ?= false
CODEQUALITY_DECISIONS_PROJECT ?=
CODEQUALITY_DECISIONS_RULES ?=
CODEQUALITY_DECISIONS_FAIL_ON_VIOLATIONS ?= false
CODEQUALITY_COLOR ?= $(COLOR)

CODEQUALITY_DECISIONS_FLAGS := --decisions=$(CODEQUALITY_DECISIONS) $(if $(CODEQUALITY_DECISIONS_PROJECT),--decisions-project="$(CODEQUALITY_DECISIONS_PROJECT)",) $(if $(CODEQUALITY_DECISIONS_RULES),--decisions-rules="$(CODEQUALITY_DECISIONS_RULES)",) --decisions-fail-on-violations=$(CODEQUALITY_DECISIONS_FAIL_ON_VIOLATIONS)
CODEQUALITY_FLAGS := --color=$(CODEQUALITY_COLOR) --similarity=$(CODEQUALITY_SIMILARITY) $(if $(CODEQUALITY_SIMILARITY_PROJECT),--similarity-project="$(CODEQUALITY_SIMILARITY_PROJECT)",) $(CODEQUALITY_DECISIONS_FLAGS) --workers=$(CODEQUALITY_WORKERS) --max-mutants=$(CODEQUALITY_MAX) --timeout=$(CODEQUALITY_TIMEOUT) $(if $(CODEQUALITY_VCS),--vcs="$(CODEQUALITY_VCS)",) $(if $(CODEQUALITY_BASE),--base="$(CODEQUALITY_BASE)",) $(if $(CODEQUALITY_TARGET),--target="$(CODEQUALITY_TARGET)",) $(if $(filter true 1,$(CODEQUALITY_FULL)),--full,)

.PHONY: coverage-go
coverage-go: $(GENERATE_BACKEND_DUMMY) $(BACKEND_TEST_SRCS) $(FRONTEND_ARTIFACT_FILES_DUMMY)## Run backend tests and generate coverage report
	KHI_SKIP_CLOUD_LOGGING=true go test -cover ./... -coverprofile=./go-cover.output
	go tool cover -html=./go-cover.output -o=go-cover.html

.PHONY: codequality-fast
codequality-fast: ## Run fast Go AST structural complexity, size, naming, ratchet, and function similarity checks on changed files
	go run ./scripts/codequality --lang=go --fast --similarity=$(CODEQUALITY_SIMILARITY) $(if $(CODEQUALITY_SIMILARITY_PROJECT),--similarity-project="$(CODEQUALITY_SIMILARITY_PROJECT)",) $(CODEQUALITY_DECISIONS_FLAGS) --color=$(CODEQUALITY_COLOR) $(if $(CODEQUALITY_VCS),--vcs="$(CODEQUALITY_VCS)",) $(if $(CODEQUALITY_BASE),--base="$(CODEQUALITY_BASE)",) $(if $(CODEQUALITY_TARGET),--target="$(CODEQUALITY_TARGET)",) $(if $(filter true 1,$(CODEQUALITY_FULL)),--full,)

.PHONY: codequality-decisions
codequality-decisions: ## Run declarative YAML Gemini Decisions semantic rules on changed files
	go run ./scripts/codequality --decisions-only $(if $(CODEQUALITY_DECISIONS_PROJECT),--decisions-project="$(CODEQUALITY_DECISIONS_PROJECT)",) $(if $(CODEQUALITY_DECISIONS_RULES),--decisions-rules="$(CODEQUALITY_DECISIONS_RULES)",) --decisions-fail-on-violations=$(CODEQUALITY_DECISIONS_FAIL_ON_VIOLATIONS) --color=$(CODEQUALITY_COLOR) $(if $(CODEQUALITY_VCS),--vcs="$(CODEQUALITY_VCS)",) $(if $(CODEQUALITY_BASE),--base="$(CODEQUALITY_BASE)",) $(if $(CODEQUALITY_TARGET),--target="$(CODEQUALITY_TARGET)",) $(if $(filter true 1,$(CODEQUALITY_FULL)),--full,)

.PHONY: codequality-ci
codequality-ci: ## Run unified differential code quality checks for Cloud Build CI
	/bin/bash ./scripts/run-codequality-ci.sh

.PHONY: post-codequality-comment
post-codequality-comment: ## Post or update sticky GitHub PR comment from completed Cloud Build Check Run
	/bin/bash ./scripts/post-codequality-pr-comment.sh

.PHONY: codequality
ifneq ($(filter true 1,$(USE_CONTAINER)),)
codequality: $(GENERATE_BACKEND_DUMMY) $(GENERATE_FRONTEND_DUMMY) build-builder
	@if [ ! -f $(FRONTEND_ARTIFACT_FILES_DUMMY) ]; then \
		$(MAKE) build-web; \
	fi
	USE_CONTAINER=true CONTAINER_CMD="$(CONTAINER_CMD)" BUILDER_IMAGE="$(BUILDER_IMAGE)" go run ./scripts/codequality --lang=all $(CODEQUALITY_FLAGS)
	@touch scripts/make/codequality.done
else
codequality: $(GENERATE_BACKEND_DUMMY) $(GENERATE_FRONTEND_DUMMY) ## Run unified Go and TypeScript code and test quality checks
	@if [ ! -f $(FRONTEND_ARTIFACT_FILES_DUMMY) ]; then \
		$(MAKE) build-web; \
	fi
	go run ./scripts/codequality --lang=all $(CODEQUALITY_FLAGS)
	@touch scripts/make/codequality.done
endif

.PHONY: codequality-go
codequality-go: $(GENERATE_BACKEND_DUMMY) ## Run unified Go quality rules and ratchets (AST structure, coverage, mutation testing, and test redundancy)
	@if [ ! -f $(FRONTEND_ARTIFACT_FILES_DUMMY) ]; then \
		$(MAKE) build-web; \
	fi
	go run ./scripts/codequality --lang=go $(CODEQUALITY_FLAGS)
	@touch scripts/make/codequality.done

.PHONY: codequality-web
ifneq ($(filter true 1,$(USE_CONTAINER)),)
codequality-web: codequality-web-container
else
codequality-web: $(GENERATE_FRONTEND_DUMMY) ## Run unified TypeScript differential coverage, mutation testing, and redundant spec detection
	go run ./scripts/codequality --lang=ts $(CODEQUALITY_FLAGS)
	@touch scripts/make/codequality.done
endif

.PHONY: codequality-web-container
codequality-web-container: $(GENERATE_FRONTEND_DUMMY) build-builder ## Run unified TypeScript differential coverage and mutation testing using container Karma
	USE_CONTAINER=true CONTAINER_CMD="$(CONTAINER_CMD)" BUILDER_IMAGE="$(BUILDER_IMAGE)" go run ./scripts/codequality --lang=ts $(CODEQUALITY_FLAGS)
	@touch scripts/make/codequality.done

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
