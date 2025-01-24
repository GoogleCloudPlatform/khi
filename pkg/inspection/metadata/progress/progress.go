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

	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
)

const ProgressMetadataKey = "progress"

const TASK_PHASE_RUNNING = "RUNNING"
const TASK_PHASE_DONE = "DONE"
const TASK_PHASE_ERROR = "ERROR"
const TASK_PHASE_CANCELLED = "CANCELLED"

type TaskProgress struct {
	Id            string  `json:"id"`
	Label         string  `json:"label"`
	Message       string  `json:"message"`
	Percentage    float32 `json:"percentage"`
	Indeterminate bool    `json:"indeterminate"`
}

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

type Progress struct {
	Phase             string          `json:"phase"`
	TotalProgress     *TaskProgress   `json:"totalProgress"`
	TaskProgresses    []*TaskProgress `json:"progresses"`
	totalTaskCount    int             `json:"-"`
	resolvedTaskCount int             `json:"-"`
	lock              sync.Mutex      `json:"-"`
}

var _ metadata.Metadata = (*Progress)(nil)

type ProgressMetadataFactory struct{}

// Instanciate implements metadata.MetadataFactory.
func (p *ProgressMetadataFactory) Instanciate() metadata.Metadata {
	return &Progress{
		Phase:             TASK_PHASE_RUNNING,
		TaskProgresses:    make([]*TaskProgress, 0),
		TotalProgress:     NewTaskProgress("Total"),
		lock:              sync.Mutex{},
		resolvedTaskCount: 0,
		totalTaskCount:    0,
	}
}

var _ metadata.MetadataFactory = (*ProgressMetadataFactory)(nil)

// Labels implements Metadata.
func (*Progress) Labels() *task.LabelSet {
	return task.NewLabelSet(
		metadata.IncludeInTaskList(),
	)
}

// ToSerializable implements Metadata.
func (p *Progress) ToSerializable() interface{} {
	return p
}

func (p *Progress) SetTotalTaskCount(count int) {
	p.totalTaskCount = count
	p.updateTotalTaskProgress()
}

func (p *Progress) GetTaskProgress(id string) (*TaskProgress, error) {
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.Phase != TASK_PHASE_RUNNING {
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

func (p *Progress) ResolveTask(id string) error {
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.Phase != TASK_PHASE_RUNNING {
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

func (p *Progress) Done() error {
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.Phase != TASK_PHASE_RUNNING {
		return fmt.Errorf("the current progress phase is not RUNNING but %s", p.Phase)
	}
	p.Phase = TASK_PHASE_DONE
	p.resolvedTaskCount = p.totalTaskCount
	p.TaskProgresses = make([]*TaskProgress, 0)
	p.updateTotalTaskProgress()
	return nil
}

func (p *Progress) Cancel() error {
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.Phase != TASK_PHASE_RUNNING {
		return fmt.Errorf("the current progress phase is not RUNNING but %s", p.Phase)
	}
	p.Phase = TASK_PHASE_CANCELLED
	p.TaskProgresses = make([]*TaskProgress, 0)
	return nil
}

func (p *Progress) Error() error {
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.Phase != TASK_PHASE_RUNNING {
		return fmt.Errorf("the current progress phase is not RUNNING but %s", p.Phase)
	}
	p.Phase = TASK_PHASE_ERROR
	p.TaskProgresses = make([]*TaskProgress, 0)
	return nil
}

func (p *Progress) updateTotalTaskProgress() {
	p.TotalProgress.Message = fmt.Sprintf("%d of %d tasks complete", p.resolvedTaskCount, p.totalTaskCount)
	p.TotalProgress.Percentage = float32(p.resolvedTaskCount) / float32(p.totalTaskCount)
}
