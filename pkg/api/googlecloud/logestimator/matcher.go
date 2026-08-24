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

package logestimator

import (
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/gcpqueryutil"
)

// LoggingMonitoringMatcher defines a self-contained filter clause for both
// Cloud Logging query strings and Cloud Monitoring metric filters.
type LoggingMonitoringMatcher interface {
	// SupportMetrics returns true if this filter condition can be natively represented in Cloud Monitoring.
	SupportMetrics() bool

	// ToLoggingQuery returns the filter fragment for Cloud Logging queries.
	ToLoggingQuery() string

	// ToMonitoringFilter returns the filter fragment for Cloud Monitoring metric filters.
	ToMonitoringFilter() string
}

// ValueMatcher defines how a set of values or patterns expands for both
// general field names (e.g. resource.labels.key) and LogID macros.
type ValueMatcher interface {
	// ToLoggingFieldQuery returns the Cloud Logging query fragment for a named field.
	ToLoggingFieldQuery(fieldName string) string

	// ToMonitoringFieldFilter returns the Cloud Monitoring metric filter fragment for a named field.
	ToMonitoringFieldFilter(fieldName string) string

	// ToLoggingLogIDQuery returns the Cloud Logging query fragment for LOG_ID(...).
	ToLoggingLogIDQuery() string

	// ToMonitoringLogIDFilter returns the Cloud Monitoring metric filter fragment for metric.labels.log.
	ToMonitoringLogIDFilter() string
}

// ----------------------------------------------------------------------------
// ResourceLabel Matcher
// ----------------------------------------------------------------------------

type resourceLabelMatcher struct {
	labelKey string
	vm       ValueMatcher
}

var _ LoggingMonitoringMatcher = (*resourceLabelMatcher)(nil)

// ResourceLabel creates a LoggingMonitoringMatcher for a resource.labels.<key> field.
func ResourceLabel(labelKey string, vm ValueMatcher) LoggingMonitoringMatcher {
	if vm == nil {
		return nil
	}
	return &resourceLabelMatcher{
		labelKey: labelKey,
		vm:       vm,
	}
}

func (m *resourceLabelMatcher) SupportMetrics() bool {
	return m.vm.ToMonitoringFieldFilter("resource.labels."+m.labelKey) != ""
}

func (m *resourceLabelMatcher) ToLoggingQuery() string {
	return m.vm.ToLoggingFieldQuery("resource.labels." + m.labelKey)
}

func (m *resourceLabelMatcher) ToMonitoringFilter() string {
	return m.vm.ToMonitoringFieldFilter("resource.labels." + m.labelKey)
}

// ----------------------------------------------------------------------------
// LogID Matcher
// ----------------------------------------------------------------------------

type logIDMatcher struct {
	vm ValueMatcher
}

var _ LoggingMonitoringMatcher = (*logIDMatcher)(nil)

// LogID creates a LoggingMonitoringMatcher for log IDs (LOG_ID macro in Logging, metric.labels.log in Monitoring).
func LogID(vm ValueMatcher) LoggingMonitoringMatcher {
	if vm == nil {
		return nil
	}
	return &logIDMatcher{
		vm: vm,
	}
}

func (m *logIDMatcher) SupportMetrics() bool {
	return m.vm.ToMonitoringLogIDFilter() != ""
}

func (m *logIDMatcher) ToLoggingQuery() string {
	return m.vm.ToLoggingLogIDQuery()
}

func (m *logIDMatcher) ToMonitoringFilter() string {
	return m.vm.ToMonitoringLogIDFilter()
}

// ----------------------------------------------------------------------------
// CustomFilter Matcher
// ----------------------------------------------------------------------------

type customFilterMatcher struct {
	rawQuery string
}

var _ LoggingMonitoringMatcher = (*customFilterMatcher)(nil)

