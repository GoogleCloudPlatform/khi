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


# Runs Storybook visual regression tests (VRT) in Cloud Build and uploads
# snapshots and interactive diff reports to Google Cloud Storage.
#
# Required tools: git, go, make, node, npm, chromium (preinstalled in builder:latest).

set -euo pipefail

readonly REPO_URL="${REPO_URL:-https://github.com/GoogleCloudPlatform/khi.git}"
readonly BASE_BRANCH="${BASE_BRANCH:-main}"
readonly VRT_MODE="${VRT_MODE:-compare}"
readonly VRT_BUCKET="${VRT_BUCKET:-}"
readonly VRT_ACCESS_MODE="${VRT_ACCESS_MODE:-console}"
readonly VRT_WORKERS="${VRT_WORKERS:-0}"
readonly VRT_FAIL_ON_DIFF="${VRT_FAIL_ON_DIFF:-false}"
readonly VRT_FORCE_FULL="${VRT_FORCE_FULL:-false}"
readonly SKIP_FETCH="${SKIP_FETCH:-false}"

TMP_DIR=""
WORKTREE_DIR=""

cleanup() {
  if [[ -n "${WORKTREE_DIR}" && -d "${WORKTREE_DIR}" ]]; then
    git worktree remove --force "${WORKTREE_DIR}" >/dev/null 2>&1 || rm -rf "${WORKTREE_DIR}"
  fi
  if [[ -n "${TMP_DIR}" && -d "${TMP_DIR}" ]]; then
    rm -rf "${TMP_DIR}"
  fi
}
trap cleanup EXIT

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

build_cloud_build_log_url() {
  if [[ -z "${PROJECT_ID:-}" || -z "${BUILD_ID:-}" ]]; then
    echo ""
    return
  fi
  local region="${LOCATION:-asia-northeast1}"
  echo "https://console.cloud.google.com/cloud-build/builds;region=${region}/${BUILD_ID}?project=${PROJECT_ID}"
}

