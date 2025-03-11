# Makefile for linting Markdown files using markdownlint

.PHONY: markdownlint check-markdownlint

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
	@echo "Linting complete." 