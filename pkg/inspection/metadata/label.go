// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metadata

import coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"

// TODO: avoid circular dependency and use namespace in the flag name
var LabelKeyIncludedInRunResultFlag = NewMetadataLabelsKey[bool]("metadata/include-in-run-result")
var LabelKeyIncludedInDryRunResultFlag = NewMetadataLabelsKey[bool]("metadata/include-in-dry-run-result")
var LabelKeyIncludedInTaskListFlag = NewMetadataLabelsKey[bool]("metadata/include-in-tasklist")
var LabelKeyIncludedInResultBinaryFlag = NewMetadataLabelsKey[bool]("metadata/include-in-result-binary")

func IncludeInRunResult() coretask.LabelOpt {
	return coretask.WithLabelValue(LabelKeyIncludedInRunResultFlag, true)
}

func IncludeInDryRunResult() coretask.LabelOpt {
	return coretask.WithLabelValue(LabelKeyIncludedInDryRunResultFlag, true)
}

func IncludeInTaskList() coretask.LabelOpt {
	return coretask.WithLabelValue(LabelKeyIncludedInTaskListFlag, true)
}

func IncludeInResultBinary() coretask.LabelOpt {
	return coretask.WithLabelValue(LabelKeyIncludedInResultBinaryFlag, true)
}
