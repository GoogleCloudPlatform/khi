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

package googlecloudclustercomposer_contract

import (
	khifilev4 "github.com/GoogleCloudPlatform/khi/pkg/generated/proto/khifile/v4"
)

var (
	RevisionStateComposerTiScheduled = &khifilev4.RevisionState{
		Label:           "Task instance is scheduled",
		BackgroundColor: khifilev4.MustHDRColor4FromHex("#D1B48CFF"),
	}
	RevisionStateComposerTiQueued = &khifilev4.RevisionState{
		Label:           "Task instance is queued",
		BackgroundColor: khifilev4.MustHDRColor4FromHex("#808080FF"),
	}
	RevisionStateComposerTiRunning = &khifilev4.RevisionState{
		Label:           "Task instance is running",
		BackgroundColor: khifilev4.MustHDRColor4FromHex("#00FF01FF"),
	}
	RevisionStateComposerTiDeferred = &khifilev4.RevisionState{
		Label:           "Task instance is deferrd",
		BackgroundColor: khifilev4.MustHDRColor4FromHex("#9470DCFF"),
	}
	RevisionStateComposerTiSuccess = &khifilev4.RevisionState{
		Label:           "Task instance completed with success state",
		BackgroundColor: khifilev4.MustHDRColor4FromHex("#008001FF"),
	}
	RevisionStateComposerTiFailed = &khifilev4.RevisionState{
		Label:           "Task instance completed with errournous state",
		BackgroundColor: khifilev4.MustHDRColor4FromHex("#FE0000FF"),
	}
	RevisionStateComposerTiUpForRetry = &khifilev4.RevisionState{
		Label:           "Task instance is waiting for next retry",
		BackgroundColor: khifilev4.MustHDRColor4FromHex("#FED700FF"),
	}
	RevisionStateComposerTiRestarting = &khifilev4.RevisionState{
		Label:           "Task instance is being restarted",
		BackgroundColor: khifilev4.MustHDRColor4FromHex("#EE82EFFF"),
	}
	RevisionStateComposerTiRemoved = &khifilev4.RevisionState{
		Label:           "Task instance is removed",
		BackgroundColor: khifilev4.MustHDRColor4FromHex("#D3D3D3FF"),
	}
	RevisionStateComposerTiUpstreamFailed = &khifilev4.RevisionState{
		Label:           "Upstream of this task is failed",
		BackgroundColor: khifilev4.MustHDRColor4FromHex("#FFA11BFF"),
	}
	RevisionStateComposerTiZombie = &khifilev4.RevisionState{
		Label:           "Task instance is being zombie",
		BackgroundColor: khifilev4.MustHDRColor4FromHex("#4B0082FF"),
	}
	RevisionStateComposerTiUpForReschedule = &khifilev4.RevisionState{
		Label:           "Task instance is waiting for being rescheduled",
		BackgroundColor: khifilev4.MustHDRColor4FromHex("#808080FF"),
	}

	RevisionStates = []*khifilev4.RevisionState{
		RevisionStateComposerTiScheduled,
		RevisionStateComposerTiQueued,
		RevisionStateComposerTiRunning,
		RevisionStateComposerTiDeferred,
		RevisionStateComposerTiSuccess,
		RevisionStateComposerTiFailed,
		RevisionStateComposerTiUpForRetry,
		RevisionStateComposerTiRestarting,
		RevisionStateComposerTiRemoved,
		RevisionStateComposerTiUpstreamFailed,
		RevisionStateComposerTiZombie,
		RevisionStateComposerTiUpForReschedule,
	}
)
