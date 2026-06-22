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

import (
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6/style"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
)

// The following block defines the registered timeline style Severities.
// These are registered as package-level variables so they are initialized immediately
// when this package is imported.
var (
	SeverityUnknown = style.MustRegisterSeverity("UNKNOWN", "", style.ColorBlack, style.ColorWhite, 0)
	SeverityInfo    = style.MustRegisterSeverity("INFO", "I", style.MustForceConvertSRGBHex("#0000FF"), style.ColorWhite, 1)
	SeverityWarning = style.MustRegisterSeverity("WARNING", "W", style.MustForceConvertSRGBHex("#FFAA44"), style.ColorWhite, 2)
	SeverityError   = style.MustRegisterSeverity("ERROR", "E", style.MustForceConvertSRGBHex("#FF3935"), style.ColorWhite, 3)
	SeverityFatal   = style.MustRegisterSeverity("FATAL", "F", style.MustForceConvertSRGBHex("#AA66AA"), style.ColorWhite, 4)
)

// DefaultSeverityFieldSet is a FieldSet struct type to hold the parsed log severity.
type DefaultSeverityFieldSet struct {
	// Severity is the parsed severity of the log.
	Severity *pb.Severity
}

// Kind implements log.FieldSet.
func (d *DefaultSeverityFieldSet) Kind() string {
	return "default_severity"
}

var _ log.FieldSet = (*DefaultSeverityFieldSet)(nil)
