#!/bin/bash
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

# Runs unified Go and TypeScript differential code quality checks in Cloud Build.
#
# Required tools: git, go, make, node, npm, chromium (preinstalled in builder:latest).

set -euo pipefail

readonly REPO_URL="${REPO_URL:-https://github.com/GoogleCloudPlatform/khi.git}"
readonly BASE_BRANCH="${BASE_BRANCH:-main}"
readonly CODEQUALITY_WORKERS="${CODEQUALITY_WORKERS:-0}"
readonly CODEQUALITY_MAX="${CODEQUALITY_MAX:-0}"
readonly CODEQUALITY_TIMEOUT="${CODEQUALITY_TIMEOUT:-10m}"
readonly CODEQUALITY_SIMILARITY="${CODEQUALITY_SIMILARITY:-false}"
readonly CODEQUALITY_SIMILARITY_PROJECT="${CODEQUALITY_SIMILARITY_PROJECT:-}"
readonly SKIP_FETCH="${SKIP_FETCH:-false}"

configure_git_and_fetch_base() {
  if [[ -n "${BUILD_ID:-}" || -n "${BUILDER_OUTPUT:-}" ]]; then
    git config --global --add safe.directory "$(pwd)"
  fi

  if [[ "${SKIP_FETCH}" != "true" ]]; then
    echo "Fetching base branch '${BASE_BRANCH}' from ${REPO_URL}..."
    git fetch --no-tags --depth=200 "${REPO_URL}" \
      "+refs/heads/${BASE_BRANCH}:refs/remotes/origin/${BASE_BRANCH}"

    if [[ "$(git rev-parse --is-shallow-repository)" == "true" ]]; then
      git fetch --no-tags --deepen=200 "${REPO_URL}" HEAD || true
    fi
    if ! git merge-base "origin/${BASE_BRANCH}" HEAD >/dev/null 2>&1; then
      echo "Deepening repository history to locate merge-base..."
      git fetch --no-tags --unshallow "${REPO_URL}" || true
    fi
  fi
}

ensure_embed_placeholder() {
  mkdir -p pkg/server/dist/browser
  if [[ ! -f pkg/server/dist/browser/index.html ]]; then
    printf '<!doctype html><html><body></body></html>\n' > pkg/server/dist/browser/index.html
  fi
}

prepare_workspace_artifacts() {
  local merge_base="$1"
  local changed_files
  changed_files="$(git diff --name-only "${merge_base}")"

  local has_go="false"
  local has_server_or_cmd="false"
  local has_ts="false"
  local has_codequality="false"

  local path
  while IFS= read -r path; do
    [[ -z "${path}" ]] && continue
    case "${path}" in
      pkg/server/*|cmd/*)
        has_go="true"
        has_server_or_cmd="true"
        ;;
      pkg/generated/*|pkg/testutil/*|web/*|*/zzz_*.go|zzz_*.go)
        ;;
      *.go)
        has_go="true"
        ;;
    esac

    case "${path}" in
      web/src/app/generated/*|web/src/environments/*|*/zzz_*|*/zzz-*)
        ;;
      web/src/*.ts)
        has_ts="true"
        ;;
    esac

    case "${path}" in
      scripts/codequality/*)
        has_codequality="true"
        ;;
    esac
  done <<< "${changed_files}"

  if [[ "${has_go}" == "false" && "${has_ts}" == "false" ]]; then
    echo "No target Go or TypeScript changes detected; skipping workspace codegen."
    return
  fi

  if [[ "${has_server_or_cmd}" == "true" ]]; then
    echo "Server/CLI changes detected; building frontend bundle and backend codegen..."
    (cd ./web && npm ci)
    make build-web
    make generate-backend
    return
  fi

  if [[ "${has_ts}" == "true" ]]; then
    echo "Frontend TypeScript changes detected; installing web dependencies and generating frontend assets..."
    (cd ./web && npm ci)
    touch scripts/make/build-proto.done
    make generate-frontend
  fi

  if [[ "${has_ts}" == "false" && "${has_codequality}" == "true" ]]; then
    echo "Codequality tool changes detected; installing web dependencies for TypeScript AST tests..."
    (cd ./web && npm ci)
  fi

  if [[ "${has_go}" == "true" ]]; then
    echo "Backend Go changes detected; generating backend code..."
    touch scripts/make/build-proto.done
    make generate-backend
    ensure_embed_placeholder
  fi
}

main() {
  configure_git_and_fetch_base

  local merge_base
  merge_base="$(git merge-base "origin/${BASE_BRANCH}" HEAD)"
  local head_sha
  head_sha="$(git rev-parse --short=12 HEAD)"
  local cpu_count
  cpu_count="$(nproc 2>/dev/null || echo unknown)"

  echo "=================================================================="
  echo "KHI Cloud Build Code Quality Gate"
  echo "PR: #${PR_NUMBER:-local} | Base: origin/${BASE_BRANCH} (${merge_base:0:12}) -> HEAD (${head_sha}) | CPUs: ${cpu_count}"
  echo "Max Mutants: ${CODEQUALITY_MAX} | Timeout: ${CODEQUALITY_TIMEOUT} | Similarity: ${CODEQUALITY_SIMILARITY}"
  echo "=================================================================="

  prepare_workspace_artifacts "${merge_base}"

  local -a similarity_args=("--similarity=${CODEQUALITY_SIMILARITY}")
  if [[ -n "${CODEQUALITY_SIMILARITY_PROJECT}" ]]; then
    similarity_args+=("--similarity-project=${CODEQUALITY_SIMILARITY_PROJECT}")
  fi

  KHI_SKIP_CLOUD_LOGGING=true go run ./scripts/codequality \
    --lang=all \
    --vcs=git \
    --base="origin/${BASE_BRANCH}" \
    --workers="${CODEQUALITY_WORKERS}" \
    --max-mutants="${CODEQUALITY_MAX}" \
    --timeout="${CODEQUALITY_TIMEOUT}" \
    "${similarity_args[@]}" \
    --color=never
}

main "$@"
