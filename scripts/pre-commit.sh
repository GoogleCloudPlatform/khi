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


# Runs fast, diff-scoped auto-formatting and static lint checks on active
# working-copy changes (Jujutsu @ or Git working tree / index).
#
# Required tools: bash, make, and jj or git.

{
set -euo pipefail

readonly REPO_ROOT="${KHI_REPO_ROOT:-$(cd "$(dirname "$0")/.." && pwd)}"
readonly MAKE_BIN="${MAKE_BIN:-make}"
readonly VCS_MODE="${VCS:-auto}"

resolve_vcs() {
  case "${VCS_MODE}" in
    jj|git)
      echo "${VCS_MODE}"
      return
      ;;
    auto)
      if [[ -n "${GIT_INDEX_FILE:-}" ]]; then
        echo "git"
        return
      fi
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

is_license_candidate() {
  local path="$1"
  [[ -f "${REPO_ROOT}/${path}" ]] || return 1
  case "${path}" in
    *.md|*.json|*.png|*.jpg|*.jpeg|*.gif|*.svg|*.woff|*.woff2|*.ttf|*.ico|*.khi|*.txt|*.golden*|LICENSE|VERSION|.gitignore|.prettierignore|.dockerignore|.gitattributes|.node-version|.root)
      return 1
      ;;
    *)
      return 0
      ;;
  esac
}

is_target_go_source() {
  local path="$1"
  case "${path}" in
    web/*|pkg/generated/*|*/zzz_*.go|zzz_*.go)
      return 1
      ;;
    *.go)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

dir_has_go_files() {
  local rel_dir="$1"
  local abs_dir="${REPO_ROOT}/${rel_dir}"
  [[ -d "${abs_dir}" ]] || return 1
  compgen -G "${abs_dir}/*.go" >/dev/null
}

join_unique_csv() {
  if [[ $# -eq 0 ]]; then
    echo "./..."
    return
  fi
  printf '%s\n' "$@" | LC_ALL=C sort -u | paste -sd ',' -
}

run_step() {
  echo "==> [pre-commit] $*"
  "$@"
}

main() {
  cd "${REPO_ROOT}"
  local vcs
  vcs="$(resolve_vcs)"

  local changed_files=""
  local re_stage="false"

  if [[ "${vcs}" == "jj" ]]; then
    changed_files="$(jj diff --name-only)"
    if [[ -z "${changed_files}" ]]; then
      echo "Error: working copy (@) has no changes. Run 'make pre-commit' before 'jj new', run 'jj edit @-' to edit the parent commit, or run 'make pre-push' to verify the branch." >&2
      exit 1
    fi
  else
    local staged_files
    staged_files="$(git diff --cached --name-only --diff-filter=ACMR)"
    local unstaged_files
    unstaged_files="$( { git diff --name-only; git ls-files --others --exclude-standard; } | LC_ALL=C sort -u )"
    if [[ -n "${staged_files}" && ( -z "${unstaged_files}" || -n "${GIT_INDEX_FILE:-}" ) ]]; then
      changed_files="${staged_files}"
      re_stage="true"
    else
      changed_files="$( { printf '%s\n' "${staged_files}" "${unstaged_files}"; } | sed '/^$/d' | LC_ALL=C sort -u )"
    fi
    if [[ -z "${changed_files}" ]]; then
      echo "pre-commit: working tree has no changes."
      return
    fi
  fi

  local has_license="false"
  local has_proto="false"
  local has_go="false"
  local go_all_pkgs="false"
  local has_web="false"
  local has_misc="false"
  local has_md="false"
  local -a go_pkg_dirs=()
  local -a existing_files=()

  local path
  while IFS= read -r path; do
    [[ -z "${path}" ]] && continue
    if [[ -e "${REPO_ROOT}/${path}" ]]; then
      existing_files+=("${path}")
    fi
    if is_license_candidate "${path}"; then
      has_license="true"
    fi

    case "${path}" in
      proto/*|buf.yaml|buf.gen.yaml)
        has_proto="true"
        has_go="true"
        go_all_pkgs="true"
        has_web="true"
        ;;
    esac

    case "${path}" in
      go.mod|go.sum|.golangci.yaml|scripts/backend-codegen/*)
        has_go="true"
        go_all_pkgs="true"
        ;;
      *.go)
        if is_target_go_source "${path}"; then
          has_go="true"
          local rel_dir
          rel_dir="$(dirname "${path}")"
          if dir_has_go_files "${rel_dir}"; then
            if [[ "${rel_dir}" == "." ]]; then
              go_pkg_dirs+=(".")
            else
              go_pkg_dirs+=("./${rel_dir}")
            fi
          else
            go_all_pkgs="true"
          fi
        fi
        ;;
    esac

    case "${path}" in
      web/*|scripts/frontend-codegen/*|scripts/generate-angular-json.sh|scripts/generate-version.sh)
        has_web="true"
        ;;
    esac

    case "${path}" in
      .cloudbuild/*|.github/*|.vscode/*)
        has_misc="true"
        ;;
    esac

    case "${path}" in
      *.md|.markdownlint-cli2.yaml)
        has_md="true"
        ;;
    esac
  done <<< "${changed_files}"

  if [[ "${has_license}" == "true" ]]; then
    run_step "${MAKE_BIN}" add-licenses
  fi

  if [[ "${has_proto}" == "true" ]]; then
    run_step "${MAKE_BIN}" format-proto
    run_step "${MAKE_BIN}" build-proto
  fi

  if [[ "${has_go}" == "true" ]]; then
    local lint_pkgs="./..."
    if [[ "${go_all_pkgs}" != "true" && ${#go_pkg_dirs[@]} -gt 0 ]]; then
      lint_pkgs="$(join_unique_csv "${go_pkg_dirs[@]}")"
    fi
    run_step "${MAKE_BIN}" format-go "LINT_PKGS=${lint_pkgs}"
  fi

  if [[ "${has_web}" == "true" ]]; then
    run_step "${MAKE_BIN}" format-web
  fi

  if [[ "${has_misc}" == "true" ]]; then
    run_step "${MAKE_BIN}" format-misc
  fi

  if [[ "${has_md}" == "true" ]]; then
    run_step "${MAKE_BIN}" format-md
  fi

  run_step "${MAKE_BIN}" qa-fast-must

  if [[ "${re_stage}" == "true" && ${#existing_files[@]} -gt 0 ]]; then
    git add -- "${existing_files[@]}"
  fi
}

main "$@"
exit 0
}
