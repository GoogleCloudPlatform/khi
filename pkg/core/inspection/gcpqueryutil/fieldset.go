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

package gcpqueryutil

import (
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
)

type GCPCommonFieldSetReader struct{}

func (c *GCPCommonFieldSetReader) FieldSetKind() string {
	return (&log.CommonFieldSet{}).Kind()
}

func (c *GCPCommonFieldSetReader) Read(reader *structured.NodeReader) (log.FieldSet, error) {
	result := &log.CommonFieldSet{}
	result.Timestamp = reader.ReadTimestampOrDefault("timestamp", time.Time{})
	return result, nil
}

var _ log.FieldSetReader = (*GCPCommonFieldSetReader)(nil)
