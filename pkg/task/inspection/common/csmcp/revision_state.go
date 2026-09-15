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

package csmcp

import (
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6/style"
)

var (
	// RevisionStateCSMCPConnectionConnected represents a connected state of a CSM CP connection.
	RevisionStateCSMCPConnectionConnected = style.MustRegisterRevisionState(
		"Established connection to CSM CP",
		"cable",
		"KHI determined the connection is established because of the log indicating \"new connection\"",
		style.Color{R: 52.0 / 255.0, G: 168.0 / 255.0, B: 83.0 / 255.0, A: 1.0}, // green
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)

	// RevisionStateCSMCPConnectionTerminated represents a terminated state of a CSM CP connection.
	RevisionStateCSMCPConnectionTerminated = style.MustRegisterRevisionState(
		"Terminated connection to CSM CP",
		"cable",
		"KHI determined the connection is terminated because of the log indicating \"terminated\"",
		style.Color{R: 234.0 / 255.0, G: 67.0 / 255.0, B: 53.0 / 255.0, A: 1.0}, // red
		pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL,
	)

	// RevisionStateCSMCPConnectionConnectedLogNotFound represents a state where termination is found without prior connection.
	RevisionStateCSMCPConnectionConnectedLogNotFound = style.MustRegisterRevisionState(
		"Established connection to CSM CP, but connection log was not found",
		"unknown_document",
		"KHI found log indicating the connection is \"terminated\", but found no log indicating the connection is \"established\" before it. The connection might be established before the specified time range.",
		style.Color{R: 52.0 / 255.0, G: 168.0 / 255.0, B: 83.0 / 255.0, A: 1.0}, // green
		pb.RevisionStateStyle_REVISION_STATE_STYLE_PARTIAL_INFO,
	)
)
