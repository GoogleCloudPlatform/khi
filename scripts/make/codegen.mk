# codegen.mk
# This file contains make tasks for generating config or source code.

$(GENERATE_FRONTEND_DUMMY): web/angular.json web/src/environments/version.*.ts $(FRONTEND_GENERATED_ASSETS_DUMMY)
	touch $(GENERATE_FRONTEND_DUMMY)
.PHONY: generate-frontend
generate-frontend: $(GENERATE_FRONTEND_DUMMY) ## Generate frontend source code

web/angular.json: scripts/generate-angular-json.sh web/angular-template.json web/src/environments/environment.*.ts
	./scripts/generate-angular-json.sh > ./web/angular.json

# These frontend files are generated from Golang template.

.PHONY: fontlist-gen
fontlist-gen: generate-backend
	go run ./scripts/fontlist-gen

# Generate web/src/environments/version.dev.ts and web/src/environments/version.prod.ts
web/src/environments/version.*.ts: VERSION
	./scripts/generate-version.sh

$(GENERATE_BACKEND_DUMMY): ## Generate backend source code
	go run ./scripts/backend-codegen/
	touch $(GENERATE_BACKEND_DUMMY)
.PHONY: generate-backend
 generate-backend: $(GENERATE_BACKEND_DUMMY) ## Generate backend source code

# TODO: eventually the following cp commands are not needed after we removed icon image dependency directly from the frontend.
$(FRONTEND_GENERATED_ASSETS_DUMMY): fontlist-gen scripts/msdf-generator/index.js scripts/msdf-generator/zzz_generated_used_icons.json $(MSDF_SETUP_DUMMY)## Generate font atlas
	cd scripts/msdf-generator && node index.js
	mkdir -p pkg/model/khifile/v6/style/assets
	cp web/src/assets/zzz-icon-codepoints.json pkg/model/khifile/v6/style/assets/
	cp web/src/assets/zzz-material-icons-msdf.json pkg/model/khifile/v6/style/assets/
	cp web/src/assets/zzz-material-icons-msdf.png pkg/model/khifile/v6/style/assets/
	touch $(FRONTEND_GENERATED_ASSETS_DUMMY)

generate-frontend-assets: $(FRONTEND_GENERATED_ASSETS_DUMMY) ## Generate font atlas

$(MSDF_SETUP_DUMMY):
	cd scripts/msdf-generator && npm i
	touch $(MSDF_SETUP_DUMMY)

.PHONY: add-licenses
add-licenses: ## Add license headers to all files
	go tool addlicense  -c "Google LLC" -l apache .

.PHONY: build-proto
build-proto: ## Generate code from protobuf definitions
	npx @bufbuild/buf generate

