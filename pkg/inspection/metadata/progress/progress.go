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

package progress

import (
	"fmt"
	"sync"

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata"
)

// ProgressMetadataKey is the key used to store and retrieve Progress metadata
// from a context or metadata map.
var ProgressMetadataKey = metadata.NewMetadataKey[*Progress]("progress")

// TaskProgressPhase represents the lifecycle phase of a task's progress.
type TaskProgressPhase string

// Constants defining the possible phases of a task's progress.
const (
	// TaskPhaseRunning indicates that the task is currently executing.
	TaskPhaseRunning TaskProgressPhase = "RUNNING"
	// TaskPhaseDone indicates that the task has completed successfully.
	TaskPhaseDone = "DONE"
	// TaskPhaseError indicates that the task terminated with an error.
	TaskPhaseError = "ERROR"
	// TaskPhaseCancelled indicates that the task was cancelled before completion.
	TaskPhaseCancelled = "CANCELLED"
)

// TaskProgress represents the progress of a single task within an inspection.
// It includes an ID, a human-readable label, a status message, and completion percentage.
type TaskProgress struct {
	Id            string  `json:"id"`
	Label         string  `json:"label"`
	Message       string  `json:"message"`
	Percentage    float32 `json:"percentage"`
	Indeterminate bool    `json:"indeterminate"`
}

// NewTaskProgress creates and initializes a new TaskProgress object with the given ID.
func NewTaskProgress(id string) *TaskProgress {
	return &TaskProgress{
		Id:            id,
		Indeterminate: false,
		Percentage:    0,
		Message:       "",
		Label:         id,
	}
}

// Update updates fields from percentage and message
func (tp *TaskProgress) Update(percentage float32, message string) {
	tp.Percentage = percentage
	tp.Message = message
	tp.Indeterminate = false
}

// MarkIndeterminate updates TaskProgress field to be indeterminate mode
func (tp *TaskProgress) MarkIndeterminate() {
	tp.Indeterminate = true
	tp.Percentage = 0
}

// Progress aggregates the progress of all tasks in an inspection run.
// It tracks the overall phase, total progress, and the progress of individual active tasks.
type Progress struct {
	Phase             TaskProgressPhase `json:"phase"`
	TotalProgress     *TaskProgress     `json:"totalProgress"`
	TaskProgresses    []*TaskProgress   `json:"progresses"`
	totalTaskCount    int               `json:"-"`
	resolvedTaskCount int               `json:"-"`
	lock              sync.Mutex        `json:"-"`
}

// NewProgress creates and initializes a new Progress object.
func NewProgress() *Progress {
	return &Progress{
		Phase:             TaskPhaseRunning,
		TaskProgresses:    make([]*TaskProgress, 0),
		TotalProgress:     NewTaskProgress("Total"),
		lock:              sync.Mutex{},
		resolvedTaskCount: 0,
		totalTaskCount:    0,
	}
}

// Labels implements Metadata.
func (*Progress) Labels() *typedmap.ReadonlyTypedMap {
	return metadata.NewLabelSet(
		metadata.IncludeInTaskList(),
	)
}

// ToSerializable implements Metadata.
func (p *Progress) ToSerializable() interface{} {
	return p
}

// SetTotalTaskCount sets the total number of tasks that will be tracked.
// This is used to calculate the overall progress percentage.
func (p *Progress) SetTotalTaskCount(count int) {
	p.totalTaskCount = count
	p.updateTotalTaskProgress()
}

// GetOrCreateTaskProgress retrieves the TaskProgress for a given task ID.
// If no progress object exists for the ID, a new one is created and added to the list.
// It returns an error if the overall progress is no longer in the RUNNING phase.
func (p *Progress) GetOrCreateTaskProgress(id string) (*TaskProgress, error) {
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.Phase != TaskPhaseRunning {
		return nil, fmt.Errorf("the current progress phase is not RUNNING but %s", p.Phase)
	}
	for _, progress := range p.TaskProgresses {
		if progress.Id == id {
			return progress, nil
		}
	}
	taskProgress := NewTaskProgress(id)
	p.TaskProgresses = append(p.TaskProgresses, taskProgress)
	return taskProgress, nil
}

// ResolveTask marks a task as resolved by removing it from the active progress list
// and increments the count of resolved tasks.
// It returns an error if the overall progress is no longer in the RUNNING phase.
func (p *Progress) ResolveTask(id string) error {
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.Phase != TaskPhaseRunning {
		return fmt.Errorf("the current progress phase is not RUNNING but %s", p.Phase)
	}
	newTaskProgress := make([]*TaskProgress, 0)
	for _, progress := range p.TaskProgresses {
		if progress.Id != id {
			newTaskProgress = append(newTaskProgress, progress)
		}
	}
	p.TaskProgresses = newTaskProgress
	p.resolvedTaskCount += 1
	p.updateTotalTaskProgress()
	return nil
}

// MarkDone transitions the overall progress to the DONE phase.
// It clears all active task progresses and marks the total progress as 100% complete.
// It returns an error if the overall progress is no longer in the RUNNING phase.
func (p *Progress) MarkDone() error {
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.Phase != TaskPhaseRunning {
		return fmt.Errorf("the current progress phase is not RUNNING but %s", p.Phase)
	}
	p.Phase = TaskPhaseDone
	p.resolvedTaskCount = p.totalTaskCount
	p.TaskProgresses = make([]*TaskProgress, 0)
	p.updateTotalTaskProgress()
	return nil
}

// MarkCancelled transitions the overall progress to the CANCELLED phase.
// It clears all active task progresses.
// It returns an error if the overall progress is no longer in the RUNNING phase.
func (p *Progress) MarkCancelled() error {
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.Phase != TaskPhaseRunning {
		return fmt.Errorf("the current progress phase is not RUNNING but %s", p.Phase)
	}
	p.Phase = TaskPhaseCancelled
	p.TaskProgresses = make([]*TaskProgress, 0)
	return nil
}

// MarkError transitions the overall progress to the ERROR phase.
// It clears all active task progresses.
// It returns an error if the overall progress is no longer in the RUNNING phase.
func (p *Progress) MarkError() error {
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.Phase != TaskPhaseRunning {
		return fmt.Errorf("the current progress phase is not RUNNING but %s", p.Phase)
	}
	p.Phase = TaskPhaseError
	p.TaskProgresses = make([]*TaskProgress, 0)
	return nil
}

func (p *Progress) updateTotalTaskProgress() {
	p.TotalProgress.Message = fmt.Sprintf("%d of %d tasks complete", p.resolvedTaskCount, p.totalTaskCount)
	p.TotalProgress.Percentage = float32(p.resolvedTaskCount) / float32(p.totalTaskCount)
}
