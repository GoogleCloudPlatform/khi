# Copyright 2024 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

.PHONY: markdownlint check-markdownlint lint-markdown lint-markdown-fix

# Check if markdownlint-cli is installed
check-markdownlint:
	@command -v markdownlint >/dev/null 2>&1 || { echo "markdownlint-cli not found. Please install it with 'npm install -g markdownlint-cli'"; exit 1; }

# Default target: lint all markdown files
markdownlint: check-markdownlint
	@echo "Linting all Markdown files with markdownlint..."
	@if [ ! -f ".markdownlint.json" ] && [ ! -f ".markdownlint.yaml" ] && [ ! -f ".markdownlint.yml" ]; then \
		echo "{ \"default\": true, \"MD013\": false, \"MD033\": false }" > .markdownlint.json; \
		echo "Created default .markdownlint.json configuration"; \
	else \
		echo "Using existing markdownlint configuration"; \
	fi
	@markdownlint "**/*.md" || exit 1

lint-markdown:
	@echo "Linting all Markdown files with markdownlint-cli2..."
	@npx markdownlint-cli2

lint-markdown-fix:
	@echo "Fixing Markdown files with markdownlint-cli2..."
	@npx markdownlint-cli2 --fix
	@echo "Automatic fixes applied. Please review changes before committing." 