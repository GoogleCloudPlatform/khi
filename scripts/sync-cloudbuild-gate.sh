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


# Aggregates Google Cloud Build Check Runs on a PR commit into a single
# GitHub Actions Check Run ("cloudbuild-gate") so GitHub Rulesets with Merge
# Queue enabled can require Cloud Build PR checks without stalling merge_group.
#
# Required tools: gh, jq, node.

set -euo pipefail

readonly GITHUB_REPO="${GITHUB_REPO:?GITHUB_REPO is required}"
readonly HEAD_SHA="${HEAD_SHA:?HEAD_SHA is required}"
readonly GATE_CHECK_NAME="${GATE_CHECK_NAME:-cloudbuild-gate}"
readonly REQUIRED_CLOUDBUILD_PREFIXES="${REQUIRED_CLOUDBUILD_PREFIXES:-github-codequality,github-deploy-ondemand}"

TMP_DIR=""
trap '[[ -n "${TMP_DIR}" ]] && rm -rf "${TMP_DIR}"' EXIT

fetch_commit_check_runs() {
  local out_file="$1"
  gh api --paginate "repos/${GITHUB_REPO}/commits/${HEAD_SHA}/check-runs" \
    --jq '.check_runs[]' | jq -s '.' > "${out_file}"
}

build_gate_action() {
  local check_runs_file="$1"
  local action_file="$2"
  local payload_file="$3"

  node - "${check_runs_file}" "${action_file}" "${payload_file}" \
    "${GITHUB_REPO}" "${HEAD_SHA}" "${GATE_CHECK_NAME}" "${REQUIRED_CLOUDBUILD_PREFIXES}" <<'EOF'
const fs = require('fs');

const [
  runsPath,
  actionPath,
  payloadPath,
  repo,
  headSha,
  gateName,
  prefixesCsv,
] = process.argv.slice(2);

const runs = JSON.parse(fs.readFileSync(runsPath, 'utf8'));
const prefixes = prefixesCsv
  .split(',')
  .map((s) => s.trim())
  .filter(Boolean);

function findLatestRun(predicate) {
  let latest = null;
  for (const r of runs) {
    if (!predicate(r)) continue;
    if (!latest || Number(r.id) > Number(latest.id)) {
      latest = r;
    }
  }
  return latest;
}

const rows = [];
let anyFailed = false;
let anyPending = false;

for (const prefix of prefixes) {
  const latest = findLatestRun(
    (r) =>
      r &&
      r.app &&
      r.app.slug === 'google-cloud-build' &&
      typeof r.name === 'string' &&
      r.name.startsWith(prefix),
  );

  if (!latest) {
    anyPending = true;
    rows.push({
      check: `\`${prefix}\``,
      status: '⏳ Waiting to start',
      details: '—',
    });
    continue;
  }

  const url = latest.details_url || latest.html_url || '';
  const link = url ? `[Logs](${url})` : '—';
  const displayName = `\`${latest.name}\``;

  if (latest.status !== 'completed') {
    anyPending = true;
    rows.push({
      check: displayName,
      status: `🔄 In progress (\`${latest.status}\`)`,
      details: link,
    });
    continue;
  }

  const conclusion = latest.conclusion || 'unknown';
  if (['success', 'neutral', 'skipped'].includes(conclusion)) {
    rows.push({
      check: displayName,
      status: `✅ \`${conclusion}\``,
      details: link,
    });
  } else {
    anyFailed = true;
    rows.push({
      check: displayName,
      status: `❌ \`${conclusion}\``,
      details: link,
    });
  }
}

let status = 'completed';
let conclusion = 'success';
let title = 'All required Cloud Build checks passed';

if (anyFailed) {
  status = 'completed';
  conclusion = 'failure';
  title = 'One or more required Cloud Build checks failed';
} else if (anyPending) {
  status = 'in_progress';
  conclusion = null;
  title = 'Waiting for required Cloud Build checks to complete';
}

const lines = [
  `### Cloud Build PR Gate (\`${headSha.slice(0, 12)}\`)`,
  '',
  '| Cloud Build Check | Status | Details |',
  '| :--- | :--- | :--- |',
  ...rows.map((r) => `| ${r.check} | ${r.status} | ${r.details} |`),
];
const summary = lines.join('\n') + '\n';

const existingGate = findLatestRun(
  (r) =>
    r &&
    r.name === gateName &&
    r.app &&
    r.app.slug === 'github-actions',
);

if (
  existingGate &&
  existingGate.status === 'completed' &&
  status === 'completed' &&
  existingGate.conclusion === conclusion
) {
  fs.writeFileSync(actionPath, JSON.stringify({ skip: true }), 'utf8');
  process.exit(0);
}

if (existingGate && existingGate.status !== 'completed') {
  const payload = {
    name: gateName,
    status,
    output: { title, summary },
  };
  if (conclusion) {
    payload.conclusion = conclusion;
  }
  fs.writeFileSync(
    actionPath,
    JSON.stringify({
      skip: false,
      method: 'PATCH',
      endpoint: `repos/${repo}/check-runs/${existingGate.id}`,
    }),
    'utf8',
  );
  fs.writeFileSync(payloadPath, JSON.stringify(payload), 'utf8');
  process.exit(0);
}

const payload = {
  name: gateName,
  head_sha: headSha,
  status,
  output: { title, summary },
};
if (conclusion) {
  payload.conclusion = conclusion;
}
fs.writeFileSync(
  actionPath,
  JSON.stringify({
    skip: false,
    method: 'POST',
    endpoint: `repos/${repo}/check-runs`,
  }),
  'utf8',
);
fs.writeFileSync(payloadPath, JSON.stringify(payload), 'utf8');
EOF
}

main() {
  TMP_DIR="$(mktemp -d)"
  local runs_file="${TMP_DIR}/check_runs.json"
  local action_file="${TMP_DIR}/action.json"
  local payload_file="${TMP_DIR}/payload.json"

  fetch_commit_check_runs "${runs_file}"
  build_gate_action "${runs_file}" "${action_file}" "${payload_file}"

  local skip
  skip="$(jq -r '.skip' "${action_file}")"
  if [[ "${skip}" == "true" ]]; then
    echo "Check Run '${GATE_CHECK_NAME}' on ${HEAD_SHA:0:12} is already up to date; skipping."
    return
  fi

  local method
  method="$(jq -r '.method' "${action_file}")"
  local endpoint
  endpoint="$(jq -r '.endpoint' "${action_file}")"

  echo "Syncing '${GATE_CHECK_NAME}' via ${method} ${endpoint}..."
  gh api --method "${method}" "${endpoint}" --input "${payload_file}" >/dev/null
}

main "$@"
