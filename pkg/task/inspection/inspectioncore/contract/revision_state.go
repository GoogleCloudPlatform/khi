// Copyright 2026 Google LLC
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

package inspectioncore_contract

import (
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6/style"
)

// RevisionStateInferred is the style for a resource where only incomplete information is available.
var RevisionStateInferred = style.MustRegisterRevisionState(
	"Resource may exist",
	"unknown_document",
	`The resource is inferred to exist from partial logs, though complete creation logs are not present in the log range.

**Tip**: Consider expanding the query time range to capture complete creation events.`,
	style.MustForceConvertSRGBHex("#999922"),
	pb.RevisionStateStyle_REVISION_STATE_STYLE_PARTIAL_INFO,
)
