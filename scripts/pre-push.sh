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


# Runs full CI-parity verification gates (GitHub Actions PR Tests + Cloud Build
# github-codequality with unlimited mutants) scoped to files changed on the
# current branch relative to BASE.
#
# Usage:
#   make pre-push
#   make pre-push BASE=<parent-pr-bookmark>
#
# Required tools: bash, make, and jj or git.

set -euo pipefail

readonly REPO_ROOT="${KHI_REPO_ROOT:-$(cd "$(dirname "$0")/.." && pwd)}"
readonly MAKE_BIN="${MAKE_BIN:-make}"
readonly VCS_MODE="${VCS:-${CODEQUALITY_VCS:-auto}}"
readonly REQUESTED_BASE="${BASE:-${CODEQUALITY_BASE:-}}"
readonly REQUESTED_TARGET="${TARGET:-${CODEQUALITY_TARGET:-}}"
readonly CQ_MAX="${CODEQUALITY_MAX:-0}"
readonly CQ_TIMEOUT="${CODEQUALITY_TIMEOUT:-10m}"

resolve_vcs() {
  case "${VCS_MODE}" in
    jj|git)
      echo "${VCS_MODE}"
      return
      ;;
    auto)
      if [[ -d "${REPO_ROOT}/.jj" ]] && command -v jj >/dev/null 2>&1; then
        echo "jj"
        return
      fi
      if git -C "${REPO_ROOT}" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
        echo "git"
        return
      fi
      echo "Error: neither jj nor git repository detected in ${REPO_ROOT}." >&2
      exit 1
      ;;
    *)
      echo "Error: unsupported VCS=${VCS_MODE} (expected auto, jj, or git)." >&2
      exit 1
      ;;
  esac
}

resolve_default_git_base() {
  local candidate
  for candidate in origin/main main HEAD~1 HEAD; do
    if git rev-parse --verify "${candidate}" >/dev/null 2>&1; then
      echo "${candidate}"
      return
    fi
  done
  echo "Error: could not resolve default git base revision." >&2
  exit 1
}

collect_changed_files() {
  local vcs="$1"
  local base_rev="$2"
  local target_rev="$3"

  if [[ "${vcs}" == "jj" ]]; then
    local from_revset="fork_point((${target_rev}) | (${base_rev}))"
    if ! jj diff --name-only --from "${from_revset}" --to "${target_rev}" 2>/dev/null; then
      jj diff --name-only --from "${base_rev}" --to "${target_rev}"
    fi
    return
  fi

  local diff_base="${base_rev}"
  local mb
  if mb="$(git merge-base "${base_rev}" "${target_rev:-HEAD}" 2>/dev/null)" && [[ -n "${mb}" ]]; then
    diff_base="${mb}"
  fi
  if [[ -n "${target_rev}" ]]; then
    git diff --name-only "${diff_base}" "${target_rev}"
  else
    git diff --name-only "${diff_base}"
  fi
}

run_step() {
  echo "==> [pre-push] $*"
  "$@"
}

