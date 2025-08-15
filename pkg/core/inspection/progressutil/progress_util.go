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

package progressutil

import (
	"fmt"

	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
)

// ReportProgressFromArraySync reports progress of processing an array synchronously.
// It updates the given TaskProgress with the current count and percentage.
// The process function is called for each element in the source array.
// If the process function returns an error, the reporting stops and the error is returned.
func ReportProgressFromArraySync[T any](progress *inspectionmetadata.TaskProgressMetadata, source []T, process func(int, T) error) error {
	fLen := float32(len(source))
	progress.Update(0, fmt.Sprintf("%d/%d", 0, len(source)))
	for i := 0; i < len(source); i++ {
		err := process(i, source[i])
		if err != nil {
			return err
		}
		progress.Update(float32(i+1)/fLen, fmt.Sprintf("%d/%d", i+1, len(source)))
	}
	return nil
}
