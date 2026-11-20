# Copyright 2026 Google LLC
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

# qa.mk
# Defines unified differential QA targets (qa-<speed>[-<filter>]).

-include $(HOME)/.config/khi/codequality.mk

BASE ?=
TARGET ?=
PATHS ?=
CHANGED_SINCE ?=
FORMAT ?= text
QA_LANG ?= all
USE_CONTAINER ?= false
COLOR ?= auto
VCS ?=
CODEQUALITY_WORKERS ?=
CODEQUALITY_MAX ?=
CODEQUALITY_TIMEOUT ?=
CODEQUALITY_SIMILARITY ?=
CODEQUALITY_SIMILARITY_PROJECT ?=
CODEQUALITY_DECISIONS ?=
CODEQUALITY_DECISIONS_PROJECT ?=

QA_COMMON_FLAGS = \
	--lang=$(QA_LANG) \
	--format=$(FORMAT) \
	--color=$(COLOR) \
	$(if $(BASE),--base="$(BASE)",) \
	$(if $(TARGET),--target="$(TARGET)",) \
	$(if $(PATHS),--paths="$(PATHS)",) \
	$(if $(CHANGED_SINCE),--changed-since="$(CHANGED_SINCE)",) \
	$(if $(VCS),--vcs="$(VCS)",) \
	$(if $(CODEQUALITY_WORKERS),--workers="$(CODEQUALITY_WORKERS)",) \
	$(if $(CODEQUALITY_MAX),--max-mutants="$(CODEQUALITY_MAX)",) \
	$(if $(CODEQUALITY_TIMEOUT),--timeout="$(CODEQUALITY_TIMEOUT)",) \
	$(if $(filter true 1,$(USE_CONTAINER)),--use-container,) \
	$(if $(filter true 1,$(CODEQUALITY_SIMILARITY)),--similarity=true,$(if $(filter false 0,$(CODEQUALITY_SIMILARITY)),--similarity=false,)) \
	$(if $(CODEQUALITY_SIMILARITY_PROJECT),--similarity-project="$(CODEQUALITY_SIMILARITY_PROJECT)",) \
	$(if $(filter true 1,$(CODEQUALITY_DECISIONS)),--decisions=true,$(if $(filter false 0,$(CODEQUALITY_DECISIONS)),--decisions=false,)) \
	$(if $(CODEQUALITY_DECISIONS_PROJECT),--decisions-project="$(CODEQUALITY_DECISIONS_PROJECT)",)

.PHONY: qa-fast qa-fast-must qa-fast-advisory qa-slow qa-slow-must qa-slow-advisory qa-all qa-all-must qa-all-advisory

qa-fast: ## Run fast MUSTFIX and ADVISORY QA checks on changed files
	@go run ./scripts/codequality qa --speed=fast --filter=all $(QA_COMMON_FLAGS)

qa-fast-must: ## Run fast blocking MUSTFIX QA checks on changed files
	@go run ./scripts/codequality qa --speed=fast --filter=must $(QA_COMMON_FLAGS)

qa-fast-advisory: ## Run fast non-blocking ADVISORY QA checks on changed files
	@go run ./scripts/codequality qa --speed=fast --filter=advisory $(QA_COMMON_FLAGS)

qa-slow: ## Run slow MUSTFIX and ADVISORY QA checks on changed files
	@go run ./scripts/codequality qa --speed=slow --filter=all $(QA_COMMON_FLAGS)

qa-slow-must: ## Run slow blocking MUSTFIX QA checks on changed files
	@go run ./scripts/codequality qa --speed=slow --filter=must $(QA_COMMON_FLAGS)

qa-slow-advisory: ## Run slow non-blocking ADVISORY QA checks on changed files
	@go run ./scripts/codequality qa --speed=slow --filter=advisory $(QA_COMMON_FLAGS)

qa-all: ## Run all (fast + slow) MUSTFIX and ADVISORY QA checks on changed files
	@go run ./scripts/codequality qa --speed=all --filter=all $(QA_COMMON_FLAGS)

qa-all-must: ## Run all (fast + slow) blocking MUSTFIX QA checks on changed files
	@go run ./scripts/codequality qa --speed=all --filter=must $(QA_COMMON_FLAGS)

qa-all-advisory: ## Run all (fast + slow) non-blocking ADVISORY QA checks on changed files
	@go run ./scripts/codequality qa --speed=all --filter=advisory $(QA_COMMON_FLAGS)
