# codegen.mk
# This file contains make tasks for generating config or source code.

FRONTEND_CODEGEN_DIR = scripts/frontend-codegen
ENUM_GO_ALL_FILES := $(wildcard pkg/model/enum/*.go)
ENUM_GO_FILES := $(filter-out %_test.go,$(ENUM_GO_ALL_FILES))
FRONTEND_CODEGEN_DEPS := $(wildcard $(FRONTEND_CODEGEN_DIR)/*.go $(FRONTEND_CODEGEN_DIR)/templates/*)
FRONTEND_CODEGEN_TARGETS = web/src/app/generated.scss web/src/app/generated.ts

# prepare-frontend make task generates source code or configurations needed for building frontend code.
# This task needs to be set as a dependency of any make tasks using frontend code.
.PHONY: prepare-frontend
prepare-frontend: web/angular.json web/src/environments/version.*.ts $(FRONTEND_CODEGEN_TARGETS)

web/angular.json: scripts/generate-angular-json.sh web/angular-template.json web/src/environments/environment.*.ts
	./scripts/generate-angular-json.sh > ./web/angular.json

# These frontend files are generated from Golang template.
$(FRONTEND_CODEGEN_TARGETS): $(ENUM_GO_FILES) $(FRONTEND_CODEGEN_DEPS)
	go run ./$(FRONTEND_CODEGEN_DIR)

# Generate web/src/environments/version.dev.ts and web/src/environments/version.prod.ts
web/src/environments/version.*.ts: VERSION
	./scripts/generate-version.sh

.PHONY: generate-backend
generate-backend: ## Generate backend source code
	go run ./scripts/backend-codegen/

.PHONY: add-licenses
add-licenses: ## Add license headers to all files
	go tool addlicense  -c "Google LLC" -l apache .

.PHONY: generate-reference
generate-reference: ## Generate reference documentation
	go run ./cmd/reference-generator/
