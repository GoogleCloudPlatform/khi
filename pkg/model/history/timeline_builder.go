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

package history

import (
	"slices"
	"sync"
	"time"
)

// An utility class to modify specific resource timeline obtained from history.
type TimelineBuilder struct {
	builder  *Builder
	timeline *ResourceTimeline
	lock     sync.Mutex
	sorted   bool
}

func newTimelineBuilder(builder *Builder, timeline *ResourceTimeline) *TimelineBuilder {
	return &TimelineBuilder{
		builder:  builder,
		timeline: timeline,
		lock:     sync.Mutex{},
	}
}

func (b *TimelineBuilder) AddEvent(event *ResourceEvent) {
	b.lock.Lock()
	defer b.lock.Unlock()
	timeline := b.timeline
	timeline.Events = append(timeline.Events, event)
	if len(timeline.Events) >= 2 {
		prev := timeline.Events[len(timeline.Events)-2]
		if b.timeDiffOfLogIndicces(event.Log, prev.Log) < 0 {
			b.sorted = false
		}
	}
}

func (b *TimelineBuilder) Sort() {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.sortWithoutLock()
}

func (b *TimelineBuilder) AddRevision(revision *ResourceRevision) {
	b.lock.Lock()
	defer b.lock.Unlock()
	timeline := b.timeline
	b.timeline.Revisions = append(b.timeline.Revisions, revision)
	if len(timeline.Revisions) >= 2 {
		prev := timeline.Revisions[len(timeline.Revisions)-2]
		if revision.ChangeTime.Sub(prev.ChangeTime) < 0 {
			b.sorted = false
		}
	}
}

// GetRevisions return ResourceRevision list already written to this timeline.
func (b *TimelineBuilder) GetRevisions() []*ResourceRevision {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.sortWithoutLock()
	return b.timeline.Revisions
}

// GetRevisionBefore find a revision before the given time. If no revision found, then this function returns nil.
// Computational complexity is O(logN)
func (b *TimelineBuilder) GetRevisionBefore(time time.Time) *ResourceRevision {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.sortWithoutLock()
	if len(b.timeline.Revisions) == 0 {
		return nil
	}
	if b.timeline.Revisions[0].ChangeTime.Sub(time) > 0 {
		return nil // given time is before the all revisions
	}
	if time.Sub(b.timeline.Revisions[len(b.timeline.Revisions)-1].ChangeTime) >= 0 {
		return b.timeline.Revisions[len(b.timeline.Revisions)-1]
	}

	min := 0
	max := len(b.timeline.Revisions) - 1
	for max-min > 1 {
		mid := (min + max) / 2
		if b.timeline.Revisions[mid].ChangeTime.Sub(time) > 0 {
			max = mid
		} else {
			min = mid
		}
	}
	return b.timeline.Revisions[min]
}

// GetRevisionAfter find a revision after the given time. If no revision found, then this function returns nil.
// Computational complexity is O(logN)
func (b *TimelineBuilder) GetRevisionAfter(time time.Time) *ResourceRevision {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.sortWithoutLock()
	if len(b.timeline.Revisions) == 0 {
		return nil
	}
	if time.Sub(b.timeline.Revisions[len(b.timeline.Revisions)-1].ChangeTime) > 0 {
		return nil
	}
	if b.timeline.Revisions[0].ChangeTime.Sub(time) >= 0 {
		return b.timeline.Revisions[0]
	}

	min := 0
	max := len(b.timeline.Revisions) - 1
	for max-min > 1 {
		mid := (min + max) / 2
		if b.timeline.Revisions[mid].ChangeTime.Sub(time) >= 0 {
			max = mid
		} else {
			min = mid
		}
	}
	return b.timeline.Revisions[max]
}

// Get the latest revision stored in a specific ResourceHistory.
// Returns nil when specified resource was nil or no any revisions recorded.
func (b *TimelineBuilder) GetLatestRevision() *ResourceRevision {
	b.lock.Lock()
	defer b.lock.Unlock()
	if !b.sorted {
		b.sortWithoutLock()
	}
	if len(b.timeline.Revisions) > 0 {
		return b.timeline.Revisions[len(b.timeline.Revisions)-1]
	}
	return nil
}

func (b *TimelineBuilder) GetLatestRevisionBody() (string, error) {
	body, err := b.builder.BinaryBuilder.Read(b.GetLatestRevision().Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// GetClonedEvents copies the list o ResourceEvent. This method is for testing purpose.
func (b *TimelineBuilder) GetClonedEvents() []ResourceEvent {
	var events []ResourceEvent
	for _, event := range b.timeline.Events {
		events = append(events, *event)
	}
	return events
}

func (b *TimelineBuilder) sortWithoutLock() {
	if b.sorted {
		return
	}
	slices.SortStableFunc(b.timeline.Events, func(x, y *ResourceEvent) int {
		return b.timeDiffOfLogIndicces(x.Log, y.Log)
	})
	slices.SortStableFunc(b.timeline.Revisions, func(x, y *ResourceRevision) int {
		return int(x.ChangeTime.Sub(y.ChangeTime))
	})
	b.sorted = true
}

func (b *TimelineBuilder) timeDiffOfLogIndicces(x string, y string) int {
	xl, errx := b.builder.GetLog(x)
	yl, erry := b.builder.GetLog(y)
	xlt := time.Unix(0, 0)
	ylt := time.Unix(0, 0)
	if errx == nil {
		xlt = xl.Timestamp
	}
	if erry == nil {
		ylt = yl.Timestamp
	}
	return int(xlt.Sub(ylt))
}