// CustomFilter creates a LoggingMonitoringMatcher for payload/custom filters.
// Cloud Monitoring does not support payload filters natively, so SupportMetrics returns false.
func CustomFilter(rawQuery string) LoggingMonitoringMatcher {
	trimmed := strings.TrimSpace(rawQuery)
	if trimmed == "" {
		return nil
	}
	return &customFilterMatcher{
		rawQuery: trimmed,
	}
}

func (m *customFilterMatcher) SupportMetrics() bool {
	return false
}

func (m *customFilterMatcher) ToLoggingQuery() string {
	return m.rawQuery
}

func (m *customFilterMatcher) ToMonitoringFilter() string {
	return ""
}

// ----------------------------------------------------------------------------
// Comment Matcher
// ----------------------------------------------------------------------------

type commentMatcher struct {
	comment string
}

var _ LoggingMonitoringMatcher = (*commentMatcher)(nil)

// Comment creates a LoggingMonitoringMatcher for a standalone comment-only line.
// The comment text is formatted as "-- <comment>".
// Since comments do not filter log entries, SupportMetrics returns true and ToMonitoringFilter returns an empty string.
func Comment(comment string) LoggingMonitoringMatcher {
	trimmed := strings.TrimSpace(comment)
	if trimmed == "" {
		return nil
	}
	return &commentMatcher{
		comment: trimmed,
	}
}

func (m *commentMatcher) SupportMetrics() bool {
	return true
}

func (m *commentMatcher) ToLoggingQuery() string {
	return fmt.Sprintf("-- %s", m.comment)
}

func (m *commentMatcher) ToMonitoringFilter() string {
	return ""
}

// ----------------------------------------------------------------------------
// WithComment Matcher Wrapper
// ----------------------------------------------------------------------------

type withCommentMatcher struct {
	inner   LoggingMonitoringMatcher
	comment string
}

var _ LoggingMonitoringMatcher = (*withCommentMatcher)(nil)

// WithComment wraps an existing LoggingMonitoringMatcher to append a trailing comment (" -- <comment>")
// to its Cloud Logging query representation.
// Cloud Monitoring metric evaluation is delegated to the wrapped matcher without appending the comment.
func WithComment(matcher LoggingMonitoringMatcher, comment string) LoggingMonitoringMatcher {
	if matcher == nil {
		return nil
	}
	trimmed := strings.TrimSpace(comment)
	if trimmed == "" {
		return matcher
	}
	return &withCommentMatcher{
		inner:   matcher,
		comment: trimmed,
	}
}

func (m *withCommentMatcher) SupportMetrics() bool {
	return m.inner.SupportMetrics()
}

func (m *withCommentMatcher) ToLoggingQuery() string {
	innerQuery := m.inner.ToLoggingQuery()
	if innerQuery == "" {
		return fmt.Sprintf("-- %s", m.comment)
	}
	return fmt.Sprintf("%s -- %s", innerQuery, m.comment)
}

func (m *withCommentMatcher) ToMonitoringFilter() string {
	return m.inner.ToMonitoringFilter()
}

// ----------------------------------------------------------------------------
// ValueMatcher Implementations
// ----------------------------------------------------------------------------

// exactMatcher matches an exact value.
type exactMatcher struct {
	val string
}

var _ ValueMatcher = (*exactMatcher)(nil)

// Exact creates a ValueMatcher for an exact single value.
func Exact(val string) ValueMatcher {
	return &exactMatcher{val: val}
}

func (m *exactMatcher) ToLoggingFieldQuery(fieldName string) string {
	return fmt.Sprintf(`%s="%s"`, fieldName, m.val)
}

func (m *exactMatcher) ToMonitoringFieldFilter(fieldName string) string {
	return fmt.Sprintf(`%s = "%s"`, fieldName, m.val)
}

func (m *exactMatcher) ToLoggingLogIDQuery() string {
	return fmt.Sprintf(`LOG_ID("%s")`, m.val)
}

func (m *exactMatcher) ToMonitoringLogIDFilter() string {
	return fmt.Sprintf(`metric.labels.log = "%s"`, m.val)
}

