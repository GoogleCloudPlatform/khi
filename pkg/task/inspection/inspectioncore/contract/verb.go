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

import khifilev4 "github.com/GoogleCloudPlatform/khi/pkg/generated/proto/khifile/v4"

// Verb is the type of the change itself, which is saved in association with a Revision.
// A Verb representing that action should be created if the log represents a specific user action.
// VerbNil should be used for logs that do not represent a specific action and do not mention what action occurred at that time.

// VerbNil represents the action displayed as a blank. It is used for logs that do not represent a specific user action.
var VerbNil = &khifilev4.Verb{
	Label:           "Nil",
	BackgroundColor: khifilev4.MustHDRColor4FromHex("#000000FF"),
	ForegroundColor: khifilev4.MustHDRColor4FromHex("#FFFFFFFF"),
	Visible:         false,
}

// VerbCreate represents the action of creating a resource.
var VerbCreate = &khifilev4.Verb{
	Label:           "Create",
	BackgroundColor: khifilev4.MustHDRColor4FromHex("#1E88E5FF"),
	ForegroundColor: khifilev4.MustHDRColor4FromHex("#FFFFFFFF"),
	Visible:         true,
}

// VerbUpdate represents the action of updating a resource.
var VerbUpdate = &khifilev4.Verb{
	Label:           "Update",
	BackgroundColor: khifilev4.MustHDRColor4FromHex("#FDD835FF"),
	ForegroundColor: khifilev4.MustHDRColor4FromHex("#FFFFFFFF"),
	Visible:         true,
}

// VerbPatch represents the action of patching a resource.
var VerbPatch = &khifilev4.Verb{
	Label:           "Patch",
	BackgroundColor: khifilev4.MustHDRColor4FromHex("#FDD835FF"),
	ForegroundColor: khifilev4.MustHDRColor4FromHex("#FFFFFFFF"),
	Visible:         true,
}

// VerbDelete represents the action of deleting a single resource.
var VerbDelete = &khifilev4.Verb{
	Label:           "Delete",
	BackgroundColor: khifilev4.MustHDRColor4FromHex("#F54945FF"),
	ForegroundColor: khifilev4.MustHDRColor4FromHex("#FFFFFFFF"),
	Visible:         true,
}

// VerbDeleteCollection represents the action of deleting a collection of resources.
var VerbDeleteCollection = &khifilev4.Verb{
	Label:           "DeleteCollection",
	BackgroundColor: khifilev4.MustHDRColor4FromHex("#F54945FF"),
	ForegroundColor: khifilev4.MustHDRColor4FromHex("#FFFFFFFF"),
	Visible:         true,
}

// Verbs is a collection of all standard verbs defined in KHI.
// These are registered by the inspectioncore package upon initialization.
var Verbs = []*khifilev4.Verb{
	VerbNil,
	VerbCreate,
	VerbUpdate,
	VerbPatch,
	VerbDelete,
	VerbDeleteCollection,
}
