# testing.mk
# This file contains make tasks related to testing.

.PHONY=test-web
test-web: prepare-frontend
	cd web && ng test --watch=false

.PHONY=test-go
test-go:
	go test ./...

.PHONY=coverage-web
coverage-web: prepare-frontend
	cd web && npx ng test --code-coverage --browsers ChromeHeadlessNoSandbox --watch false --progress false

.PHONY=coverage-go
coverage-go:
	go test -cover ./... -coverprofile=./go-cover.output
	go tool cover -html=./go-cover.output -o=go-cover.html