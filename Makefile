VERSION=$(shell cat ./VERSION)
GIT_SHORT_HASH=$(shell git rev-parse --short HEAD)
GIT_TAG_NAME="release-"$(VERSION)

include scripts/make/*.mk

# Top level commands for development
## Test

.PHONY=test
test: test-web test-go

# Generate the coverage report
.PHONY=coverage
coverage: coverage-go coverage-web

.PHONY=lint
lint: lint-web lint-go

.PHONY=format
format: format-web format-go

### Initial setup

.PHONY=setup-hooks
setup-hooks:
	cp ./scripts/pre-commit .git/hooks/
	chmod +x .git/hooks/pre-commit