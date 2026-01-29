.DEFAULT_GOAL := help

VERSION=$(shell cat ./VERSION)
GIT_SHORT_HASH=$(shell git rev-parse --short HEAD)
GIT_TAG_NAME="release-"$(VERSION)

DUMMY_DIR := scripts/make
GENERATE_FRONTEND_DUMMY := $(DUMMY_DIR)/generate-frontend.done
GENERATE_BACKEND_DUMMY := $(DUMMY_DIR)/generate-backend.done
FRONTEND_GENERATED_ASSETS_DUMMY := $(DUMMY_DIR)/generate-font-atlas.done
MSDF_SETUP_DUMMY := $(DUMMY_DIR)/msdf-setup.done

BACKEND_TEST_SRCS = $(shell find . -name "*_test.go" -not -path "web/*" -not -path "zzz_*.go")
BACKEND_SRCS = $(shell find . -name "*.go" -not -path "web/*" -not -path "zzz_*.go" -not -path "*_test.go")
ENUM_GO_ALL_FILES := $(wildcard pkg/model/enum/*.go)
ENUM_GO_FILES := $(filter-out %_test.go,$(ENUM_GO_ALL_FILES))

FRONTEND_CODEGEN_DIR := scripts/frontend-codegen
FRONTEND_CODEGEN_DEPS := $(wildcard $(FRONTEND_CODEGEN_DIR)/*.go $(FRONTEND_CODEGEN_DIR)/templates/*)
FRONTEND_SOURCE_FILES = $(shell find web -not -path "*/node_modules/*" -not -path "*/.angular/*" -not -path "web/src/assets/*" -not -path "web/src/environments/version.*.ts" -not -path "*/zzz-generated.*" -not -path "web/angular.json")
FRONTEND_GENERATED_SRCS = web/src/app/zzz-generated.scss web/src/app/zzz-generated.ts web/angular.json
FRONTEND_ARTIFACT_FILES_DUMMY = pkg/server/dist/browser/build-web.done

include scripts/make/*.mk

# ====================================================================================
#  Development commands
# ====================================================================================

## Test
.PHONY: test
test: test-web test-go ## Run all tests

.PHONY: coverage
coverage: coverage-go coverage-web ## Run all tests and generate coverage report

## Lint
.PHONY: lint
lint: lint-web lint-go ## Run all linters

.PHONY: lint-warning
lint-warning: generate-depguard-rules ## Lint warning contains lint rules that is warning at this moment but should be fixed long term.
	 golangci-lint run --config=.generated-golangci-depguard.yaml

.PHONY: generate-depguard-rules
generate-depguard-rules: ## Generate depguard rule from Go source. This rule prevents packages being imported from unexpected package and enforce packages to follow the package structure rule.
	cd ./scripts/depguard-generator/ && go run . --package-root=../.. --output=../../.generated-golangci-depguard.yaml

## Format
.PHONY: format
format: format-web format-go ## Format all source code

# ====================================================================================
#  Setup
# ====================================================================================

.PHONY: setup
setup: setup-hooks
	cd web && npm install
	make build

.PHONY: setup-hooks
setup-hooks: ## Set up git hooks
	@HOOK_DIR=$$(git rev-parse --git-path hooks); \
	PRE_COMMIT_HOOK="$$HOOK_DIR/pre-commit"; \
	mkdir -p "$$HOOK_DIR"; \
	printf '%s\n' '#!/bin/sh' 'cd "$$(git rev-parse --show-toplevel)"' 'exec make pre-commit' > "$$PRE_COMMIT_HOOK"; \
	chmod +x "$$PRE_COMMIT_HOOK"

# ====================================================================================
#  Utils
# ====================================================================================

.PHONY: help
help: ## Show this help message
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)


