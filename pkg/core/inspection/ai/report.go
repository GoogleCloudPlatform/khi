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

package ai

import (
	"context"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	core_contract "github.com/GoogleCloudPlatform/khi/pkg/task/core/contract"
)

func resolveSummaryAndCaller(ctx context.Context) (*Summary, string) {
	metadataSet, err := khictx.GetValue(ctx, inspectionmetadata.MapContextKey)
	if err != nil {
		return nil, ""
	}
	summary, found := typedmap.Get(metadataSet, SummaryMetadataKey)
	if !found {
		return nil, ""
	}
	var callerID string
	if taskImplID, err := khictx.GetValue(ctx, core_contract.TaskImplementationIDContextKey); err == nil {
		callerID = taskImplID.String()
	}
	return summary, callerID
}

func (s *Summary) setCoreLabel(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.coreLabels[key] = value
}

func (s *Summary) setProperty(callerID, key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	dest := s.resolver.destinationOf(callerID)
	switch dest.kind {
	case destinationForm:
		s.common.set(key, value)
	case destinationFeature:
		s.getOrCreateSection(dest.featureID).properties.set(key, value)
	case destinationFeatureMember:
		s.getOrCreateSection(dest.featureID).getOrCreateTaskReport(callerID).properties.set(key, value)
	case destinationShared:
		s.getOrCreateShared().properties.set(key, value)
	}
}

func (s *Summary) addIntProperty(callerID, key string, delta int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	dest := s.resolver.destinationOf(callerID)
	switch dest.kind {
	case destinationForm:
		s.common.addInt(key, delta)
	case destinationFeature:
		s.getOrCreateSection(dest.featureID).properties.addInt(key, delta)
	case destinationFeatureMember:
		s.getOrCreateSection(dest.featureID).getOrCreateTaskReport(callerID).properties.addInt(key, delta)
	case destinationShared:
		s.getOrCreateShared().properties.addInt(key, delta)
	}
}

func (s *Summary) addToSetProperty(callerID, key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	dest := s.resolver.destinationOf(callerID)
	switch dest.kind {
	case destinationForm:
		s.common.addToSet(key, value)
	case destinationFeature:
		s.getOrCreateSection(dest.featureID).properties.addToSet(key, value)
	case destinationFeatureMember:
		s.getOrCreateSection(dest.featureID).getOrCreateTaskReport(callerID).properties.addToSet(key, value)
	case destinationShared:
		s.getOrCreateShared().properties.addToSet(key, value)
	}
}

func (s *Summary) addFeatureIntProperty(callerID, key string, delta int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	dest := s.resolver.destinationOf(callerID)
	switch dest.kind {
	case destinationFeature, destinationFeatureMember:
		s.getOrCreateSection(dest.featureID).properties.addInt(key, delta)
	case destinationForm:
		s.common.addInt(key, delta)
	case destinationShared:
		s.getOrCreateShared().properties.addInt(key, delta)
	}
}

func (s *Summary) appendSummaryMarkdown(callerID, markdown string) {
	if markdown == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	dest := s.resolver.destinationOf(callerID)
	switch dest.kind {
	case destinationFeature:
		s.getOrCreateSection(dest.featureID).appendInsight(markdown)
	case destinationFeatureMember:
		s.getOrCreateSection(dest.featureID).getOrCreateTaskReport(callerID).appendMarkdown(markdown)
	case destinationForm, destinationShared:
		s.getOrCreateShared().appendInsight(markdown)
	}
}

func (s *Summary) recordQuery(callerID, id, name, text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	dest := s.resolver.destinationOf(callerID)
	switch dest.kind {
	case destinationFeature, destinationFeatureMember:
		s.getOrCreateSection(dest.featureID).recordQuery(id, name, text)
	case destinationForm, destinationShared:
		s.getOrCreateShared().recordQuery(id, name, text)
	}
}

// SetCoreLabel records a core label describing the target environment or resource.
func SetCoreLabel(ctx context.Context, key, value string) {
	s, _ := resolveSummaryAndCaller(ctx)
	if s == nil {
		return
	}
	s.setCoreLabel(key, value)
}

// SetProperty records or updates a key-value property.
func SetProperty(ctx context.Context, key, value string) {
	s, callerID := resolveSummaryAndCaller(ctx)
	if s == nil {
		return
	}
	s.setProperty(callerID, key, value)
}

// AddIntProperty adds delta to the integer property with the given key.
func AddIntProperty(ctx context.Context, key string, delta int) {
	s, callerID := resolveSummaryAndCaller(ctx)
	if s == nil {
		return
	}
	s.addIntProperty(callerID, key, delta)
}

// AddToSetProperty adds a value to the set property with the given key.
func AddToSetProperty(ctx context.Context, key, value string) {
	s, callerID := resolveSummaryAndCaller(ctx)
	if s == nil {
		return
	}
	s.addToSetProperty(callerID, key, value)
}

// AddFeatureIntProperty adds delta to the integer property of the enclosing feature section.
func AddFeatureIntProperty(ctx context.Context, key string, delta int) {
	s, callerID := resolveSummaryAndCaller(ctx)
	if s == nil {
		return
	}
	s.addFeatureIntProperty(callerID, key, delta)
}

// AppendSummaryMarkdown appends markdown content to the appropriate insight or task report.
func AppendSummaryMarkdown(ctx context.Context, markdown string) {
	s, callerID := resolveSummaryAndCaller(ctx)
	if s == nil {
		return
	}
	s.appendSummaryMarkdown(callerID, markdown)
}

// RecordQuery records an inspection query associated with the section.
func RecordQuery(ctx context.Context, id, name, text string) {
	s, callerID := resolveSummaryAndCaller(ctx)
	if s == nil {
		return
	}
	s.recordQuery(callerID, id, name, text)
}
