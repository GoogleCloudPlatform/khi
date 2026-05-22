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
		"Task instance is scheduled",
		style.MustForceConvertSRGBHex("#d1b48c"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStateComposerTiQueued = style.MustRegisterRevisionState(
		"Task instance is queued",
		"transition_push",
		"Task instance is queued",
		style.MustForceConvertSRGBHex("#808080"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStateComposerTiRunning = style.MustRegisterRevisionState(
		"Task instance is running",
		"directions_run",
		"Task instance is running",
		style.MustForceConvertSRGBHex("#00ff01"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStateComposerTiDeferred = style.MustRegisterRevisionState(
		"Task instance is deferred",
		"pause",
		"Task instance is deferred",
		style.MustForceConvertSRGBHex("#9470dc"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStateComposerTiSuccess = style.MustRegisterRevisionState(
		"Task instance completed with success state",
		"check",
		"Task instance completed with success state",
		style.MustForceConvertSRGBHex("#008001"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStateComposerTiFailed = style.MustRegisterRevisionState(
		"Task instance completed with erroneous state",
		"exclamation",
		"Task instance completed with erroneous state",
		style.MustForceConvertSRGBHex("#fe0000"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStateComposerTiUpForRetry = style.MustRegisterRevisionState(
		"Task instance is waiting for next retry",
		"camping",
		"Task instance is waiting for next retry",
		style.MustForceConvertSRGBHex("#fed700"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStateComposerTiRestarting = style.MustRegisterRevisionState(
		"Task instance is restarting",
		"restart_alt",
		"Task instance is restarting",
		style.MustForceConvertSRGBHex("#ee82ef"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStateComposerTiRemoved = style.MustRegisterRevisionState(
		"Task instance is removed",
		"waving_hand",
		"Task instance is removed",
		style.MustForceConvertSRGBHex("#d3d3d3"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStateComposerTiUpstreamFailed = style.MustRegisterRevisionState(
		"Upstream task has failed",
		"falling",
		"Upstream task has failed",
		style.MustForceConvertSRGBHex("#ffa11b"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStateComposerTiZombie = style.MustRegisterRevisionState(
		"Task instance is a zombie",
		"skull",
		"Task instance is a zombie",
		style.MustForceConvertSRGBHex("#4b0082"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStateComposerTiUpForReschedule = style.MustRegisterRevisionState(
		"Task instance is waiting to be rescheduled",
		"history",
		"Task instance is waiting to be rescheduled",
		style.MustForceConvertSRGBHex("#808080"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
	RevisionStateComposerTiSkipped = style.MustRegisterRevisionState(
		"Task instance is skipped",
		"step_over",
		"Task instance is skipped",
		style.MustForceConvertSRGBHex("#e60076"),
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)
)
