// Copyright 2024 Google LLC
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

package enum

type LogType int

const (
	LogTypeUnknown               LogType = 0
	LogTypeEvent                 LogType = 1
	LogTypeAudit                 LogType = 2
	LogTypeContainer             LogType = 3
	LogTypeNode                  LogType = 4
	LogTypeGkeAudit              LogType = 5
	LogTypeComputeApi            LogType = 6
	LogTypeMulticloudAPI         LogType = 7
	LogTypeOnPremAPI             LogType = 8
	LogTypeNetworkAPI            LogType = 9
	LogTypeAutoscaler            LogType = 10
	LogTypeComposerEnvironment   LogType = 11
	LogTypeControlPlaneComponent LogType = 12
	LogTypeSerialPort            LogType = 13

	logTypeUnusedEnd
)

const EnumLogTypeLength = int(logTypeUnusedEnd) + 1

type LogTypeFrontendMetadata struct {
	// EnumKeyName is the name of this enum value. Must match with the enum key.
	EnumKeyName string
	// Label string shown on frontnend to indicate the log type.
	Label string
	// Background color of the label on log pane.
	LabelBackgroundColor string
}

var LogTypes = map[LogType]LogTypeFrontendMetadata{
	LogTypeUnknown: {
		EnumKeyName:          "LogTypeUnknown",
		Label:                "unknown",
		LabelBackgroundColor: "#000000",
	},
	LogTypeEvent: {
		EnumKeyName:          "LogTypeEvent",
		Label:                "k8s_event",
		LabelBackgroundColor: "#3fb549",
	},
	LogTypeAudit: {
		EnumKeyName:          "LogTypeAudit",
		Label:                "k8s_audit",
		LabelBackgroundColor: "#000000",
	},
	LogTypeContainer: {
		EnumKeyName:          "LogTypeContainer",
		Label:                "k8s_container",
		LabelBackgroundColor: "#fe9bab",
	},
	LogTypeNode: {
		EnumKeyName:          "LogTypeNode",
		Label:                "k8s_node",
		LabelBackgroundColor: "#0077CC",
	},
	LogTypeGkeAudit: {
		EnumKeyName:          "LogTypeGkeAudit",
		Label:                "gke_audit",
		LabelBackgroundColor: "#AA00FF",
	},
	LogTypeComputeApi: {
		EnumKeyName:          "LogTypeComputeApi",
		Label:                "compute_api",
		LabelBackgroundColor: "#FFCC33",
	},
	LogTypeMulticloudAPI: {
		EnumKeyName:          "LogTypeMulticloudAPI",
		Label:                "multicloud_api",
		LabelBackgroundColor: "#AA00FF",
	},
	LogTypeOnPremAPI: {
		EnumKeyName:          "LogTypeOnPremAPI",
		Label:                "onprem_api",
		LabelBackgroundColor: "#AA00FF",
	},
	LogTypeNetworkAPI: {
		EnumKeyName:          "LogTypeNetworkAPI",
		Label:                "network_api",
		LabelBackgroundColor: "#33CCFF",
	},
	LogTypeAutoscaler: {
		EnumKeyName:          "LogTypeAutoscaler",
		Label:                "autoscaler",
		LabelBackgroundColor: "#FF5555",
	},
	LogTypeComposerEnvironment: {
		EnumKeyName:          "LogTypeComposerEnvironment",
		Label:                "composer_environment",
		LabelBackgroundColor: "#88AA55",
	},
	LogTypeControlPlaneComponent: {
		EnumKeyName:          "LogTypeControlPlaneComponent",
		Label:                "control_plane_component",
		LabelBackgroundColor: "#FF3333",
	},
	LogTypeSerialPort: {
		EnumKeyName:          "LogTypeSerialPort",
		Label:                "serial_port",
		LabelBackgroundColor: "#333333",
	},
}
