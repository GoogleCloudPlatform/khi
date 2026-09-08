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

# ==============================================================================
# Generate .agents/plugins.json by merging all .agents/plugins-*.json files
# ==============================================================================
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
AGENTS_DIR="${REPO_ROOT}/.agents"
OUT_JSON="${AGENTS_DIR}/plugins.json"

mkdir -p "${AGENTS_DIR}"

# Initialize plugins-oss.json if not present
if [[ ! -f "${AGENTS_DIR}/plugins-oss.json" ]]; then
  echo '{"entries":[]}' > "${AGENTS_DIR}/plugins-oss.json"
fi

# Collect all plugins-*.json config files
shopt -s nullglob
CONFIG_FILES=("${AGENTS_DIR}"/plugins-*.json)
shopt -u nullglob

if [[ ${#CONFIG_FILES[@]} -eq 0 ]]; then
  echo '{"entries":[]}' > "${OUT_JSON}"
else
  # Merge entries from all matching config files
  jq -s '{entries: ([.[].entries // []] | add)}' "${CONFIG_FILES[@]}" > "${OUT_JSON}"
fi

echo "Generated ${OUT_JSON} from ${#CONFIG_FILES[@]} config file(s):"
for f in "${CONFIG_FILES[@]}"; do
  echo "  - $(basename "$f")"
done
