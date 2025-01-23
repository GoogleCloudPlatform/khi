// Copyright 2025 Google LLC
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

package errorreport

import (
	"context"
	"fmt"
)

// CheckAndReportPanic checks current function is not raising an error with panic and it reports the error when recover returns an error.
// This function be called with defer to catch the error happend in the function.
func CheckAndReportPanic() {
	if r := recover(); r != nil {
		err := fmt.Errorf("panic occurred: %v", r)
		DefaultErrorReporter.ReportSync(context.Background(), err)
		panic(r)
	}
}
