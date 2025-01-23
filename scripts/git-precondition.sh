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


TAG=$1

if [[ `git status --porcelain` ]];then
    echo "Change detected! Commit all before deploy"
    exit 1
fi

if [[ ! `git rev-parse --abbrev-ref HEAD` =~ release/[0-9]+\.[0-9]+\.[0-9]+ ]];then
    echo "You are not on release task branch!"
    exit 1
fi

if [ $(git tag -l "$TAG") ]; then
    echo "Tag $TAG is already exists on git history"
    exit 1
fi

if [[ $(gcloud config get-value project 2> /dev/null) != 'kubernetes-history-inspector' ]];then
    echo "Active project must be 'kubernetes-history-inspector'"
    exit 1
fi

exit 0