#!/bin/bash
# Copyright 2025 Google LLC
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

PROD_ENV_FILES=$(ls ./web/src/environments/environment.prod*.ts)
DEV_ENV_FILES=$(ls ./web/src/environments/environment.dev*.ts)

PROD_ENV_NAMES=$(echo $PROD_ENV_FILES| jq -R -s -c '[split(" ")[]|capture(".web/src/environments/environment.(?<env>.*).ts")|.env]')
DEV_ENV_NAMES=$(echo $DEV_ENV_FILES| jq -R -s -c '[split(" ")[]|capture(".web/src/environments/environment.(?<env>.*).ts")|.env]')

PROD_VERSION_CONTENT=$(echo "export const VERSION=\"$(cat VERSION)\"")
DEV_VERSION_CONTENT=$(echo "export const VERSION=\"$(cat VERSION)@$(git log -1 --pretty=format:%h )\"")

for PROD_ENV in $(echo $PROD_ENV_NAMES | jq -r '.[]'); do
    echo "$PROD_VERSION_CONTENT" > ./web/src/environments/version.${PROD_ENV}.ts
done

for DEV_ENV in $(echo $DEV_ENV_NAMES | jq -r '.[]'); do
    echo "$DEV_VERSION_CONTENT" > ./web/src/environments/version.${DEV_ENV}.ts
done