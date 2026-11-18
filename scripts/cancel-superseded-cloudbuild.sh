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

# Cancels superseded Cloud Build PR runs in both directions:
#   1. If this build's commit is already older than the PR head, the PR is
#      closed/merged, or a newer build exists for the same trigger & PR, this
#      build cancels itself immediately to release the worker slot.
#   2. Otherwise, this build cancels any older QUEUED or WORKING builds for
#      the same trigger & PR.
#
# Required tools: bash, gcloud, git (optional: curl).

set -euo pipefail
export LC_ALL=C

readonly PROJECT_ID="${PROJECT_ID:-}"
readonly BUILD_ID="${BUILD_ID:-}"
readonly REGION="${LOCATION:-asia-northeast1}"
readonly PR_NUMBER="${PR_NUMBER:-}"
readonly COMMIT_SHA="${COMMIT_SHA:-}"
readonly REPO_URL="${REPO_URL:-https://github.com/GoogleCloudPlatform/khi.git}"
readonly GITHUB_REPO="${GITHUB_REPO:-GoogleCloudPlatform/khi}"
readonly SKIP_GITHUB_API="${SKIP_GITHUB_API:-false}"

cancel_build() {
  local target_id="$1"
  gcloud builds cancel "${target_id}" \
    --region="${REGION}" \
    --project="${PROJECT_ID}" \
    --quiet || true
}

cancel_self_and_exit() {
  local reason="$1"
  echo "cancel-superseded: ${reason}; cancelling current build ${BUILD_ID}."
  cancel_build "${BUILD_ID}"
  exit 1
}

# Normalizes a UTC RFC 3339 timestamp (Z or +00:00) to fixed-width nanosecond
# precision so ASCII lexicographical comparison matches chronological order.
normalize_utc_ts() {
  local raw="$1"
  if [[ ! "${raw}" =~ ^([0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2})(\.([0-9]+))?(Z|\+00:00)$ ]]; then
    return 1
  fi
  local base="${BASH_REMATCH[1]}"
  local frac="${BASH_REMATCH[3]}000000000"
  printf '%s.%sZ' "${base}" "${frac:0:9}"
}

check_git_pr_head() {
  if [[ -z "${COMMIT_SHA}" ]] || ! command -v git >/dev/null 2>&1; then
    return
  fi
  local ls_out
  if ! ls_out="$(git ls-remote "${REPO_URL}" "refs/pull/${PR_NUMBER}/head" 2>/dev/null)"; then
    return
  fi
  local latest_sha="${ls_out%%[[:space:]]*}"
  if [[ -n "${latest_sha}" && "${latest_sha}" != "${COMMIT_SHA}" ]]; then
    cancel_self_and_exit "commit ${COMMIT_SHA:0:12} is superseded by PR #${PR_NUMBER} head ${latest_sha:0:12}"
  fi
}

check_github_pr_open() {
  if [[ "${SKIP_GITHUB_API}" == "true" ]] || ! command -v curl >/dev/null 2>&1; then
    return
  fi
  local pr_json
  if ! pr_json="$(curl -fsS --max-time 5 \
    -H "Accept: application/vnd.github+json" \
    "https://api.github.com/repos/${GITHUB_REPO}/pulls/${PR_NUMBER}" 2>/dev/null)"; then
    return
  fi
  # In GitHub's PR REST payload, top-level "state" precedes title/body and any nested objects.
  local pr_state=""
  if [[ "${pr_json}" =~ \"state\"[[:space:]]*:[[:space:]]*\"([^\"]+)\" ]]; then
    pr_state="${BASH_REMATCH[1]}"
  fi
  if [[ -n "${pr_state}" && "${pr_state}" != "open" ]]; then
    cancel_self_and_exit "PR #${PR_NUMBER} is already ${pr_state}"
  fi
}

reconcile_ongoing_builds() {
  local current_info
  if ! current_info="$(gcloud builds describe "${BUILD_ID}" \
    --region="${REGION}" \
    --project="${PROJECT_ID}" \
    --format="value[separator=|](buildTriggerId,createTime.date(format='%Y-%m-%dT%H:%M:%S.%fZ',tz=UTC))" 2>/dev/null)"; then
    echo "cancel-superseded: unable to describe build ${BUILD_ID}; continuing."
    return
  fi

  local trigger_id=""
  local current_create_time=""
  IFS='|' read -r trigger_id current_create_time <<< "${current_info}"
  if [[ -z "${trigger_id}" || -z "${current_create_time}" ]]; then
    return
  fi

  local cur_norm_ts
  if ! cur_norm_ts="$(normalize_utc_ts "${current_create_time}")"; then
    echo "cancel-superseded: unable to parse build createTime '${current_create_time}'; continuing."
    return
  fi

  local ongoing_rows
  if ! ongoing_rows="$(gcloud builds list \
    --ongoing \
    --region="${REGION}" \
    --project="${PROJECT_ID}" \
    --filter="buildTriggerId=${trigger_id} AND substitutions._PR_NUMBER=${PR_NUMBER}" \
    --format="value[separator=|](createTime.date(format='%Y-%m-%dT%H:%M:%S.%fZ',tz=UTC),id)" 2>/dev/null)"; then
    echo "cancel-superseded: unable to list ongoing builds; continuing."
    return
  fi

  local newer_id=""
  local -a older_ids=()
  local row_ts row_id norm_ts
  while IFS='|' read -r row_ts row_id; do
    if [[ -z "${row_id}" || "${row_id}" == "${BUILD_ID}" || -z "${row_ts}" ]]; then
      continue
    fi
    if ! norm_ts="$(normalize_utc_ts "${row_ts}")"; then
      continue
    fi
    if [[ "${norm_ts}" > "${cur_norm_ts}" || ( "${norm_ts}" == "${cur_norm_ts}" && "${row_id}" > "${BUILD_ID}" ) ]]; then
      newer_id="${row_id}"
      break
    else
      older_ids+=("${row_id}")
    fi
  done <<< "${ongoing_rows}"

  if [[ -n "${newer_id}" ]]; then
    cancel_self_and_exit "newer ongoing build ${newer_id} already exists for PR #${PR_NUMBER}"
  fi

  if [[ "${#older_ids[@]}" -eq 0 ]]; then
    return
  fi

  local target_id
  for target_id in "${older_ids[@]}"; do
    echo "cancel-superseded: cancelling older build ${target_id} for PR #${PR_NUMBER}..."
    cancel_build "${target_id}"
  done
}

main() {
  if [[ -z "${PROJECT_ID}" || -z "${BUILD_ID}" || -z "${PR_NUMBER}" ]]; then
    echo "cancel-superseded: non-PR or local build detected; skipping."
    return
  fi

  check_git_pr_head
  check_github_pr_open
  reconcile_ongoing_builds
}

main "$@"
