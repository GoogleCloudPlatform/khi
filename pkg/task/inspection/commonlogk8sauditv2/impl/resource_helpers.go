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

package commonlogk8sauditv2_impl

import (
	"log/slog"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
)

// GetDeletionGracePeriodSeconds returns the value of metadata.deletionGracePeriodSeconds.
// It returns the value and true if the field exists and is an integer.
// Otherwise, it returns 0 and false.
func GetDeletionGracePeriodSeconds(reader *structured.NodeReader) (int, bool) {
	if reader == nil {
		return 0, false
	}
	val, err := reader.ReadInt("metadata.deletionGracePeriodSeconds")
	if err != nil {
		return 0, false
	}
	return val, true
}

// GetDeletionTimestamp returns the value of metadata.deletionTimestamp.
// It returns the value and true if the field exists and is a string.
// Otherwise, it returns empty string and false.
func GetDeletionTimestamp(reader *structured.NodeReader) (string, bool) {
	if reader == nil {
		return "", false
	}
	val, err := reader.ReadString("metadata.deletionTimestamp")
	if err != nil {
		return "", false
	}
	return val, true
}

// GetFinalizers returns the list of finalizers from metadata.finalizers or spec.finalizers.
// It checks metadata.finalizers first, then spec.finalizers (for Namespace).
// It returns the list and true if at least one finalizer list exists.
// Note that it returns true even if the list is empty, as long as the field exists.
func GetFinalizers(reader *structured.NodeReader) ([]string, bool) {
	if reader == nil {
		return nil, false
	}

	readFinalizers := func(path string) ([]string, bool) {
		r, err := reader.GetReader(path)
		if err != nil {
			return nil, false
		}
		// Assuming finalizers is a list of strings
		// We need to iterate or read as string list.
		// structured.NodeReader doesn't have ReadStringList directly visible in the previous context,
		// but let's check how it was used: `finalizers.Len() > 0`.
		// We can try to read it as a list.
		var result []string
		// Iterate over the list
		for _, v := range r.Children() {
			v, err := v.NodeScalarValue()
			if err != nil {
				slog.Warn("an error occurred while reading finalizer elements", "err", err)
				continue
			}
			result = append(result, v.(string))
		}
		return result, true
	}

	if list, ok := readFinalizers("metadata.finalizers"); ok {
		return list, true
	}
	if list, ok := readFinalizers("spec.finalizers"); ok {
		return list, true
	}
	return nil, false
}

// GetPodPhase returns the value of status.phase.
// It returns the value and true if the field exists and is a string.
// Otherwise, it returns empty string and false.
func GetPodPhase(reader *structured.NodeReader) (string, bool) {
	if reader == nil {
		return "", false
	}
	val, err := reader.ReadString("status.phase")
	if err != nil {
		return "", false
	}
	return val, true
}

// GetUID returns the value of metadata.uid.
// It returns the value and true if the field exists and is a string.
// Otherwise, it returns empty string and false.
func GetUID(reader *structured.NodeReader) (string, bool) {
	if reader == nil {
		return "", false
	}
	val, err := reader.ReadString("metadata.uid")
	if err != nil {
		return "", false
	}
	return val, true
}

// GetNodeNameOfPod returns the value of spec.nodeName.
// It returns the value and true if the field exists and is a string.
// Otherwise, it returns empty string and false.
func GetNodeNameOfPod(reader *structured.NodeReader) (string, bool) {
	if reader == nil {
		return "", false
	}
	val, err := reader.ReadString("spec.nodeName")
	if err != nil {
		return "", false
	}
	return val, true
}

// GetCreationTimestamp returns the value of metadata.creationTimestamp.
// It returns the value and true if the field exists and is a valid timestamp.
// Otherwise, it returns time.Time{} and false.
func GetCreationTimestamp(reader *structured.NodeReader) (time.Time, bool) {
	if reader == nil {
		return time.Time{}, false
	}
	val, err := reader.ReadString("metadata.creationTimestamp")
	if err != nil {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339, val)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}
