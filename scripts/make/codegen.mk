# codegen.mk
# This file contains make tasks for generating config or source code.

# prepare-frontend make task generates source code or configurations needed for building frontend code.
# This task needs to be set as a dependency of any make tasks using frontend code.
.PHONY=prepare-frontend
prepare-frontend: web/angular.json web/src/app/generated.sass web/src/app/generated.ts

web/angular.json: scripts/generate-angular-json.sh web/angular-template.json web/src/environments/environment.*.ts
	./scripts/generate-angular-json.sh > ./web/angular.json

# These frontend files are generated from Golang template.
web/src/app/generated.sass web/src/app/generated.ts: pkg/model/enum/log_type.go pkg/model/enum/parent_relationship.go pkg/model/enum/revision_state.go pkg/model/enum/severity.go pkg/model/enum/verb.go 
	go run ./scripts/frontend-codegen