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

// Definitions for Severity used in KHI.
// While it is possible to add custom severities, it can make the UI harder to understand.
// Generally, logs should be mapped to these 5 severities. Custom severities can be
// defined within specific task packages if absolutely necessary.

// SeverityUnknown represents a severity level for logs where the intended severity
// cannot be determined or parsed from the log entry.
var SeverityUnknown = &khifilev4.Severity{
	Label:           "UNKNOWN",
	ShortLabel:      "U",
	BackgroundColor: khifilev4.MustHDRColor4FromHex("#000000FF"),
	ForegroundColor: khifilev4.MustHDRColor4FromHex("#FFFFFFFF"),
	Order:           500,
}

// SeverityInfo represents an informational severity level, typically used for
// routine, expected events.
var SeverityInfo = &khifilev4.Severity{
	Label:           "INFO",
	ShortLabel:      "I",
	BackgroundColor: khifilev4.MustHDRColor4FromHex("#0000FFFF"),
	ForegroundColor: khifilev4.MustHDRColor4FromHex("#FFFFFFFF"),
	Order:           400,
}

// SeverityWarning represents a warning severity level, indicating a potential
// issue or unexpected condition that is not critical.
var SeverityWarning = &khifilev4.Severity{
	Label:           "WARNING",
	ShortLabel:      "W",
	BackgroundColor: khifilev4.MustHDRColor4FromHex("#FFAA44FF"),
	ForegroundColor: khifilev4.MustHDRColor4FromHex("#FFFFFFFF"),
	Order:           300,
}

// SeverityError represents an error severity level, indicating a problem or
// failure that requires attention but may not halt the entire system.
var SeverityError = &khifilev4.Severity{
	Label:           "ERROR",
	ShortLabel:      "E",
	BackgroundColor: khifilev4.MustHDRColor4FromHex("#FF3935FF"),
	ForegroundColor: khifilev4.MustHDRColor4FromHex("#FFFFFFFF"),
	Order:           200,
}

// SeverityFatal represents a fatal severity level, indicating a critical
// failure that severely impacts or halts the system or component.
var SeverityFatal = &khifilev4.Severity{
	Label:           "FATAL",
	ShortLabel:      "F",
	BackgroundColor: khifilev4.MustHDRColor4FromHex("#AA66AAFF"),
	ForegroundColor: khifilev4.MustHDRColor4FromHex("#FFFFFFFF"),
	Order:           100,
}

// Severities is a collection of all standard severities defined in KHI.
// These are registered by the inspectioncore package upon initialization.
var Severities = []*khifilev4.Severity{
	SeverityUnknown,
	SeverityInfo,
	SeverityWarning,
	SeverityError,
	SeverityFatal,
}
