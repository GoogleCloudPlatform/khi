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

package commonlogk8saudit_contract

import (
	"github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6/style"
)

// The following block defines the registered timeline style Verbs.
// These are registered as package-level variables so they are initialized immediately
// when this package is imported.
var (
	VerbCreate           = style.MustRegisterVerb("Create", style.MustForceConvertSRGBHex("#1E88E5"), style.ColorWhite, true)
	VerbDelete           = style.MustRegisterVerb("Delete", style.MustForceConvertSRGBHex("#F54945"), style.ColorWhite, true)
	VerbUpdate           = style.MustRegisterVerb("Update", style.MustForceConvertSRGBHex("#FDD835"), style.ColorWhite, true)
	VerbPatch            = style.MustRegisterVerb("Patch", style.MustForceConvertSRGBHex("#FDD835"), style.ColorWhite, true)
	VerbDeleteCollection = style.MustRegisterVerb("DeleteCollection", style.MustForceConvertSRGBHex("#F54945"), style.ColorWhite, true)

	// TODO: This will be removed when the history pane starts showing states as well as verbs.
	VerbReady = style.MustRegisterVerb("Ready", style.MustForceConvertSRGBHex("#22CC22"), style.ColorWhite, true)
	// TODO: This will be removed when the history pane starts showing states as well as verbs.
	VerbNonReady = style.MustRegisterVerb("NonReady", style.MustForceConvertSRGBHex("#FF7700"), style.ColorWhite, true)
	// TODO: This will be removed when the history pane starts showing states as well as verbs.
	VerbTerminating = style.MustRegisterVerb("Terminating", style.MustForceConvertSRGBHex("#FFAA00"), style.ColorWhite, true)
	// TODO: This will be removed when the history pane starts showing states as well as verbs.
	VerbStatusUnknown = style.MustRegisterVerb("Condition(Unknown)", style.MustForceConvertSRGBHex("#AA66AA"), style.ColorWhite, true)
	// TODO: This will be removed when the history pane starts showing states as well as verbs.
	VerbStatusTrue = style.MustRegisterVerb("Condition(True)", style.MustForceConvertSRGBHex("#22CC22"), style.ColorWhite, true)
	// TODO: This will be removed when the history pane starts showing states as well as verbs.
	VerbStatusFalse = style.MustRegisterVerb("Condition(False)", style.MustForceConvertSRGBHex("#FF7700"), style.ColorWhite, true)
)
