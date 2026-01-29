# testing.mk
# This file contains make tasks related to testing.

.PHONY: test-web
test-web: $(GENERATE_FRONTEND_DUMMY) $(FRONTEND_SOURCE_FILES)## Run frontend tests
	cd web && npx ng test --watch=false

.PHONY: test-go
test-go: $(GENERATE_BACKEND_DUMMY) $(BACKEND_TEST_SRCS) $(FRONTEND_ARTIFACT_FILES_DUMMY) ## Run backend tests
	go test ./...

.PHONY: coverage-web
coverage-web: $(GENERATE_FRONTEND_DUMMY) $(FRONTEND_SOURCE_FILES)## Run frontend tests and generate coverage report
	cd web && npx ng test --code-coverage --browsers ChromeHeadlessNoSandbox --watch false --progress false

.PHONY: coverage-go
coverage-go: $(GENERATE_BACKEND_DUMMY) $(BACKEND_TEST_SRCS) $(FRONTEND_ARTIFACT_FILES_DUMMY)## Run backend tests and generate coverage report
	go test -cover ./... -coverprofile=./go-cover.output
	go tool cover -html=./go-cover.output -o=go-cover.html
