# testing.mk
# This file contains make tasks related to testing.

.PHONY: test-web
test-web: prepare-frontend ## Run frontend tests
	cd web && npx ng test --watch=false

.PHONY: test-go
test-go: generate-backend ## Run backend tests
	go test ./...

.PHONY: coverage-web
coverage-web: prepare-frontend ## Run frontend tests and generate coverage report
	cd web && npx ng test --code-coverage --browsers ChromeHeadlessNoSandbox --watch false --progress false

.PHONY: coverage-go
coverage-go: generate-backend ## Run backend tests and generate coverage report
	go test -cover ./... -coverprofile=./go-cover.output
	go tool cover -html=./go-cover.output -o=go-cover.html
