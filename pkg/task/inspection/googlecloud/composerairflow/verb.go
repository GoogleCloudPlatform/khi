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

package composerairflow

import (
	"github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6/style"
)

// The following block defines the registered timeline style Verbs.
// These are registered as package-level variables so they are initialized immediately
// when this package is imported.
var (
	// VerbComposerTaskInstanceScheduled represents the scheduled state verb for an Airflow task instance.
	// TODO: This will be removed when the history pane starts showing states as well as verbs.
	VerbComposerTaskInstanceScheduled = style.MustRegisterVerb("Scheduled", style.MustForceConvertSRGBHex("#1E88E5"), style.ColorWhite, true)
	// VerbComposerTaskInstanceQueued represents the queued state verb for an Airflow task instance.
	// TODO: This will be removed when the history pane starts showing states as well as verbs.
	VerbComposerTaskInstanceQueued = style.MustRegisterVerb("Queued", style.MustForceConvertSRGBHex("#22CC22"), style.ColorWhite, true)
	// VerbComposerTaskInstanceRunning represents the running state verb for an Airflow task instance.
	// TODO: This will be removed when the history pane starts showing states as well as verbs.
	VerbComposerTaskInstanceRunning = style.MustRegisterVerb("Running", style.MustForceConvertSRGBHex("#22CC22"), style.ColorWhite, true)
	// VerbComposerTaskInstanceUpForRetry represents the up-for-retry state verb for an Airflow task instance.
	// TODO: This will be removed when the history pane starts showing states as well as verbs.
	VerbComposerTaskInstanceUpForRetry = style.MustRegisterVerb("UpForRetry", style.MustForceConvertSRGBHex("#FF7700"), style.ColorWhite, true)
	// VerbComposerTaskInstanceSuccess represents the success state verb for an Airflow task instance.
	// TODO: This will be removed when the history pane starts showing states as well as verbs.
	VerbComposerTaskInstanceSuccess = style.MustRegisterVerb("Success", style.MustForceConvertSRGBHex("#22CC22"), style.ColorWhite, true)
	// VerbComposerTaskInstanceFailed represents the failed state verb for an Airflow task instance.
	// TODO: This will be removed when the history pane starts showing states as well as verbs.
	VerbComposerTaskInstanceFailed = style.MustRegisterVerb("Failed", style.MustForceConvertSRGBHex("#A51915"), style.ColorWhite, true)
	// VerbComposerTaskInstanceDeferred represents the deferred state verb for an Airflow task instance.
	// TODO: This will be removed when the history pane starts showing states as well as verbs.
	VerbComposerTaskInstanceDeferred = style.MustRegisterVerb("Deferred", style.MustForceConvertSRGBHex("#9470DC"), style.ColorWhite, true)
	// VerbComposerTaskInstanceUpForReschedule represents the up-for-reschedule state verb for an Airflow task instance.
	// TODO: This will be removed when the history pane starts showing states as well as verbs.
	VerbComposerTaskInstanceUpForReschedule = style.MustRegisterVerb("UpForReschedule", style.MustForceConvertSRGBHex("#FF7700"), style.ColorWhite, true)
	// VerbComposerTaskInstanceRemoved represents the removed state verb for an Airflow task instance.
	// TODO: This will be removed when the history pane starts showing states as well as verbs.
	VerbComposerTaskInstanceRemoved = style.MustRegisterVerb("Removed", style.MustForceConvertSRGBHex("#A51915"), style.ColorWhite, true)
	// VerbComposerTaskInstanceUpstreamFailed represents the upstream-failed state verb for an Airflow task instance.
	// TODO: This will be removed when the history pane starts showing states as well as verbs.
	VerbComposerTaskInstanceUpstreamFailed = style.MustRegisterVerb("UpstreamFailed", style.MustForceConvertSRGBHex("#A51915"), style.ColorWhite, true)
	// VerbComposerTaskInstanceZombie represents the zombie state verb for an Airflow task instance.
	// TODO: This will be removed when the history pane starts showing states as well as verbs.
	VerbComposerTaskInstanceZombie = style.MustRegisterVerb("Zombie", style.MustForceConvertSRGBHex("#696969"), style.ColorWhite, true)
	// VerbComposerTaskInstanceStats represents the stats verb for an Airflow task instance.
	// TODO: This will be removed when the history pane starts showing states as well as verbs.
	VerbComposerTaskInstanceStats = style.MustRegisterVerb("Stats", style.MustForceConvertSRGBHex("#DDDDDD"), style.ColorWhite, true)
	// VerbComposerTaskInstanceUnimplemented represents an unimplemented state verb for an Airflow task instance.
	// TODO: This will be removed when the history pane starts showing states as well as verbs.
	VerbComposerTaskInstanceUnimplemented = style.MustRegisterVerb("Unimplemented", style.MustForceConvertSRGBHex("#DDDDDD"), style.ColorWhite, true)
	// VerbComposerTaskInstanceSkipped represents the skipped state verb for an Airflow task instance.
	// TODO: This will be removed when the history pane starts showing states as well as verbs.
	VerbComposerTaskInstanceSkipped = style.MustRegisterVerb("Skipped", style.MustForceConvertSRGBHex("#e60076"), style.ColorWhite, true)
)
