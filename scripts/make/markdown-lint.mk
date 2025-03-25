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

.PHONY: check-markdownlint lint-markdown lint-markdown-fix

# Check if markdownlint-cli2 is installed
check-markdownlint:
	@command -v npx >/dev/null 2>&1 || { echo "npx not found. Please install Node.js and npm"; exit 1; }

# Lint all markdown files
lint-markdown: check-markdownlint
	@echo "Linting all Markdown files with markdownlint-cli2..."
	@npx markdownlint-cli2

# Fix markdown files
lint-markdown-fix: check-markdownlint
	@echo "Fixing Markdown files with markdownlint-cli2..."
	@npx markdownlint-cli2 --fix
	@echo "Automatic fixes applied. Please review changes before committing." 