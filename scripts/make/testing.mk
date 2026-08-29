# testing.mk
# This file contains make tasks related to testing.

.PHONY: test-web
test-web: $(GENERATE_FRONTEND_DUMMY) $(FRONTEND_SOURCE_FILES)## Run frontend tests
	cd web && npx ng test --browsers ChromeHeadlessNoSandbox --watch=false

.PHONY: watch-test-web
watch-test-web: $(GENERATE_FRONTEND_DUMMY) ## Run frontend tests in watch mode
	cd web && npx ng test

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

FIXTURES_GCS_BUCKET ?= gs://khi-fixtures

.PHONY: download-fixtures
download-fixtures: ## Download benchmark test fixtures from GCS
	@echo "Downloading fixtures from $(FIXTURES_GCS_BUCKET)..."
	gcloud storage rsync -r $(FIXTURES_GCS_BUCKET) ./pkg --include-glob="**/testdata/fixtures/**.json"

.PHONY: upload-fixtures
upload-fixtures: ## Upload local benchmark test fixtures to GCS
	@echo "Uploading fixtures to $(FIXTURES_GCS_BUCKET)..."
	gcloud storage rsync -r ./pkg $(FIXTURES_GCS_BUCKET) --include-glob="**/testdata/fixtures/**.json"
