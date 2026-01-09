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


mkdir -p vendor
cd vendor
git clone --recursive https://github.com/google/woff2.git
cd woff2
make clean all
cd ../..

./vendor/woff2/woff2_decompress ./node_modules/@fontsource/roboto/files/roboto-latin-700-normal.woff2
./vendor/woff2/woff2_decompress ./node_modules/material-symbols/material-symbols-outlined.woff2