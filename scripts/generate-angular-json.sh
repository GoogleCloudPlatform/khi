#!/bin/bash
# Copyright 2024 Google LLC
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


ANGULAR_TEMPLATE="./web/angular-template.json"

PROD_ENV_FILES=$(ls ./web/src/environments/environment.prod*.ts)
DEV_ENV_FILES=$(ls ./web/src/environments/environment.dev*.ts)
ALL_ENV_FILES=$(ls ./web/src/environments/environment.*.ts)

PROD_ENV_NAMES=$(echo $PROD_ENV_FILES| jq -R -s -c '[split(" ")[]|capture(".web/src/environments/environment.(?<env>.*).ts")|.env]')
DEV_ENV_NAMES=$(echo $DEV_ENV_FILES| jq -R -s -c '[split(" ")[]|capture(".web/src/environments/environment.(?<env>.*).ts")|.env]')
ALL_ENV_NAMES=$(echo $ALL_ENV_FILES| jq -R -s -c '[split(" ")[]|capture(".web/src/environments/environment.(?<env>.*).ts")|.env]')

PROD_BUILD_CONFIGURATIONS=$(echo $PROD_ENV_NAMES | jq '[.[]|{(.):{"outputHashing":"all","fileReplacements":[{"replace":"src/environments/environment.ts","with":("src/environments/environment." + . +".ts")}]}}]|add')
DEV_BUILD_CONFIGURATIONS=$(echo $DEV_ENV_NAMES |jq '[.[]|{(.):{"buildOptimizer":false,"optimization":false,"vendorChunk":true,"extractLicenses":false,"sourceMap":true,"namedChunks":true,"fileReplacements":[{"replace":"src/environments/environment.ts","with":("src/environments/environment." + . +".ts")}]}}]|add')

BUILD_CONFIGURATIONS=$(echo "$PROD_BUILD_CONFIGURATIONS$DEV_BUILD_CONFIGURATIONS" | jq -s add)
SERVE_CONFIGURATIONS=$(echo $ALL_ENV_NAMES| jq '[.[]|{(.):{"browserTarget":("frontend:build:" + .)}}]|add')

echo $(cat $ANGULAR_TEMPLATE)$BUILD_CONFIGURATIONS$SERVE_CONFIGURATIONS | jq -s '.[0].projects.frontend.architect.build.configurations=.[1]|.[0].projects.frontend.architect.serve.configurations=.[2]|.[0]'