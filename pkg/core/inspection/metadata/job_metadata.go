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

package inspectionmetadata

import (
	"sync"

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
)

// JobModeCommandMetadata stores the command-line instruction to replicate the current dry-run configuration in job mode.
type JobModeCommandMetadata struct {
	command string
	lock    sync.Mutex
}

// JobModeCommandSerializable is a helper struct for JSON serialization.
type JobModeCommandSerializable struct {
	Command string `json:"command"`
}

// Labels implements Metadata.
func (*JobModeCommandMetadata) Labels() *typedmap.ReadonlyTypedMap {
	return NewLabelSet(IncludeInDryRunResult())
}

// ToSerializable implements Metadata.
func (j *JobModeCommandMetadata) ToSerializable() interface{} {
	j.lock.Lock()
	defer j.lock.Unlock()
	return &JobModeCommandSerializable{
		Command: j.command,
	}
}

// SetCommand updates the stored command example in a thread-safe way.
func (j *JobModeCommandMetadata) SetCommand(command string) {
	j.lock.Lock()
	defer j.lock.Unlock()
	j.command = command
}

// GetCommand retrieves the stored command example in a thread-safe way.
func (j *JobModeCommandMetadata) GetCommand() string {
	j.lock.Lock()
	defer j.lock.Unlock()
	return j.command
}

var _ Metadata = (*JobModeCommandMetadata)(nil)

// NewJobModeCommandMetadata creates a new JobModeCommandMetadata instance.
func NewJobModeCommandMetadata(command string) *JobModeCommandMetadata {
	return &JobModeCommandMetadata{
		command: command,
	}
}
