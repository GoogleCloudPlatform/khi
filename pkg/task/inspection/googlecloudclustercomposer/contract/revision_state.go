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
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6/style"
)

// The following block defines the registered timeline style RevisionStates.
// These are registered as package-level variables so they are initialized immediately
// when this package is imported.
var (
	RevisionStateComposerTiScheduled = style.MustRegisterRevisionState(
		"Task instance is scheduled",
		"schedule",
		"The Airflow task instance has been scheduled and is waiting to be queued.",
		style.MustForceConvertSRGBHex("#d1b48c"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStateComposerTiQueued = style.MustRegisterRevisionState(
		"Task instance is queued",
		"transition_push",
		"The Airflow task instance has been queued in the executor and is waiting to run.",
		style.MustForceConvertSRGBHex("#808080"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStateComposerTiRunning = style.MustRegisterRevisionState(
		"Task instance is running",
		"directions_run",
		"The Airflow task instance is currently executing.",
		style.MustForceConvertSRGBHex("#00ff01"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStateComposerTiDeferred = style.MustRegisterRevisionState(
		"Task instance is deferred",
		"pause",
		"The Airflow task instance is deferred, waiting for a trigger to resume.",
		style.MustForceConvertSRGBHex("#9470dc"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStateComposerTiSuccess = style.MustRegisterRevisionState(
		"Task instance succeeded",
		"check",
		"The Airflow task instance has completed successfully.",
		style.MustForceConvertSRGBHex("#008001"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStateComposerTiFailed = style.MustRegisterRevisionState(
		"Task instance failed",
		"exclamation",
		"The Airflow task instance has failed during execution.",
		style.MustForceConvertSRGBHex("#fe0000"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStateComposerTiUpForRetry = style.MustRegisterRevisionState(
		"Task instance is up for retry",
		"camping",
		"The Airflow task instance has failed and is waiting to be retried.",
		style.MustForceConvertSRGBHex("#fed700"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStateComposerTiRestarting = style.MustRegisterRevisionState(
		"Task instance is restarting",
		"restart_alt",
		"The Airflow task instance is being restarted.",
		style.MustForceConvertSRGBHex("#ee82ef"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStateComposerTiRemoved = style.MustRegisterRevisionState(
		"Task instance is removed",
		"waving_hand",
		"The Airflow task instance has been removed from the DAG run.",
		style.MustForceConvertSRGBHex("#d3d3d3"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStateComposerTiUpstreamFailed = style.MustRegisterRevisionState(
		"Upstream task failed",
		"falling",
		"The Airflow task instance has been skipped because one of its upstream dependencies failed.",
		style.MustForceConvertSRGBHex("#ffa11b"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStateComposerTiZombie = style.MustRegisterRevisionState(
		"Task instance is a zombie",
		"skull",
		"The Airflow task instance is detected as a zombie (the process died without updating the database state).",
		style.MustForceConvertSRGBHex("#4b0082"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStateComposerTiUpForReschedule = style.MustRegisterRevisionState(
		"Task instance is up for reschedule",
		"history",
		"The Airflow task instance is in up_for_reschedule state, waiting for the next sensor poll.",
		style.MustForceConvertSRGBHex("#808080"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStateComposerTiSkipped = style.MustRegisterRevisionState(
		"Task instance is skipped",
		"step_over",
		"The Airflow task instance has been skipped during execution.",
		style.MustForceConvertSRGBHex("#e60076"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
)
