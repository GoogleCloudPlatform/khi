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

# Extracts the Base64 codequality report from a completed Google Cloud Build
# Check Run and posts or updates a single sticky comment on the matching PR.
#
# Required tools: gh, jq, node.

set -euo pipefail

readonly COMMENT_MARKER="<!-- khi-codequality-report -->"
readonly ENVELOPE_BEGIN="KHI_CODEQUALITY_B64_BEGIN"
readonly ENVELOPE_END="KHI_CODEQUALITY_B64_END"

readonly GITHUB_REPO="${GITHUB_REPO:?GITHUB_REPO is required}"
readonly CHECK_RUN_ID="${CHECK_RUN_ID:?CHECK_RUN_ID is required}"
readonly HEAD_SHA="${HEAD_SHA:?HEAD_SHA is required}"
readonly CONCLUSION="${CONCLUSION:-unknown}"
readonly DETAILS_URL="${DETAILS_URL:-}"

find_open_pr_numbers_for_head() {
  gh api "repos/${GITHUB_REPO}/commits/${HEAD_SHA}/pulls" \
    --jq ".[] | select(.state == \"open\" and .head.sha == \"${HEAD_SHA}\") | .number"
}

extract_envelope_from_log() {
  local log_file="$1"
  local out_file="$2"
  node - "${log_file}" "${out_file}" "${ENVELOPE_BEGIN}" "${ENVELOPE_END}" <<'EOF'
const fs = require('fs');
const [logPath, outPath, beginTag, endTag] = process.argv.slice(2);
const text = fs.readFileSync(logPath, 'utf8');
const beginIdx = text.lastIndexOf(beginTag);
if (beginIdx < 0) {
  process.exit(1);
}
const rest = text.slice(beginIdx + beginTag.length);
const endIdx = rest.indexOf(endTag);
if (endIdx < 0) {
  process.exit(1);
}
const b64 = rest.slice(0, endIdx).replace(/\s+/g, '');
const decoded = Buffer.from(b64, 'base64').toString('utf8');
if (!decoded.trim()) {
  process.exit(1);
}
fs.writeFileSync(outPath, decoded, 'utf8');
EOF
}

write_fallback_body() {
  local out_file="$1"
  local short_sha="${HEAD_SHA:0:12}"
  local verdict_icon="❌"
  if [[ "${CONCLUSION}" == "success" ]]; then
    verdict_icon="✅"
  elif [[ "${CONCLUSION}" == "cancelled" ]]; then
    verdict_icon="⚪"
  fi

  {
    printf '%s\n' "${COMMENT_MARKER}"
    printf '## Code Quality Report — %s Cloud Build `%s`\n\n' "${verdict_icon}" "${CONCLUSION}"
    if [[ "${CONCLUSION}" == "success" ]]; then
      printf 'Cloud Build completed without an embedded report envelope.\n'
    else
      printf 'Cloud Build exited before `codequality` finished emitting its report (for example, compilation or test setup failed).\n'
    fi
    printf '\n---\n_Commit: `%s`' "${short_sha}"
    if [[ -n "${DETAILS_URL}" ]]; then
      printf ' · [Cloud Build Logs](%s)' "${DETAILS_URL}"
    fi
    printf '_\n'
  } > "${out_file}"
}

upsert_sticky_comment() {
  local pr_number="$1"
  local body_file="$2"

  local existing_id
  existing_id="$(
    gh api --paginate "repos/${GITHUB_REPO}/issues/${pr_number}/comments" \
      --jq ".[] | select(.body | contains(\"${COMMENT_MARKER}\")) | .id" \
      | head -n 1
  )"

  if [[ -n "${existing_id}" ]]; then
    echo "Updating existing sticky comment #${existing_id} on PR #${pr_number}..."
    jq -Rs '{body: .}' < "${body_file}" \
      | gh api --method PATCH "repos/${GITHUB_REPO}/issues/comments/${existing_id}" --input - >/dev/null
  else
    echo "Creating sticky comment on PR #${pr_number}..."
    jq -Rs '{body: .}' < "${body_file}" \
      | gh api --method POST "repos/${GITHUB_REPO}/issues/${pr_number}/comments" --input - >/dev/null
  fi
}

TMP_DIR=""
trap '[[ -n "${TMP_DIR}" ]] && rm -rf "${TMP_DIR}"' EXIT

main() {
  local pr_numbers
  pr_numbers="$(find_open_pr_numbers_for_head)"
  if [[ -z "${pr_numbers}" ]]; then
    echo "No open pull request currently points at HEAD ${HEAD_SHA}; skipping comment."
    return
  fi

  TMP_DIR="$(mktemp -d)"

  local log_file="${TMP_DIR}/check_run_log.txt"
  local body_file="${TMP_DIR}/comment_body.md"

  gh api "repos/${GITHUB_REPO}/check-runs/${CHECK_RUN_ID}" --jq '.output.text // ""' > "${log_file}"

  if ! extract_envelope_from_log "${log_file}" "${body_file}"; then
    echo "No embedded report envelope found in Check Run #${CHECK_RUN_ID}; using fallback summary."
    write_fallback_body "${body_file}"
  fi

  local pr_num
  while IFS= read -r pr_num; do
    [[ -z "${pr_num}" ]] && continue
    upsert_sticky_comment "${pr_num}" "${body_file}"
  done <<< "${pr_numbers}"
}

main "$@"
