// Copyright 2025 Google LLC
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

var HeaderMetadataKey = NewMetadataKey[*HeaderMetadata]("header")
var FormFieldSetMetadataKey = NewMetadataKey[*FormFieldSetMetadata]("form")
var ErrorMessageSetMetadataKey = NewMetadataKey[*ErrorMessageSetMetadata]("error")

// LogMetadataKey is a key to get LogMetadata from the metadata set.
var LogMetadataKey = NewMetadataKey[*LogMetadata]("log")
var InspectionPlanMetadataKey = NewMetadataKey[*InspectionPlanMetadata]("plan")

// ProgressMetadataKey is the key used to store and retrieve Progress metadata
// from a context or metadata map.
var ProgressMetadataKey = NewMetadataKey[*Progress]("progress")
var QueryMetadataKey = NewMetadataKey[*QueryMetadata]("query")
