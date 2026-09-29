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

// Package summary collects bounded-size facts that tasks report during an inspection run and exposes them as a snapshot.
package summary

import (
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
)

// MetadataKey holds the typed metadata key for retrieving Collector.
var MetadataKey = inspectionmetadata.NewMetadataKey[*Collector]("inspectionSummary")

// KeyValue represents a single key-value pair.
type KeyValue struct {
	// Key holds the property name or label key.
	Key string
	// Value holds the string representation of the property or label value.
	Value string
}

// Query represents a recorded inspection query.
type Query struct {
	// Name holds the human-readable display name of the query.
	Name string
	// Text holds the query statement or filter string.
	Text string
}

// TaskReport represents summarized observations reported by an individual task.
type TaskReport struct {
	// Title holds the formatted display title of the task.
	Title string
	// Properties holds sorted key-value properties reported by the task.
	Properties []KeyValue
	// Markdown holds markdown entries joined by double newlines.
	Markdown string
}

// Section represents a summarized feature or shared section.
type Section struct {
	// Title holds the display title of the section.
	Title string
	// Queries holds sorted queries recorded within the section.
	Queries []Query
	// Properties holds sorted key-value properties of the section.
	Properties []KeyValue
	// TaskReports holds individual task reports sorted by task ID.
	TaskReports []TaskReport
	// Insights holds markdown insights joined by double newlines.
	Insights string
}

// Snapshot represents the immutable snapshot of the summary state.
type Snapshot struct {
	// CoreLabels holds sorted core labels describing the inspection target.
	CoreLabels []KeyValue
	// CommonProperties holds sorted form-level common properties.
	CommonProperties []KeyValue
	// Sections holds feature sections and the shared section.
	Sections []Section
}

type propertyKind int

const (
	propertyKindString propertyKind = iota
	propertyKindInt
	propertyKindSet
)

type property struct {
	kind      propertyKind
	strVal    string
	intVal    int
	setValues map[string]struct{}
}

// propertyTable stores properties where a key maps to a string value, an int sum, or a string set.
// Writing a different property kind to an existing key replaces the existing entry with the new kind.
type propertyTable struct {
	properties map[string]*property
}

func newPropertyTable() *propertyTable {
	return &propertyTable{
		properties: make(map[string]*property),
	}
}

func (p *propertyTable) set(key, value string) {
	p.properties[key] = &property{
		kind:   propertyKindString,
		strVal: value,
	}
}

func (p *propertyTable) addInt(key string, delta int) {
	prop, ok := p.properties[key]
	if !ok || prop.kind != propertyKindInt {
		prop = &property{
			kind:   propertyKindInt,
			intVal: 0,
		}
		p.properties[key] = prop
	}
	prop.intVal += delta
}

func (p *propertyTable) addToSet(key, value string) {
	prop, ok := p.properties[key]
	if !ok || prop.kind != propertyKindSet {
		prop = &property{
			kind:      propertyKindSet,
			setValues: make(map[string]struct{}),
		}
		p.properties[key] = prop
	}
	prop.setValues[value] = struct{}{}
}

