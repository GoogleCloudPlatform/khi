export NODE_OPTIONS ?= --max-old-space-size=6144
BASE ?=

.PHONY: pre-commit
pre-commit: ## Auto-format and run fast static checks on modified files in working copy
	@/bin/bash ./scripts/pre-commit.sh

.PHONY: pre-push
pre-push: ## Run full CI-parity verification gates on changed branch files (optional: BASE=<rev>)
	@BASE="$(BASE)" /bin/bash ./scripts/pre-push.sh
