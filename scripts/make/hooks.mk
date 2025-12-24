
.PHONY: pre-commit
pre-commit: ## Run pre-commit checks (internal)
	@scripts_files=$$(git diff --cached --name-only --diff-filter=ACMR | sed 's| |\\ |g'); \
	if [ -n "$$scripts_files" ]; then \
		$(MAKE) add-licenses && \
		$(MAKE) format && \
		$(MAKE) lint && \
		$(MAKE) lint-markdown-fix && \
		$(MAKE) lint-markdown && \
		echo "$$scripts_files" | xargs git add; \
	fi
