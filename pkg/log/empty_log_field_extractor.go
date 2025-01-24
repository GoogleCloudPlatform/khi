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

package log

import (
	"fmt"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
)

// Empty commmon log field extractor. Just for testing purpose.
type UnreachableCommonFieldExtractor struct{}

var _ CommonLogFieldExtractor = (*UnreachableCommonFieldExtractor)(nil)

// LogBody implements CommonLogFieldExtractor.
func (u *UnreachableCommonFieldExtractor) LogBody(l *LogEntity) string {
	panic("unimplemented")
}

// DisplayID implements CommonLogFieldExtractor.
func (UnreachableCommonFieldExtractor) DisplayID(l *LogEntity) string {
	panic("unimplemented")
}

func (UnreachableCommonFieldExtractor) ID(l *LogEntity) string {
	panic("Unreachable")
}

func (UnreachableCommonFieldExtractor) Timestamp(l *LogEntity) time.Time {
	panic("Unreachable")
}

func (UnreachableCommonFieldExtractor) MainMessage(l *LogEntity) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (UnreachableCommonFieldExtractor) Severity(l *LogEntity) (enum.Severity, error) {
	return enum.SeverityUnknown, fmt.Errorf("not implemented")
}
