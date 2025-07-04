.DEFAULT_GOAL := help

VERSION=$(shell cat ./VERSION)
GIT_SHORT_HASH=$(shell git rev-parse --short HEAD)
GIT_TAG_NAME="release-"$(VERSION)

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

## Format
.PHONY: format
format: format-web format-go ## Format all source code

# ====================================================================================
#  Setup
# ====================================================================================

.PHONY: setup-hooks
setup-hooks: ## Set up git hooks
	cp ./scripts/pre-commit .git/hooks/
	chmod +x .git/hooks/pre-commit

# ====================================================================================
#  Utils
# ====================================================================================

.PHONY: help
help: ## Show this help message
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)


