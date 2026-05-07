
.PHONY: pre-commit
pre-commit: ## Run pre-commit checks (internal)
	@if git rev-parse --is-inside-work-tree >/dev/null 2>&1 && [ -n "$$(git diff --cached --name-only 2>/dev/null)" ]; then \
		scripts_files=$$(git diff --cached --name-only --diff-filter=ACMR | sed 's| |\\ |g'); \
		if [ -n "$$scripts_files" ]; then \
			$(MAKE) add-licenses && \
			$(MAKE) format && \
			$(MAKE) lint && \
			$(MAKE) format-md && \
			$(MAKE) lint-md && \
			echo "$$scripts_files" | xargs git add; \
		fi \
	elif jj root >/dev/null 2>&1; then \
		scripts_files=$$(jj diff --name-only | while IFS= read -r f; do [ -e "$$f" ] && echo "$$f"; done | sed 's| |\\ |g'); \
		if [ -n "$$scripts_files" ]; then \
			$(MAKE) add-licenses && \
			$(MAKE) format && \
			$(MAKE) lint && \
			$(MAKE) format-md && \
			$(MAKE) lint-md; \
		fi \
	fi
