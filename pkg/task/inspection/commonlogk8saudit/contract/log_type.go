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

// The following block defines the registered timeline style LogTypes.
// These are registered as package-level variables so they are initialized immediately
// when this package is imported.
var (
	LogTypeEvent = style.MustRegisterLogType("k8s event", "Kubernetes Event Logs", style.MustForceConvertSRGBHex("#3fb549"), style.ColorWhite)
	LogTypeAudit = style.MustRegisterLogType("k8s audit", "Kubernetes Audit Logs", style.ColorBlack, style.ColorWhite)
)
