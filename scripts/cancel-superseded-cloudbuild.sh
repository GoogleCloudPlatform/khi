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
# Required tools: bash, gcloud, git, python3 (optional: curl).

set -euo pipefail

readonly PROJECT_ID="${PROJECT_ID:-}"
readonly BUILD_ID="${BUILD_ID:-}"
readonly REGION="${LOCATION:-asia-northeast1}"
readonly PR_NUMBER="${PR_NUMBER:-}"
readonly COMMIT_SHA="${COMMIT_SHA:-}"
readonly REPO_URL="${REPO_URL:-https://github.com/GoogleCloudPlatform/khi.git}"
readonly GITHUB_REPO="${GITHUB_REPO:-GoogleCloudPlatform/khi}"
readonly SKIP_GITHUB_API="${SKIP_GITHUB_API:-false}"

TMP_FILE=""
trap '[[ -n "${TMP_FILE}" ]] && rm -f "${TMP_FILE}"' EXIT

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

check_git_pr_head() {
  if [[ -z "${COMMIT_SHA}" ]]; then
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
  local pr_state
  pr_state="$(python3 -c '
import json, sys
try:
    data = json.load(sys.stdin)
    print(data.get("state", ""))
except Exception:
    pass
' <<< "${pr_json}")"
  if [[ -n "${pr_state}" && "${pr_state}" != "open" ]]; then
    cancel_self_and_exit "PR #${PR_NUMBER} is already ${pr_state}"
  fi
}

reconcile_ongoing_builds() {
  local current_json
  if ! current_json="$(gcloud builds describe "${BUILD_ID}" \
    --region="${REGION}" \
    --project="${PROJECT_ID}" \
    --format=json 2>/dev/null)"; then
    echo "cancel-superseded: unable to describe build ${BUILD_ID}; continuing."
    return
  fi

  local trigger_id
  local current_create_time
  read -r trigger_id current_create_time < <(python3 -c '
import json, sys
data = json.load(sys.stdin)
print(data.get("buildTriggerId", ""), data.get("createTime", ""))
' <<< "${current_json}")

  if [[ -z "${trigger_id}" || -z "${current_create_time}" ]]; then
    return
  fi

  local ongoing_json
  if ! ongoing_json="$(gcloud builds list \
    --ongoing \
    --region="${REGION}" \
    --project="${PROJECT_ID}" \
    --filter="buildTriggerId=${trigger_id} AND substitutions._PR_NUMBER=${PR_NUMBER}" \
    --format=json 2>/dev/null)"; then
    echo "cancel-superseded: unable to list ongoing builds; continuing."
    return
  fi

  TMP_FILE="$(mktemp)"
  printf '%s\n' "${ongoing_json}" > "${TMP_FILE}"

  local plan
  plan="$(python3 - "${BUILD_ID}" "${current_create_time}" "${TMP_FILE}" <<'EOF'
import datetime
import json
import sys

cur_id = sys.argv[1]
cur_ts_raw = sys.argv[2]
ongoing_path = sys.argv[3]

def parse_ts(raw):
    return datetime.datetime.fromisoformat(raw.replace("Z", "+00:00"))

cur_key = (parse_ts(cur_ts_raw), cur_id)
with open(ongoing_path, encoding="utf-8") as fh:
    builds = json.load(fh) or []

newer = []
older = []
for b in builds:
    bid = b.get("id", "")
    ts_raw = b.get("createTime", "")
    if not bid or bid == cur_id or not ts_raw:
        continue
    key = (parse_ts(ts_raw), bid)
    if key > cur_key:
        newer.append(bid)
    else:
        older.append(bid)

if newer:
    print("SELF_STALE " + newer[0])
else:
    for bid in older:
        print("CANCEL_OLD " + bid)
EOF
)"

  if [[ -z "${plan}" ]]; then
    return
  fi

  local action target_id
  while read -r action target_id; do
    [[ -z "${action}" ]] && continue
    if [[ "${action}" == "SELF_STALE" ]]; then
      cancel_self_and_exit "newer ongoing build ${target_id} already exists for PR #${PR_NUMBER}"
    elif [[ "${action}" == "CANCEL_OLD" ]]; then
      echo "cancel-superseded: cancelling older build ${target_id} for PR #${PR_NUMBER}..."
      cancel_build "${target_id}"
    fi
  done <<< "${plan}"
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
