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

# lint warning contains lint rules that is warning at this moment but should be fixed long term.
.PHONY=lint-warning
lint-warning: generate-depguard-rules
	 golangci-lint run --config=.generated-golangci-depguard.yaml


.PHONY=format
format: format-web format-go

.PHONY=generate-depguard-rules
generate-depguard-rules:
	cd ./scripts/depguard-generator/ && go run .

### Initial setup

.PHONY=setup-hooks
setup-hooks:
	cp ./scripts/pre-commit .git/hooks/
	chmod +x .git/hooks/pre-commit