// oneOfMatcher matches any value in a set (OR).
type oneOfMatcher struct {
	vals []string
}

var _ ValueMatcher = (*oneOfMatcher)(nil)

// OneOf creates a ValueMatcher that matches any of the provided values (OR).
func OneOf(vals ...string) ValueMatcher {
	filtered := filterNonEmpty(vals)
	if len(filtered) == 0 {
		return nil
	}
	return &oneOfMatcher{vals: filtered}
}

func (m *oneOfMatcher) ToLoggingFieldQuery(fieldName string) string {
	if len(m.vals) == 1 {
		return fmt.Sprintf(`%s="%s"`, fieldName, m.vals[0])
	}
	quoted := quoteStrings(m.vals)
	return fmt.Sprintf(`%s=(%s)`, fieldName, strings.Join(quoted, " OR "))
}

func (m *oneOfMatcher) ToMonitoringFieldFilter(fieldName string) string {
	if len(m.vals) == 1 {
		return fmt.Sprintf(`%s = "%s"`, fieldName, m.vals[0])
	}
	quoted := quoteStrings(m.vals)
	return fmt.Sprintf(`%s = one_of(%s)`, fieldName, strings.Join(quoted, ", "))
}

func (m *oneOfMatcher) ToLoggingLogIDQuery() string {
	if len(m.vals) == 1 {
		return fmt.Sprintf(`LOG_ID("%s")`, m.vals[0])
	}
	parts := make([]string, len(m.vals))
	for i, v := range m.vals {
		parts[i] = fmt.Sprintf(`LOG_ID("%s")`, v)
	}
	return "(" + strings.Join(parts, " OR ") + ")"
}

func (m *oneOfMatcher) ToMonitoringLogIDFilter() string {
	if len(m.vals) == 1 {
		return fmt.Sprintf(`metric.labels.log = "%s"`, m.vals[0])
	}
	quoted := quoteStrings(m.vals)
	return fmt.Sprintf(`metric.labels.log = one_of(%s)`, strings.Join(quoted, ", "))
}

// noneOfMatcher excludes all values in a set (NOT).
type noneOfMatcher struct {
	vals []string
}

var _ ValueMatcher = (*noneOfMatcher)(nil)

// NoneOf creates a ValueMatcher that excludes all provided values (NOT).
func NoneOf(vals ...string) ValueMatcher {
	filtered := filterNonEmpty(vals)
	if len(filtered) == 0 {
		return nil
	}
	return &noneOfMatcher{vals: filtered}
}

func (m *noneOfMatcher) ToLoggingFieldQuery(fieldName string) string {
	if len(m.vals) == 1 {
		return fmt.Sprintf(`-%s="%s"`, fieldName, m.vals[0])
	}
	quoted := quoteStrings(m.vals)
	return fmt.Sprintf(`-%s=(%s)`, fieldName, strings.Join(quoted, " OR "))
}

func (m *noneOfMatcher) ToMonitoringFieldFilter(fieldName string) string {
	parts := make([]string, len(m.vals))
	for i, v := range m.vals {
		parts[i] = fmt.Sprintf(`%s != "%s"`, fieldName, v)
	}
	return strings.Join(parts, " AND ")
}

func (m *noneOfMatcher) ToLoggingLogIDQuery() string {
	lines := make([]string, len(m.vals))
	for i, v := range m.vals {
		lines[i] = fmt.Sprintf(`-LOG_ID("%s")`, v)
	}
	return strings.Join(lines, "\n")
}

func (m *noneOfMatcher) ToMonitoringLogIDFilter() string {
	parts := make([]string, len(m.vals))
	for i, v := range m.vals {
		parts[i] = fmt.Sprintf(`metric.labels.log != "%s"`, v)
	}
	return strings.Join(parts, " AND ")
}

// containsAnyMatcher matches any substring in a set.
type containsAnyMatcher struct {
	subs []string
}

var _ ValueMatcher = (*containsAnyMatcher)(nil)

