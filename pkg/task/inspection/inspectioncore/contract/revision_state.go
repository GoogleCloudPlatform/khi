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
	RevisionStateInferred = &khifilev4.RevisionState{
		Label:           "Resource may be existing",
		BackgroundColor: khifilev4.MustHDRColor4FromHex("#999922FF"),
		Icon:            "unknown_document",
		Style:           khifilev4.RevisionStateStyle_PARTIAL_INFO,
	}

	RevisionStateExisting = &khifilev4.RevisionState{
		Label:           "Resource is existing",
		BackgroundColor: khifilev4.MustHDRColor4FromHex("#0000FFFF"),
		Icon:            "deployed_code",
		Style:           khifilev4.RevisionStateStyle_NORMAL,
	}

	RevisionStateDeleted = &khifilev4.RevisionState{
		Label:           "Resource is deleted",
		BackgroundColor: khifilev4.MustHDRColor4FromHex("#CC0000FF"),
		Icon:            "delete_forever",
		Style:           khifilev4.RevisionStateStyle_DELETED,
	}

	RevisionStateDeleting = &khifilev4.RevisionState{
		Label:           "Resource is under deleting with graceful period",
		BackgroundColor: khifilev4.MustHDRColor4FromHex("#CC5500FF"),
		Icon:            "auto_delete",
		Style:           khifilev4.RevisionStateStyle_NORMAL,
	}

	RevisionStateProvisioning = &khifilev4.RevisionState{
		Label:           "Resource is being provisioned",
		BackgroundColor: khifilev4.MustHDRColor4FromHex("#6666FFFF"),
		Icon:            "deployed_code_history",
		Style:           khifilev4.RevisionStateStyle_NORMAL,
	}

	RevisionStateOperationStarted = &khifilev4.RevisionState{
		Label:           "Processing operation",
		BackgroundColor: khifilev4.MustHDRColor4FromHex("#004400FF"),
		Icon:            "change_circle",
		Style:           khifilev4.RevisionStateStyle_NORMAL,
	}

	RevisionStateOperationFinished = &khifilev4.RevisionState{
		Label:           "Operation is finished",
		BackgroundColor: khifilev4.MustHDRColor4FromHex("#333333FF"),
		Icon:            "check_circle",
		Style:           khifilev4.RevisionStateStyle_DELETED,
	}

	RevisionStates = []*khifilev4.RevisionState{
		RevisionStateInferred,
		RevisionStateExisting,
		RevisionStateDeleted,
		RevisionStateDeleting,
		RevisionStateProvisioning,
		RevisionStateOperationStarted,
		RevisionStateOperationFinished,
	}
)