has_frontend_changes() {
  local merge_base="$1"
  if [[ "${VRT_FORCE_FULL}" == "true" ]]; then
    return 0
  fi

  local changed_files
  changed_files="$(git diff --name-only "${merge_base}")"
  local path
  while IFS= read -r path; do
    [[ -z "${path}" ]] && continue
    case "${path}" in
      web/*|scripts/msdf-generator/*|scripts/fontlist-gen/*|pkg/task/inspection/*|pkg/model/enum/*|proto/*|scripts/builder/*|scripts/vrt/*|scripts/run-vrt-ci.sh|.cloudbuild/github-vrt*.yaml)
        return 0
        ;;
    esac
  done <<< "${changed_files}"
  return 1
}

build_storybook_in_dir() {
  local dir="$1"
  echo "Building Storybook in ${dir}..."
  (
    cd "${dir}"
    (cd ./web && npm ci --prefer-offline)
    touch scripts/make/build-proto.done
    make generate-frontend
    make build-storybook
  )
}

run_baseline_mode() {
  local vrt_bin="$1"
  local fingerprint="$2"
  local head_sha="$3"

  echo "=================================================================="
  echo "KHI Cloud Build Frontend VRT Baseline Warmup"
  echo "Commit: ${head_sha:0:12} | Fingerprint: ${fingerprint} | Bucket: ${VRT_BUCKET:-<none>}"
  echo "=================================================================="

  if [[ -n "${VRT_BUCKET}" ]]; then
    if "${vrt_bin}" --mode=has-snapshot \
      --gcs-bucket="${VRT_BUCKET}" \
      --fingerprint="${fingerprint}" \
      --commit-sha="${head_sha}"; then
      echo "Snapshot already cached in GCS for ${head_sha:0:12} (${fingerprint}); skipping rebuild."
      return 0
    fi
  fi

  build_storybook_in_dir "$(pwd)"
  local snap_dir="${TMP_DIR}/head-snapshot"

  local -a upload_args=()
  if [[ -n "${VRT_BUCKET}" ]]; then
    upload_args=(
      "--gcs-bucket=${VRT_BUCKET}"
      "--gcs-access-mode=${VRT_ACCESS_MODE}"
      "--upload-head-snapshot"
    )
  fi

  "${vrt_bin}" --mode=capture \
    --storybook-dir="web/storybook-static" \
    --out-dir="${snap_dir}" \
    --workers="${VRT_WORKERS}" \
    --fingerprint="${fingerprint}" \
    --commit-sha="${head_sha}" \
    "${upload_args[@]}"
}

run_compare_mode() {
  local vrt_bin="$1"
  local fingerprint="$2"

  configure_git_and_fetch_base

  local merge_base
  merge_base="$(git merge-base "origin/${BASE_BRANCH}" HEAD)"
  local head_sha
  head_sha="${COMMIT_SHA:-$(git rev-parse HEAD)}"
  local run_id
  run_id="${BUILD_ID:-$(date -u +%Y%m%d-%H%M%S)}"
  local build_url
  build_url="$(build_cloud_build_log_url)"

  local emit_envelope="${EMIT_COMMENT_ENVELOPE:-false}"
  if [[ -n "${BUILD_ID:-}" ]]; then
    emit_envelope="true"
  fi

  echo "=================================================================="
  echo "KHI Cloud Build Frontend Visual Regression Test"
  echo "PR: #${PR_NUMBER:-local} | Base: origin/${BASE_BRANCH} (${merge_base:0:12}) -> HEAD (${head_sha:0:12})"
  echo "Fingerprint: ${fingerprint} | Bucket: ${VRT_BUCKET:-<local>} (${VRT_ACCESS_MODE}) | FailOnDiff: ${VRT_FAIL_ON_DIFF}"
  echo "=================================================================="

  if ! has_frontend_changes "${merge_base}"; then
    echo "No frontend-affecting files changed; skipping Storybook VRT."
    "${vrt_bin}" --mode=skip-comment \
      --commit-sha="${head_sha}" \
      --base-sha="${merge_base}" \
      --base-branch="${BASE_BRANCH}" \
      --build-url="${build_url}" \
      --comment-envelope="${emit_envelope}" \
      --skip-reason="No frontend-affecting files changed compared to \`origin/${BASE_BRANCH}\`."
    return 0
  fi

  local base_dir="${TMP_DIR}/base-snapshot"
  local head_dir="${TMP_DIR}/head-snapshot"
  local report_dir="${TMP_DIR}/report"
  local baseline_cache_hit="false"

  if [[ -n "${VRT_BUCKET}" ]]; then
    local dl_status=0
    "${vrt_bin}" --mode=download-snapshot \
      --gcs-bucket="${VRT_BUCKET}" \
      --fingerprint="${fingerprint}" \
      --commit-sha="${merge_base}" \
      --out-dir="${base_dir}" \
      --workers="${VRT_WORKERS}" || dl_status=$?
    if [[ "${dl_status}" -eq 0 ]]; then
      echo "Baseline cache HIT in GCS for ${merge_base:0:12} (${fingerprint})."
      baseline_cache_hit="true"
    elif [[ "${dl_status}" -eq 2 ]]; then
      echo "Baseline cache MISS in GCS for ${merge_base:0:12} (${fingerprint}); building baseline via git worktree..."
    else
      echo "Failed to query GCS baseline cache (exit ${dl_status})." >&2
      return "${dl_status}"
    fi
  fi

  if [[ "${baseline_cache_hit}" == "false" ]]; then
    WORKTREE_DIR="${TMP_DIR}/base-worktree"
    git worktree add --detach "${WORKTREE_DIR}" "${merge_base}"
    build_storybook_in_dir "${WORKTREE_DIR}"

    local -a base_upload_args=()
    if [[ -n "${VRT_BUCKET}" ]]; then
      base_upload_args=(
        "--gcs-bucket=${VRT_BUCKET}"
        "--gcs-access-mode=${VRT_ACCESS_MODE}"
        "--upload-head-snapshot"
      )
    fi

    # Continue even if an older baseline commit has a broken story so HEAD comparison still reports accurately.
    "${vrt_bin}" --mode=capture \
      --storybook-dir="${WORKTREE_DIR}/web/storybook-static" \
      --out-dir="${base_dir}" \
      --workers="${VRT_WORKERS}" \
      --fingerprint="${fingerprint}" \
      --commit-sha="${merge_base}" \
      "${base_upload_args[@]}" || true

    git worktree remove --force "${WORKTREE_DIR}" >/dev/null 2>&1 || rm -rf "${WORKTREE_DIR}"
    WORKTREE_DIR=""
  fi

  build_storybook_in_dir "$(pwd)"

  local -a gcs_args=()
  if [[ -n "${VRT_BUCKET}" ]]; then
    gcs_args=(
      "--gcs-bucket=${VRT_BUCKET}"
      "--gcs-access-mode=${VRT_ACCESS_MODE}"
      "--upload-head-snapshot"
      "--upload-report"
    )
  fi

  "${vrt_bin}" --mode=compare \
    --storybook-dir="web/storybook-static" \
    --base-dir="${base_dir}" \
    --head-dir="${head_dir}" \
    --report-dir="${report_dir}" \
    --workers="${VRT_WORKERS}" \
    --fingerprint="${fingerprint}" \
    --commit-sha="${head_sha}" \
    --base-sha="${merge_base}" \
    --base-branch="${BASE_BRANCH}" \
    --run-id="${run_id}" \
    --build-url="${build_url}" \
    --baseline-cache-hit="${baseline_cache_hit}" \
    --comment-envelope="${emit_envelope}" \
    --fail-on-diff="${VRT_FAIL_ON_DIFF}" \
    "${gcs_args[@]}"
}

main() {
  TMP_DIR="$(mktemp -d)"
  local vrt_bin="${TMP_DIR}/khi-vrt"

  echo "Building VRT runner binary..."
  go build -o "${vrt_bin}" ./scripts/vrt

  local fingerprint
  fingerprint="$("${vrt_bin}" --mode=fingerprint)"

  if [[ "${VRT_MODE}" == "baseline" ]]; then
    local head_sha="${COMMIT_SHA:-$(git rev-parse HEAD)}"
    run_baseline_mode "${vrt_bin}" "${fingerprint}" "${head_sha}"
  else
    run_compare_mode "${vrt_bin}" "${fingerprint}"
  fi
}

main "$@"
