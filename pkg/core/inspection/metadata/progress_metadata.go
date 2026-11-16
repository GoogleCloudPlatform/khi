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

package inspectionmetadata

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
)

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

var (
	_ json.Marshaler = (*TaskProgressMetadata)(nil)
	_ json.Marshaler = (*Progress)(nil)
)

// TaskProgressPhase represents the lifecycle phase of a task's progress.
type TaskProgressPhase string

// TaskProgressMetadata represents the progress of a single task within an inspection.
type TaskProgressMetadata struct {
	snapshot TaskProgressSnapshot
	noop     bool
	lock     sync.RWMutex
}

// TaskProgressSnapshot represents an immutable, point-in-time snapshot of TaskProgressMetadata without mutex locks.
type TaskProgressSnapshot struct {
	// ID is the task identifier.
	ID string `json:"id"`
	// Label is the human-readable display label of the task.
	Label string `json:"label"`
	// Message is the current progress status message.
	Message string `json:"message"`
	// Ratio is the completion fraction between 0.0 and 1.0.
	Ratio float32 `json:"percentage"`
	// Indeterminate indicates whether progress completion percentage is unknown.
	Indeterminate bool `json:"indeterminate"`
}

// Progress aggregates the progress of all tasks in an inspection run.
// It tracks the overall phase, total progress, and the progress of individual active tasks.
type Progress struct {
	phase             TaskProgressPhase
	totalProgress     *TaskProgressMetadata
	taskProgresses    []*TaskProgressMetadata
	totalTaskCount    int
	resolvedTaskCount int
	lock              sync.Mutex
}

// ProgressSnapshot represents an immutable, point-in-time snapshot of Progress.
type ProgressSnapshot struct {
	// Phase is the current overall lifecycle phase of the inspection.
	Phase TaskProgressPhase `json:"phase"`
	// TotalProgress is the snapshot of aggregate progress across all tasks.
	TotalProgress *TaskProgressSnapshot `json:"totalProgress"`
	// TaskProgresses lists snapshots of currently active tasks.
	TaskProgresses []TaskProgressSnapshot `json:"progresses"`
}

// NewTaskProgressMetadata creates and initializes a new TaskProgressMetadata object with the given ID.
func NewTaskProgressMetadata(id string) *TaskProgressMetadata {
	return &TaskProgressMetadata{
		snapshot: TaskProgressSnapshot{
			ID:    id,
			Label: id,
		},
	}
}

// NewNoopTaskProgressMetadata creates an immutable no-op TaskProgressMetadata instance that discards all updates.
func NewNoopTaskProgressMetadata() *TaskProgressMetadata {
	return &TaskProgressMetadata{
		snapshot: TaskProgressSnapshot{
			ID:    "noop",
			Label: "noop",
		},
		noop: true,
	}
}

// ID returns the task ID associated with this progress metadata.
func (tp *TaskProgressMetadata) ID() string {
	tp.lock.RLock()
	defer tp.lock.RUnlock()
	return tp.snapshot.ID
}

// Update updates fields from completion ratio (0.0 to 1.0) and message.
func (tp *TaskProgressMetadata) Update(ratio float32, message string) {
	if tp.noop {
		return
	}
	tp.lock.Lock()
	defer tp.lock.Unlock()
	tp.snapshot.Ratio = ratio
	tp.snapshot.Message = message
	tp.snapshot.Indeterminate = false
}

// UpdateIndeterminate marks the task progress as indeterminate and updates the status message.
func (tp *TaskProgressMetadata) UpdateIndeterminate(message string) {
	if tp.noop {
		return
	}
	tp.lock.Lock()
	defer tp.lock.Unlock()
	tp.snapshot.Indeterminate = true
	tp.snapshot.Ratio = 0
	tp.snapshot.Message = message
}

// SetLabel updates the human-readable label of the task progress.
func (tp *TaskProgressMetadata) SetLabel(label string) {
	if tp.noop {
		return
	}
	tp.lock.Lock()
	defer tp.lock.Unlock()
	tp.snapshot.Label = label
}

// Snapshot returns a thread-safe value copy of TaskProgressMetadata.
func (tp *TaskProgressMetadata) Snapshot() TaskProgressSnapshot {
	tp.lock.RLock()
	defer tp.lock.RUnlock()
	return tp.snapshot
}

// MarshalJSON implements json.Marshaler in a thread-safe manner.
func (tp *TaskProgressMetadata) MarshalJSON() ([]byte, error) {
	return json.Marshal(tp.Snapshot())
}

