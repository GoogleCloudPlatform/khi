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

package inspectionmetadata

// LabelKeyIncludedInRunResultFlag indicates whether metadata is included in the inspection run result.
// TODO: avoid circular dependency and use namespace in the flag name.
var LabelKeyIncludedInRunResultFlag = NewMetadataLabelsKey[bool]("metadata/include-in-run-result")

// LabelKeyIncludedInDryRunResultFlag indicates whether metadata is included in the dry-run result.
var LabelKeyIncludedInDryRunResultFlag = NewMetadataLabelsKey[bool]("metadata/include-in-dry-run-result")

// LabelKeyIncludedInTaskListFlag indicates whether metadata is included in the task list response.
var LabelKeyIncludedInTaskListFlag = NewMetadataLabelsKey[bool]("metadata/include-in-tasklist")

// LabelKeyIncludedInResultBinaryFlag indicates whether metadata is included in the serialized result binary.
var LabelKeyIncludedInResultBinaryFlag = NewMetadataLabelsKey[bool]("metadata/include-in-result-binary")

// IncludeInRunResult returns a label option that marks metadata for inclusion in the run result.
func IncludeInRunResult() MetadataLabelOpt {
	return WithLabelValue(LabelKeyIncludedInRunResultFlag, true)
}

// IncludeInDryRunResult returns a label option that marks metadata for inclusion in the dry-run result.
func IncludeInDryRunResult() MetadataLabelOpt {
	return WithLabelValue(LabelKeyIncludedInDryRunResultFlag, true)
}

// IncludeInTaskList returns a label option that marks metadata for inclusion in the task list.
func IncludeInTaskList() MetadataLabelOpt {
	return WithLabelValue(LabelKeyIncludedInTaskListFlag, true)
}

// IncludeInResultBinary returns a label option that marks metadata for inclusion in the result binary.
func IncludeInResultBinary() MetadataLabelOpt {
	return WithLabelValue(LabelKeyIncludedInResultBinaryFlag, true)
}
