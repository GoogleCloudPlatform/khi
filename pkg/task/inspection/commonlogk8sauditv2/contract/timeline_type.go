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

package commonlogk8sauditv2_contract

import (
	khifilev4 "github.com/GoogleCloudPlatform/khi/pkg/generated/proto/khifile/v4"
)

var (
	TimelineTypeCondition = &khifilev4.TimelineType{
		Label:           "condition",
		Description:     "A timeline showing the state changes on `.status.conditions` of the parent resource",
		BackgroundColor: khifilev4.MustHDRColor4FromHex("#4C29E8FF"),
		ForegroundColor: khifilev4.MustHDRColor4FromHex("#FFFFFFFF"),
		Visible:         true,
		SortPriority:    2000,
	}
	TimelineTypeEndpointSlice = &khifilev4.TimelineType{
		Label:           "endpoint",
		Description:     "A timeline indicates the status of endpoint related to the parent resource(Pod or Service)",
		BackgroundColor: khifilev4.MustHDRColor4FromHex("#008000FF"),
		ForegroundColor: khifilev4.MustHDRColor4FromHex("#FFFFFFFF"),
		Visible:         true,
		SortPriority:    20000,
	}
	TimelineTypeContainer = &khifilev4.TimelineType{
		Label:           "container",
		Description:     "A timline of a container included in the parent timeline of a Pod",
		BackgroundColor: khifilev4.MustHDRColor4FromHex("#FE9BABFF"),
		ForegroundColor: khifilev4.MustHDRColor4FromHex("#000000FF"),
		Visible:         true,
		SortPriority:    5000,
	}
	TimelineTypeOwnerReference = &khifilev4.TimelineType{
		Label:           "owns",
		Description:     "Owning children timeline",
		BackgroundColor: khifilev4.MustHDRColor4FromHex("#33DD88FF"),
		ForegroundColor: khifilev4.MustHDRColor4FromHex("#000000FF"),
		Visible:         true,
		SortPriority:    7000,
	}
	TimelineTypePodBinding = &khifilev4.TimelineType{
		Label:           "binds",
		Description:     "Pod binding timeline",
		BackgroundColor: khifilev4.MustHDRColor4FromHex("#FF8855FF"),
		ForegroundColor: khifilev4.MustHDRColor4FromHex("#000000FF"),
		Visible:         true,
		SortPriority:    8000,
	}
	TimelineTypePodPhase = &khifilev4.TimelineType{
		Label:           "pod",
		Description:     "Pod phase", // Using longname
		BackgroundColor: khifilev4.MustHDRColor4FromHex("#FF8855FF"),
		ForegroundColor: khifilev4.MustHDRColor4FromHex("#FFFFFFFF"),
		Visible:         true,
		SortPriority:    8000,
	}

	TimelineTypes = []*khifilev4.TimelineType{
		TimelineTypeCondition,
		TimelineTypeEndpointSlice,
		TimelineTypeContainer,
		TimelineTypeOwnerReference,
		TimelineTypePodBinding,
		TimelineTypePodPhase,
	}
)