// NewProgress creates and initializes a new Progress object.
func NewProgress() *Progress {
	return &Progress{
		phase:             TaskPhaseRunning,
		taskProgresses:    make([]*TaskProgressMetadata, 0),
		totalProgress:     NewTaskProgressMetadata("Total"),
		resolvedTaskCount: 0,
		totalTaskCount:    0,
	}
}

// Labels implements Metadata.
func (*Progress) Labels() *typedmap.ReadonlyTypedMap {
	return NewLabelSet(
		IncludeInTaskList(),
	)
}

// ToSerializable implements Metadata.
func (p *Progress) ToSerializable() interface{} {
	return p
}

// Snapshot returns a thread-safe point-in-time copy of Progress.
func (p *Progress) Snapshot() ProgressSnapshot {
	p.lock.Lock()
	defer p.lock.Unlock()
	progresses := make([]TaskProgressSnapshot, len(p.taskProgresses))
	for i, tp := range p.taskProgresses {
		progresses[i] = tp.Snapshot()
	}
	var total *TaskProgressSnapshot
	if p.totalProgress != nil {
		snap := p.totalProgress.Snapshot()
		total = &snap
	}
	return ProgressSnapshot{
		Phase:          p.phase,
		TotalProgress:  total,
		TaskProgresses: progresses,
	}
}

// MarshalJSON implements json.Marshaler in a thread-safe manner.
func (p *Progress) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.Snapshot())
}

// SetTotalTaskCount sets the total number of tasks that will be tracked.
// This is used to calculate the overall progress completion ratio.
func (p *Progress) SetTotalTaskCount(count int) {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.totalTaskCount = count
	p.updateTotalTaskProgress()
}

// GetOrCreateTaskProgress retrieves the TaskProgress for a given task ID.
// If no progress object exists for the ID, a new one is created and added to the list.
// It returns an error if the overall progress is no longer in the RUNNING phase.
func (p *Progress) GetOrCreateTaskProgress(id string) (*TaskProgressMetadata, error) {
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.phase != TaskPhaseRunning {
		return nil, fmt.Errorf("the current progress phase is not RUNNING but %s", p.phase)
	}
	for _, progress := range p.taskProgresses {
		if progress.ID() == id {
			return progress, nil
		}
	}
	taskProgress := NewTaskProgressMetadata(id)
	p.taskProgresses = append(p.taskProgresses, taskProgress)
	return taskProgress, nil
}

// ResolveTask marks a task as resolved by removing it from the active progress list
// and increments the count of resolved tasks.
// It returns an error if the overall progress is no longer in the RUNNING phase.
func (p *Progress) ResolveTask(id string) error {
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.phase != TaskPhaseRunning {
		return fmt.Errorf("the current progress phase is not RUNNING but %s", p.phase)
	}
	newTaskProgress := make([]*TaskProgressMetadata, 0)
	for _, progress := range p.taskProgresses {
		if progress.ID() != id {
			newTaskProgress = append(newTaskProgress, progress)
		}
	}
	p.taskProgresses = newTaskProgress
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
	if p.phase != TaskPhaseRunning {
		return fmt.Errorf("the current progress phase is not RUNNING but %s", p.phase)
	}
	p.phase = TaskPhaseDone
	p.resolvedTaskCount = p.totalTaskCount
	p.taskProgresses = make([]*TaskProgressMetadata, 0)
	p.updateTotalTaskProgress()
	return nil
}

// MarkCancelled transitions the overall progress to the CANCELLED phase.
// It clears all active task progresses.
// It returns an error if the overall progress is no longer in the RUNNING phase.
func (p *Progress) MarkCancelled() error {
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.phase != TaskPhaseRunning {
		return fmt.Errorf("the current progress phase is not RUNNING but %s", p.phase)
	}
	p.phase = TaskPhaseCancelled
	p.taskProgresses = make([]*TaskProgressMetadata, 0)
	return nil
}

// MarkError transitions the overall progress to the ERROR phase.
// It clears all active task progresses.
// It returns an error if the overall progress is no longer in the RUNNING phase.
func (p *Progress) MarkError() error {
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.phase != TaskPhaseRunning {
		return fmt.Errorf("the current progress phase is not RUNNING but %s", p.phase)
	}
	p.phase = TaskPhaseError
	p.taskProgresses = make([]*TaskProgressMetadata, 0)
	return nil
}

func (p *Progress) updateTotalTaskProgress() {
	if p.totalTaskCount <= 0 {
		if p.phase == TaskPhaseDone {
			p.totalProgress.Update(1, "Complete")
		} else {
			p.totalProgress.Update(0, "")
		}
		return
	}
	p.totalProgress.Update(
		float32(p.resolvedTaskCount)/float32(p.totalTaskCount),
		fmt.Sprintf("%d of %d tasks complete", p.resolvedTaskCount, p.totalTaskCount),
	)
}
