# codegen.mk
# This file contains make tasks for generating config or source code.

FRONTEND_CODEGEN_DIR = scripts/frontend-codegen
ENUM_GO_ALL_FILES := $(shell find pkg/model/enum -name "*.go")
ENUM_GO_FILES := $(filter-out %_test.go,$(ENUM_GO_ALL_FILES))

ZZZ_GO_FILES := $(shell find . -name "zzz_*.go")
NON_ZZZ_NON_TEST_GO_FILES := $(shell find . -type f -name "*.go" ! -name "*_test.go" ! -name "zzz*")

FRONTEND_CODEGEN_DEPS := $(wildcard $(FRONTEND_CODEGEN_DIR)/*.go $(FRONTEND_CODEGEN_DIR)/templates/*)
FRONTEND_CODEGEN_TARGETS = web/src/app/generated.scss web/src/app/generated.ts scripts/msdf-generator/zzz_generated_used_icons.json

BACKEND_CODEGEN_DIR = scripts/backend-codegen

MSDF_GENERATOR_TTFS_TARGETS =scripts/msdf-generator/node_modules/@fontsource/roboto/files/roboto-latin-700-normal.ttf scripts/msdf-generator/node_modules/material-symbols/material-symbols-outlined.ttf
MSDF_GENERATOR_TARGETS= web/src/assets/icon-codepoints.json web/src/assets/material-icons-msdf.json web/src/assets/material-icons-msdf.png web/src/assets/roboto-number-msdf.json web/src/assets/roboto-number-msdf.png

# prepare-frontend make task generates source code or configurations needed for building frontend code.
# This task needs to be set as a dependency of any make tasks using frontend code.
.PHONY: prepare-frontend
prepare-frontend: web/angular.json web/src/environments/version.*.ts $(FRONTEND_CODEGEN_TARGETS) $(MSDF_GENERATOR_TARGETS)

web/angular.json: scripts/generate-angular-json.sh web/angular-template.json web/src/environments/environment.*.ts
	./scripts/generate-angular-json.sh > ./web/angular.json

# These frontend files are generated from Golang template.
$(FRONTEND_CODEGEN_TARGETS): $(ENUM_GO_FILES) $(FRONTEND_CODEGEN_DEPS)
	go run ./$(FRONTEND_CODEGEN_DIR)

# Generate web/src/environments/version.dev.ts and web/src/environments/version.prod.ts
web/src/environments/version.*.ts: VERSION
	./scripts/generate-version.sh

$(ZZZ_GO_FILES): $(NON_ZZZ_NON_TEST_GO_FILES) ## Generate backend source code
	go run ./scripts/backend-codegen/

$(MSDF_GENERATOR_TARGETS): scripts/msdf-generator/zzz_generated_used_icons.json $(MSDF_GENERATOR_TTFS_TARGETS) ## Generate font atlas
	cd scripts/msdf-generator && node index.js

$(MSDF_GENERATOR_TTFS_TARGETS): 
	cd scripts/msdf-generator && npm i

.PHONY: add-licenses
add-licenses: ## Add license headers to all files
	go tool addlicense  -c "Google LLC" -l apache .

.PHONY: generate-reference
generate-reference: ## Generate reference documentation
	go run ./cmd/reference-generator/
