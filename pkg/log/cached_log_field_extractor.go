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
	"sync"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
)

// CachedLogFieldExtractor implements CommonLogFieldExtractor and call the parent CommonLogFieldExtractor only when it was needed
// because accessing log fields from the Reader interface is a heavy operation.
type CachedLogFieldExtractor struct {
	id           string
	timestamp    time.Time
	hasTimestamp bool
	mainMessage  string
	severity     enum.Severity
	displayID    string
	parent       CommonLogFieldExtractor
	logBody      string
	lock         sync.Mutex
}

func NewCachedLogFieldExtractor(parent CommonLogFieldExtractor) *CachedLogFieldExtractor {
	return &CachedLogFieldExtractor{parent: parent}
}

// LogBody implements CommonLogFieldExtractor.
func (c *CachedLogFieldExtractor) LogBody(l *LogEntity) string {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.logBody == "" {
		c.logBody = c.parent.LogBody(l)
	}
	return c.logBody
}

// SetLogBodyCacheDirect set the given logBody as the cache of LogBody.
func (c *CachedLogFieldExtractor) SetLogBodyCacheDirect(logBody string) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.logBody = logBody
}

// DisplayID implements CommonLogFieldExtractor.
func (c *CachedLogFieldExtractor) DisplayID(l *LogEntity) string {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.displayID == "" {
		c.displayID = c.parent.DisplayID(l)
	}
	return c.displayID
}

// ID implements CommonLogFieldExtractor.
func (c *CachedLogFieldExtractor) ID(l *LogEntity) string {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.id == "" {
		c.id = c.parent.ID(l)
	}
	return c.id
}

// MainMessage implements CommonLogFieldExtractor.
func (c *CachedLogFieldExtractor) MainMessage(l *LogEntity) (string, error) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.mainMessage == "" {
		mainMessage, err := c.parent.MainMessage(l)
		if err != nil {
			return "", err
		}
		c.mainMessage = mainMessage
	}
	return c.mainMessage, nil
}

// Severity implements CommonLogFieldExtractor.
func (c *CachedLogFieldExtractor) Severity(l *LogEntity) (enum.Severity, error) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.severity == enum.SeverityUnknown {
		severity, err := c.parent.Severity(l)
		if err != nil {
			return enum.SeverityUnknown, err
		}
		c.severity = severity
	}
	return c.severity, nil
}

// Timestamp implements CommonLogFieldExtractor.
func (c *CachedLogFieldExtractor) Timestamp(l *LogEntity) time.Time {
	c.lock.Lock()
	defer c.lock.Unlock()
	if !c.hasTimestamp {
		c.timestamp = c.parent.Timestamp(l)
		c.hasTimestamp = true
	}
	return c.timestamp
}

var _ CommonLogFieldExtractor = (*CachedLogFieldExtractor)(nil)