func (p *propertyTable) toKeyValues() []KeyValue {
	if len(p.properties) == 0 {
		return nil
	}
	keys := make([]string, 0, len(p.properties))
	for k := range p.properties {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	rows := make([]KeyValue, len(keys))
	for i, k := range keys {
		prop := p.properties[k]
		var val string
		switch prop.kind {
		case propertyKindString:
			val = prop.strVal
		case propertyKindInt:
			val = strconv.Itoa(prop.intVal)
		case propertyKindSet:
			elements := make([]string, 0, len(prop.setValues))
			for elem := range prop.setValues {
				elements = append(elements, elem)
			}
			slices.Sort(elements)
			val = strings.Join(elements, ", ")
		}
		rows[i] = KeyValue{
			Key:   k,
			Value: val,
		}
	}
	return rows
}

type taskReport struct {
	properties *propertyTable
	markdown   []string
}

func newTaskReport() *taskReport {
	return &taskReport{
		properties: newPropertyTable(),
	}
}

func (tr *taskReport) appendMarkdown(md string) {
	if md == "" {
		return
	}
	tr.markdown = append(tr.markdown, md)
}

func (tr *taskReport) hasWrites() bool {
	return len(tr.properties.properties) > 0 || len(tr.markdown) > 0
}

type section struct {
	queries     map[string]Query
	properties  *propertyTable
	taskReports map[string]*taskReport
	insights    []string
}

func newSection() *section {
	return &section{
		queries:     make(map[string]Query),
		properties:  newPropertyTable(),
		taskReports: make(map[string]*taskReport),
	}
}

func (sec *section) getOrCreateTaskReport(taskID string) *taskReport {
	tr, ok := sec.taskReports[taskID]
	if !ok {
		tr = newTaskReport()
		sec.taskReports[taskID] = tr
	}
	return tr
}

func (sec *section) appendInsight(md string) {
	if md == "" {
		return
	}
	sec.insights = append(sec.insights, md)
}

func (sec *section) recordQuery(id, name, text string) {
	sec.queries[id] = Query{
		Name: name,
		Text: text,
	}
}

func (sec *section) hasWrites() bool {
	return len(sec.queries) > 0 || len(sec.properties.properties) > 0 || len(sec.taskReports) > 0 || len(sec.insights) > 0
}

// Collector aggregates structured facts and observations during an inspection run.
type Collector struct {
	resolver   *sectionResolver
	coreLabels map[string]string
	common     *propertyTable
	sections   map[string]*section
	shared     *section
	mu         sync.Mutex
}

var _ inspectionmetadata.Metadata = (*Collector)(nil)

// NewCollector creates a new Collector instance initialized with the given task graph.
func NewCollector(taskGraph *coretask.TaskSet) *Collector {
	return &Collector{
		resolver:   newSectionResolver(taskGraph),
		coreLabels: make(map[string]string),
		common:     newPropertyTable(),
		sections:   make(map[string]*section),
	}
}

// Labels returns an empty label set so the summary is not serialized into run results.
func (c *Collector) Labels() *typedmap.ReadonlyTypedMap {
	return inspectionmetadata.NewLabelSet()
}

// ToSerializable returns the immutable summary snapshot for serialization.
func (c *Collector) ToSerializable() interface{} {
	return c.Snapshot()
}

func (c *Collector) getOrCreateSection(featureID string) *section {
	sec, ok := c.sections[featureID]
	if !ok {
		sec = newSection()
		c.sections[featureID] = sec
	}
	return sec
}

func (c *Collector) getOrCreateShared() *section {
	if c.shared == nil {
		c.shared = newSection()
	}
	return c.shared
}

func (c *Collector) renderSection(title string, sec *section) Section {
	var queries []Query
	if len(sec.queries) > 0 {
		queryIDs := make([]string, 0, len(sec.queries))
		for qid := range sec.queries {
			queryIDs = append(queryIDs, qid)
		}
		slices.Sort(queryIDs)
		queries = make([]Query, len(queryIDs))
		for i, qid := range queryIDs {
			queries[i] = sec.queries[qid]
		}
	}

	var insights string
	if len(sec.insights) > 0 {
		insights = strings.Join(sec.insights, "\n\n")
	}

	return Section{
		Title:       title,
		Queries:     queries,
		Properties:  sec.properties.toKeyValues(),
		TaskReports: c.renderTaskReports(sec),
		Insights:    insights,
	}
}

func (c *Collector) renderTaskReports(sec *section) []TaskReport {
	if len(sec.taskReports) == 0 {
		return nil
	}
	taskIDs := make([]string, 0, len(sec.taskReports))
	for tid, tr := range sec.taskReports {
		if tr.hasWrites() {
			taskIDs = append(taskIDs, tid)
		}
	}
	if len(taskIDs) == 0 {
		return nil
	}
	slices.Sort(taskIDs)
	taskReports := make([]TaskReport, len(taskIDs))
	for i, tid := range taskIDs {
		tr := sec.taskReports[tid]
		var md string
		if len(tr.markdown) > 0 {
			md = strings.Join(tr.markdown, "\n\n")
		}
		taskReports[i] = TaskReport{
			Title:      c.resolver.titleOf(tid),
			Properties: tr.properties.toKeyValues(),
			Markdown:   md,
		}
	}
	return taskReports
}

func sortedKeyValues(m map[string]string) []KeyValue {
	if len(m) == 0 {
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	result := make([]KeyValue, len(keys))
	for i, k := range keys {
		result[i] = KeyValue{
			Key:   k,
			Value: m[k],
		}
	}
	return result
}

type featureSectionEntry struct {
	id    string
	order int
	title string
	sec   *section
}

func (c *Collector) featureSections() []Section {
	var featureEntries []featureSectionEntry
	for featID, sec := range c.sections {
		if !sec.hasWrites() {
			continue
		}
		info := c.resolver.features[featID]
		featureEntries = append(featureEntries, featureSectionEntry{
			id:    featID,
			order: info.order,
			title: info.title,
			sec:   sec,
		})
	}

	slices.SortFunc(featureEntries, func(a, b featureSectionEntry) int {
		if a.order != b.order {
			if a.order < b.order {
				return -1
			}
			return 1
		}
		return strings.Compare(a.id, b.id)
	})

	var sections []Section
	for _, entry := range featureEntries {
		sections = append(sections, c.renderSection(entry.title, entry.sec))
	}
	return sections
}

// Snapshot captures the current summary state as an immutable Snapshot.
func (c *Collector) Snapshot() Snapshot {
	c.mu.Lock()
	defer c.mu.Unlock()

	sections := c.featureSections()
	if c.shared != nil && c.shared.hasWrites() {
		sections = append(sections, c.renderSection("Shared & Filter Context", c.shared))
	}

	return Snapshot{
		CoreLabels:       sortedKeyValues(c.coreLabels),
		CommonProperties: c.common.toKeyValues(),
		Sections:         sections,
	}
}