main() {
  cd "${REPO_ROOT}"
  local vcs
  vcs="$(resolve_vcs)"

  local base_rev="${REQUESTED_BASE}"
  local target_rev="${REQUESTED_TARGET}"
  if [[ "${vcs}" == "jj" ]]; then
    base_rev="${base_rev:-trunk()}"
    target_rev="${target_rev:-@}"
  else
    if [[ -z "${base_rev}" ]]; then
      base_rev="$(resolve_default_git_base)"
    fi
  fi

  local changed_files
  changed_files="$(collect_changed_files "${vcs}" "${base_rev}" "${target_rev}")"
  if [[ -z "${changed_files}" ]]; then
    echo "pre-push: no changed files relative to ${base_rev}; skipping."
    return
  fi

  local has_md="false"
  local has_misc="false"
  local has_proto="false"
  local has_go="false"
  local has_web="false"

  local path
  while IFS= read -r path; do
    [[ -z "${path}" ]] && continue

    case "${path}" in
      *.md|.markdownlint-cli2.yaml)
        has_md="true"
        ;;
    esac

    case "${path}" in
      .cloudbuild/*|.github/*|.vscode/*)
        has_misc="true"
        ;;
    esac

    case "${path}" in
      proto/*|buf.yaml|buf.gen.yaml)
        has_proto="true"
        has_go="true"
        has_web="true"
        ;;
    esac

    case "${path}" in
      web/*)
        has_web="true"
        ;;
      go.mod|go.sum|.golangci.yaml|scripts/backend-codegen/*|scripts/codequality/*|scripts/gotestutil/*|*.go)
        has_go="true"
        ;;
    esac

    case "${path}" in
      .node-version|scripts/frontend-codegen/*|scripts/fontlist-gen/*|scripts/msdf-generator/*|scripts/generate-angular-json.sh|scripts/generate-version.sh)
        has_web="true"
        ;;
    esac
  done <<< "${changed_files}"

  local -a cq_scope_args=("CODEQUALITY_VCS=${vcs}" "CODEQUALITY_BASE=${base_rev}")
  if [[ -n "${target_rev}" ]]; then
    cq_scope_args+=("CODEQUALITY_TARGET=${target_rev}")
  fi

  echo "=================================================================="
  echo "KHI Pre-Push CI-Parity Gate (VCS: ${vcs}, Base: ${base_rev}${target_rev:+, Target: ${target_rev}})"
  echo "Scopes -> go:${has_go} web:${has_web} proto:${has_proto} md:${has_md} misc:${has_misc}"
  echo "=================================================================="

  # Phase 1: Fast static, format, and lint checks.
  run_step "${MAKE_BIN}" check-licenses

  if [[ "${has_md}" == "true" ]]; then
    run_step "${MAKE_BIN}" lint-md
  fi

  if [[ "${has_misc}" == "true" ]]; then
    run_step "${MAKE_BIN}" check-format-misc
  fi

  if [[ "${has_proto}" == "true" ]]; then
    run_step "${MAKE_BIN}" lint-proto
    run_step "${MAKE_BIN}" build-proto
  fi

  if [[ "${has_go}" == "true" ]]; then
    run_step "${MAKE_BIN}" lint-go
    run_step "${MAKE_BIN}" codequality-fast "${cq_scope_args[@]}"
  fi

  if [[ "${has_web}" == "true" ]]; then
    run_step "${MAKE_BIN}" check-format-web
    run_step "${MAKE_BIN}" lint-web
  fi

  # Phase 2: Build and unit test parity.
  if [[ "${has_web}" == "true" ]]; then
    run_step "${MAKE_BIN}" build-web
    run_step "${MAKE_BIN}" build-web-dev
    run_step "${MAKE_BIN}" build-storybook
    run_step "${MAKE_BIN}" test-web
  fi

  if [[ "${has_go}" == "true" ]]; then
    KHI_SKIP_CLOUD_LOGGING="${KHI_SKIP_CLOUD_LOGGING:-true}" run_step "${MAKE_BIN}" test-go
    run_step "${MAKE_BIN}" build-go-binaries
  fi

  # Phase 3: Cloud Build differential codequality parity (unlimited mutants).
  local -a cq_args=(
    "CODEQUALITY_MAX=${CQ_MAX}"
    "CODEQUALITY_TIMEOUT=${CQ_TIMEOUT}"
    "${cq_scope_args[@]}"
  )
  if [[ "${has_go}" == "true" && "${has_web}" == "true" ]]; then
    run_step "${MAKE_BIN}" codequality "${cq_args[@]}"
  elif [[ "${has_go}" == "true" ]]; then
    run_step "${MAKE_BIN}" codequality-go "${cq_args[@]}"
  elif [[ "${has_web}" == "true" ]]; then
    run_step "${MAKE_BIN}" codequality-web "${cq_args[@]}"
  fi
}

main "$@"
