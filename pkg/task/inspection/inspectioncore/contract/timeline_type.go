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
	khifilev4 "github.com/GoogleCloudPlatform/khi/pkg/generated/proto/khifile/v4"
)

var (
	TimelineTypeResource = &khifilev4.TimelineType{
		Label:           "resource",
		Description:     "A default timeline recording the history of Kubernetes resources",
		BackgroundColor: khifilev4.MustHDRColor4FromHex("#CCCCCCFF"),
		ForegroundColor: khifilev4.MustHDRColor4FromHex("#000000FF"),
		Visible:         false,
		SortPriority:    1000,
	}

	TimelineTypes = []*khifilev4.TimelineType{
		TimelineTypeResource,
	}
)