// ContainsAny creates a ValueMatcher matching any of the substrings.
func ContainsAny(subs ...string) ValueMatcher {
	filtered := filterNonEmpty(subs)
	if len(filtered) == 0 {
		return nil
	}
	return &containsAnyMatcher{subs: filtered}
}

func (m *containsAnyMatcher) ToLoggingFieldQuery(fieldName string) string {
	if len(m.subs) == 1 {
		return fmt.Sprintf(`%s:"%s"`, fieldName, m.subs[0])
	}
	quoted := quoteStrings(m.subs)
	return fmt.Sprintf(`%s:(%s)`, fieldName, strings.Join(quoted, " OR "))
}

func (m *containsAnyMatcher) ToMonitoringFieldFilter(fieldName string) string {
	if len(m.subs) == 1 {
		return fmt.Sprintf(`%s = has_substring("%s")`, fieldName, m.subs[0])
	}
	parts := make([]string, len(m.subs))
	for i, s := range m.subs {
		parts[i] = fmt.Sprintf(`%s = has_substring("%s")`, fieldName, s)
	}
	return "(" + strings.Join(parts, " OR ") + ")"
}

func (m *containsAnyMatcher) ToLoggingLogIDQuery() string {
	return ""
}

func (m *containsAnyMatcher) ToMonitoringLogIDFilter() string {
	return ""
}

// notContainsAnyMatcher excludes all substrings in a set.
type notContainsAnyMatcher struct {
	subs []string
}

var _ ValueMatcher = (*notContainsAnyMatcher)(nil)

// NotContainsAny creates a ValueMatcher excluding all of the substrings.
func NotContainsAny(subs ...string) ValueMatcher {
	filtered := filterNonEmpty(subs)
	if len(filtered) == 0 {
		return nil
	}
	return &notContainsAnyMatcher{subs: filtered}
}

func (m *notContainsAnyMatcher) ToLoggingFieldQuery(fieldName string) string {
	if len(m.subs) == 1 {
		return fmt.Sprintf(`-%s:"%s"`, fieldName, m.subs[0])
	}
	quoted := quoteStrings(m.subs)
	return fmt.Sprintf(`-%s:(%s)`, fieldName, strings.Join(quoted, " OR "))
}

func (m *notContainsAnyMatcher) ToMonitoringFieldFilter(fieldName string) string {
	parts := make([]string, len(m.subs))
	for i, s := range m.subs {
		parts[i] = fmt.Sprintf(`%s != has_substring("%s")`, fieldName, s)
	}
	if len(parts) == 1 {
		return parts[0]
	}
	return "(" + strings.Join(parts, " AND ") + ")"
}

func (m *notContainsAnyMatcher) ToLoggingLogIDQuery() string {
	return ""
}

func (m *notContainsAnyMatcher) ToMonitoringLogIDFilter() string {
	return ""
}

// FromSetFilter converts a KHI SetFilterParseResult into a ValueMatcher.
func FromSetFilter(filter *gcpqueryutil.SetFilterParseResult, isSubstring bool) ValueMatcher {
	if filter == nil || filter.ValidationError != "" {
		return nil
	}
	if filter.SubtractMode {
		if len(filter.Subtractives) == 0 {
			return nil
		}
		if isSubstring {
			return NotContainsAny(filter.Subtractives...)
		}
		return NoneOf(filter.Subtractives...)
	}
	if len(filter.Additives) == 0 {
		return nil
	}
	if isSubstring {
		return ContainsAny(filter.Additives...)
	}
	return OneOf(filter.Additives...)
}

func quoteStrings(vals []string) []string {
	quoted := make([]string, len(vals))
	for i, v := range vals {
		quoted[i] = fmt.Sprintf(`"%s"`, v)
	}
	return quoted
}

func filterNonEmpty(vals []string) []string {
	var res []string
	for _, v := range vals {
		trimmed := strings.TrimSpace(v)
		if trimmed != "" {
			res = append(res, trimmed)
		}
	}
	return res
}
