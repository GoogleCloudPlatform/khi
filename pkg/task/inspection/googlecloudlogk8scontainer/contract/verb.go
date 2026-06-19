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

package googlecloudlogk8scontainer_contract

import (
	"github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6/style"
)

// The following block defines the registered timeline style Verbs.
// These are registered as package-level variables so they are initialized immediately
// when this package is imported.
var (
	// TODO: This will be removed when the history pane starts showing states as well as verbs.
	VerbContainerWaiting = style.MustRegisterVerb("Waiting", style.MustForceConvertSRGBHex("#FDD835"), style.ColorWhite, true)
	// TODO: This will be removed when the history pane starts showing states as well as verbs.
	VerbContainerReady = style.MustRegisterVerb("Ready", style.MustForceConvertSRGBHex("#22CC22"), style.ColorWhite, true)
	// TODO: This will be removed when the history pane starts showing states as well as verbs.
	VerbContainerNonReady = style.MustRegisterVerb("NonReady", style.MustForceConvertSRGBHex("#FF7700"), style.ColorWhite, true)
	// TODO: This will be removed when the history pane starts showing states as well as verbs.
	VerbContainerSuccess = style.MustRegisterVerb("Success", style.MustForceConvertSRGBHex("#007700"), style.ColorWhite, true)
	// TODO: This will be removed when the history pane starts showing states as well as verbs.
	VerbContainerError = style.MustRegisterVerb("Error", style.MustForceConvertSRGBHex("#A51915"), style.ColorWhite, true)
)
