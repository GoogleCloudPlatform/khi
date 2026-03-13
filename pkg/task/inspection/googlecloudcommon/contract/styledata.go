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

package googlecloudcommon_contract

import khifilev4 "github.com/GoogleCloudPlatform/khi/pkg/generated/proto/khifile/v4"

var RevisionStateInferred = &khifilev4.RevisionState{
	Label:           "Resource may be existing",
	Icon:            "unknown_document",
	BackgroundColor: khifilev4.MustHDRColor4FromHex("#999922FF"),
	Style:           khifilev4.RevisionStateStyle_PARTIAL_INFO,
}

var RevisionStateExisting = &khifilev4.RevisionState{
	Label:           "Resource is existing",
	Icon:            "deployed_code",
	BackgroundColor: khifilev4.MustHDRColor4FromHex("#0000FFFF"),
	Style:           khifilev4.RevisionStateStyle_NORMAL,
}

var RevisionStateDeleted = &khifilev4.RevisionState{
	Label:           "Resource is deleted",
	Icon:            "delete_forever",
	BackgroundColor: khifilev4.MustHDRColor4FromHex("#CC0000FF"),
	Style:           khifilev4.RevisionStateStyle_DELETED,
}
