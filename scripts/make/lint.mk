# Copyright 2025 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

GOLANGCILINT_VERSION := v1.64.2
GOLANGCILINT_BIN := $(shell go env GOPATH)/bin/golangci-lint

.PHONY=lint-web
lint-web: prepare-frontend
	cd web && npx ng lint

.PHONY=lint-go
lint-go: install-golangci-lint
	$(GOLANGCILINT_BIN) run --config=.golangci.yaml

.PHONY=install-golangci-lint
install-golangci-lint:
	@if ! [ -x "$(GOLANGCILINT_BIN)" ]; then \
		echo "Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCILINT_VERSION); \
	fi

.PHONY=format-go
format-go:
	gofmt -s -w .

.PHONY=format-web
format-web: prepare-frontend
	cd web && npx prettier --ignore-path .gitignore --write "./**/*.+(ts|json|html|scss)"

.PHONY=check-format-go
check-format-go:
	test -z `gofmt -l .`

.PHONY=check-format-web
check-format-web: prepare-frontend
	cd web && npx prettier --ignore-path .gitignore --check "./**/*.+(ts|json|html|scss)"

.PHONY: lint-markdown
lint-markdown:
	npx markdownlint-cli2

.PHONY: lint-markdown-fix
lint-markdown-fix:
	npx markdownlint-cli2 --fix
