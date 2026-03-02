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

package googlecloudk8scommon_contract

import (
	khifilev4 "github.com/GoogleCloudPlatform/khi/pkg/generated/proto/khifile/v4"
)

var (
	TimelineTypeNetworkEndpointGroup = &khifilev4.TimelineType{
		Label:           "neg",
		Description:     "Network Endpoint Group timeline",
		BackgroundColor: khifilev4.MustHDRColor4FromHex("#A52A2AFF"),
		ForegroundColor: khifilev4.MustHDRColor4FromHex("#FFFFFFFF"),
		Visible:         true,
		SortPriority:    20500,
	}

	TimelineTypes = []*khifilev4.TimelineType{
		TimelineTypeNetworkEndpointGroup,
	}
)
