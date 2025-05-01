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

.PHONY=lint-web
lint-web: prepare-frontend
	cd web && npx ng lint

.PHONY=lint-go
lint-go:
	go vet ./...
.PHONY=format-go
format-go:
	gofmt -s -w .

.PHONY=format-web
format-web: prepare-frontend
	cd web && npx prettier --ignore-path .gitignore --write "./**/*.+(scss|ts|json|html)"

.PHONY=check-format-go
check-format-go:
	test -z `gofmt -l .`

.PHONY=check-format-web
check-format-web: prepare-frontend
	cd web && npx prettier --ignore-path .gitignore --check "./**/*.+(scss|ts|json|html)"

.PHONY: lint-markdown
lint-markdown:
	npx markdownlint-cli2

.PHONY: lint-markdown-fix
lint-markdown-fix:
	npx markdownlint-cli2 --fix
