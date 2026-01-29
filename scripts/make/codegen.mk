# codegen.mk
# This file contains make tasks for generating config or source code.

$(GENERATE_FRONTEND_DUMMY): web/angular.json web/src/environments/version.*.ts web/src/app/zzz-generated.scss web/src/app/zzz-generated.ts $(FRONTEND_GENERATED_ASSETS_DUMMY)
	touch $(GENERATE_FRONTEND_DUMMY)
.PHONY: generate-frontend
generate-frontend: $(GENERATE_FRONTEND_DUMMY) ## Generate frontend source code

web/angular.json: scripts/generate-angular-json.sh web/angular-template.json web/src/environments/environment.*.ts
	./scripts/generate-angular-json.sh > ./web/angular.json

# These frontend files are generated from Golang template.
web/src/app/zzz-generated.scss web/src/app/zzz-generated.ts scripts/msdf-generator/zzz_generated_used_icons.json: $(ENUM_GO_FILES) $(FRONTEND_CODEGEN_DEPS)
	go run ./scripts/frontend-codegen

# Generate web/src/environments/version.dev.ts and web/src/environments/version.prod.ts
web/src/environments/version.*.ts: VERSION
	./scripts/generate-version.sh

$(GENERATE_BACKEND_DUMMY): ## Generate backend source code
	go run ./scripts/backend-codegen/
	touch $(GENERATE_BACKEND_DUMMY)
.PHONY: generate-backend
 generate-backend: $(GENERATE_BACKEND_DUMMY) ## Generate backend source code

$(FRONTEND_GENERATED_ASSETS_DUMMY): scripts/msdf-generator/index.js scripts/msdf-generator/zzz_generated_used_icons.json $(MSDF_SETUP_DUMMY)## Generate font atlas
	cd scripts/msdf-generator && node index.js
	touch $(FRONTEND_GENERATED_ASSETS_DUMMY)

generate-frontend-assets: $(FRONTEND_GENERATED_ASSETS_DUMMY) ## Generate font atlas

$(MSDF_SETUP_DUMMY):
	cd scripts/msdf-generator && npm i
	touch $(MSDF_SETUP_DUMMY)

.PHONY: add-licenses
add-licenses: ## Add license headers to all files
	go tool addlicense  -c "Google LLC" -l apache .

.PHONY: generate-reference
generate-reference: ## Generate reference documentation
	go run ./cmd/reference-generator/
