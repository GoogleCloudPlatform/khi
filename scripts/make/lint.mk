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

GOLANGCILINT_VERSION := v2.1.6
GOLANGCILINT_CMD ?= $(shell command -v golangci-lint)
CONTAINER_CMD ?= $(shell command -v docker || command -v podman)

.PHONY: lint-web
lint-web: prepare-frontend ## Run frontend linter
	cd web && npx ng lint

.PHONY: lint-go
lint-go: ## Run backend linter
ifeq ($(GOLANGCILINT_CMD),)
	ifeq ($(CONTAINER_CMD),)
		$(error "lint-go requires golangci-lint,docker or podman, but neither was found.")
	else
		$(CONTAINER_CMD) run --rm -v $(CURDIR):/app -w /app golangci/golangci-lint:$(GOLANGCILINT_VERSION) golangci-lint run --config=.golangci.yaml
	endif
else
	@if ! $(GOLANGCILINT_CMD) version | grep -q "$(subst v,,$(GOLANGCILINT_VERSION))"; then \
		printf "\e[0;33mWarning: local golangci-lint version ($$($(GOLANGCILINT_CMD) version)) does not match $(GOLANGCILINT_VERSION). Results may differ from CI.\e[0m\n"; \
	fi
	$(GOLANGCILINT_CMD) run --config=.golangci.yaml
endif

.PHONY: format-go
format-go: ## Format backend source code
	gofmt -s -w .

.PHONY: format-web
format-web: prepare-frontend ## Format frontend source code
	cd web && npx prettier --ignore-path .gitignore --write "./**/*.+(ts|json|html|scss)"

.PHONY: check-format-go
check-format-go: ## Check backend source code format
	test -z `gofmt -l .`

.PHONY: check-format-web
check-format-web: prepare-frontend ## Check frontend source code format
	cd web && npx prettier --ignore-path .gitignore --check "./**/*.+(ts|json|html|scss)"

.PHONY: lint-markdown
lint-markdown: ## Run markdown linter
	npx markdownlint-cli2

.PHONY: lint-markdown-fix
lint-markdown-fix: ## Fix markdown linter errors
	npx markdownlint-cli2 --fix
