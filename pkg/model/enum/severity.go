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

type Severity int

const (
	SeverityUnknown   Severity = 0
	SeverityInfo      Severity = 1
	SeverityWarning   Severity = 2
	SeverityError     Severity = 3
	SeverityFatal     Severity = 4
	severityUnusedEnd          // Adds items above. This value is used for counting items in this enum to test.
)

type SeverityFrontendMetadata struct {
	// EnumKeyName is the name of this enum value. Must match with the enum key.
	EnumKeyName string
	// Label string shown on frontnend to indicate the severity.
	Label string
	// Label color used in log pane.
	LabelColor string
	// Background color of the label on log pane and the diamond shape on timeline view.
	BackgroundColor string
	// Border color of the diamond shape on timeline view.
	BorderColor string
}

var Severities = map[Severity]SeverityFrontendMetadata{
	SeverityUnknown: {
		EnumKeyName:     "SeverityUnknown",
		Label:           "UNKNOWN",
		LabelColor:      "#FFFFFF",
		BackgroundColor: "#000000",
		BorderColor:     "#AAAAAA",
	},
	SeverityInfo: {
		EnumKeyName:     "SeverityInfo",
		Label:           "INFO",
		LabelColor:      "#FFFFFF",
		BackgroundColor: "#0000FF",
		BorderColor:     "#1E88E5",
	},
	SeverityWarning: {
		EnumKeyName:     "SeverityWarning",
		Label:           "WARNING",
		LabelColor:      "#FFFFFF",
		BackgroundColor: "#FFAA44",
		BorderColor:     "#FDD835",
	},
	SeverityError: {
		EnumKeyName:     "SeverityError",
		Label:           "ERROR",
		LabelColor:      "#FFFFFF",
		BackgroundColor: "#FF3935",
		BorderColor:     "#FF8888",
	},
	SeverityFatal: {
		EnumKeyName:     "SeverityFatal",
		Label:           "FATAL",
		LabelColor:      "#FFFFFF",
		BackgroundColor: "#AA66AA",
		BorderColor:     "#FF99FF",
	},
}
