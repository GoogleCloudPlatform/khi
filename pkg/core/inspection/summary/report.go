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

package summary

import (
	"context"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	core_contract "github.com/GoogleCloudPlatform/khi/pkg/task/core/contract"
)

func resolveCollectorAndCaller(ctx context.Context) (*Collector, string) {
	metadataSet := khictx.MustGetValue(ctx, inspectionmetadata.MapContextKey)
	collector, found := typedmap.Get(metadataSet, MetadataKey)
	if !found {
		panic("summary collector metadata not found in metadata map")
	}
	taskImplID := khictx.MustGetValue(ctx, core_contract.TaskImplementationIDContextKey)
	return collector, taskImplID.String()
}

func (c *Collector) setCoreLabel(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.coreLabels[key] = value
}

func (c *Collector) setProperty(callerID, key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	dest := c.resolver.destinationOf(callerID)
	switch dest.kind {
	case destinationForm:
		c.common.set(key, value)
	case destinationFeature:
		c.getOrCreateSection(dest.featureID).properties.set(key, value)
	case destinationFeatureMember:
		c.getOrCreateSection(dest.featureID).getOrCreateTaskReport(callerID).properties.set(key, value)
	case destinationShared:
		c.getOrCreateShared().properties.set(key, value)
	}
}

func (c *Collector) addIntProperty(callerID, key string, delta int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	dest := c.resolver.destinationOf(callerID)
	switch dest.kind {
	case destinationForm:
		c.common.addInt(key, delta)
	case destinationFeature:
		c.getOrCreateSection(dest.featureID).properties.addInt(key, delta)
	case destinationFeatureMember:
		c.getOrCreateSection(dest.featureID).getOrCreateTaskReport(callerID).properties.addInt(key, delta)
	case destinationShared:
		c.getOrCreateShared().properties.addInt(key, delta)
	}
}

func (c *Collector) addToSetProperty(callerID, key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	dest := c.resolver.destinationOf(callerID)
	switch dest.kind {
	case destinationForm:
		c.common.addToSet(key, value)
	case destinationFeature:
		c.getOrCreateSection(dest.featureID).properties.addToSet(key, value)
	case destinationFeatureMember:
		c.getOrCreateSection(dest.featureID).getOrCreateTaskReport(callerID).properties.addToSet(key, value)
	case destinationShared:
		c.getOrCreateShared().properties.addToSet(key, value)
	}
}

func (c *Collector) addFeatureIntProperty(callerID, key string, delta int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	dest := c.resolver.destinationOf(callerID)
	switch dest.kind {
	case destinationFeature, destinationFeatureMember:
		c.getOrCreateSection(dest.featureID).properties.addInt(key, delta)
	case destinationForm:
		c.common.addInt(key, delta)
	case destinationShared:
		c.getOrCreateShared().properties.addInt(key, delta)
	}
}

func (c *Collector) appendMarkdown(callerID, markdown string) {
	if markdown == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	dest := c.resolver.destinationOf(callerID)
	switch dest.kind {
	case destinationFeature:
		c.getOrCreateSection(dest.featureID).appendInsight(markdown)
	case destinationFeatureMember:
		c.getOrCreateSection(dest.featureID).getOrCreateTaskReport(callerID).appendMarkdown(markdown)
	case destinationForm, destinationShared:
		c.getOrCreateShared().appendInsight(markdown)
	}
}

func (c *Collector) recordQuery(callerID, id, name, text string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	dest := c.resolver.destinationOf(callerID)
	switch dest.kind {
	case destinationFeature, destinationFeatureMember:
		c.getOrCreateSection(dest.featureID).recordQuery(id, name, text)
	case destinationForm, destinationShared:
		c.getOrCreateShared().recordQuery(id, name, text)
	}
}

// SetCoreLabel records a core label describing the target environment or resource.
func SetCoreLabel(ctx context.Context, key, value string) {
	c, _ := resolveCollectorAndCaller(ctx)
	c.setCoreLabel(key, value)
}

// SetProperty records or updates a key-value property.
func SetProperty(ctx context.Context, key, value string) {
	c, callerID := resolveCollectorAndCaller(ctx)
	c.setProperty(callerID, key, value)
}

// AddIntProperty adds delta to the integer property with the given key.
func AddIntProperty(ctx context.Context, key string, delta int) {
	c, callerID := resolveCollectorAndCaller(ctx)
	c.addIntProperty(callerID, key, delta)
}

// AddToSetProperty adds a value to the set property with the given key.
func AddToSetProperty(ctx context.Context, key, value string) {
	c, callerID := resolveCollectorAndCaller(ctx)
	c.addToSetProperty(callerID, key, value)
}

// AddFeatureIntProperty adds delta to the integer property of the enclosing feature section.
func AddFeatureIntProperty(ctx context.Context, key string, delta int) {
	c, callerID := resolveCollectorAndCaller(ctx)
	c.addFeatureIntProperty(callerID, key, delta)
}

// AppendMarkdown appends markdown content to the appropriate insight or task report.
func AppendMarkdown(ctx context.Context, markdown string) {
	c, callerID := resolveCollectorAndCaller(ctx)
	c.appendMarkdown(callerID, markdown)
}

// RecordQuery records an inspection query associated with the section.
func RecordQuery(ctx context.Context, id, name, text string) {
	c, callerID := resolveCollectorAndCaller(ctx)
	c.recordQuery(callerID, id, name, text)
}
