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

package schema

// Log severity used in KHI.
// There would be more various severity depending on the log type.
// But KHI only has these 4 different type and each parser should change the severity to them if the original severity was not in there.
type KHILogSeverity = string

const SeverityInfo = "INFO"
const SeverityWarn = "WARN"
const SeverityError = "ERROR"
const SeverityFatal = "FATAL"
const SeverityUnknown = "UNKNOWN"
