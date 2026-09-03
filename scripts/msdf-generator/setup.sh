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


set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${SCRIPT_DIR}"

WOFF2_COMMIT="1c69169e9e1811dccd6c54c532fedda300233968"

if [ ! -f "./vendor/woff2/woff2_decompress" ]; then
  mkdir -p vendor
  rm -rf vendor/woff2
  git clone https://github.com/google/woff2.git vendor/woff2
  (
    cd vendor/woff2
    git checkout "${WOFF2_COMMIT}"
    git submodule update --init --recursive
    make clean all
  )
fi

./vendor/woff2/woff2_decompress ./node_modules/@fontsource/roboto/files/roboto-latin-700-normal.woff2
./vendor/woff2/woff2_decompress ./node_modules/material-symbols/material-symbols-outlined.woff2