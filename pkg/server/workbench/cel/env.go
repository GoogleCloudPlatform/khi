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

package cel

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"
)

func extractStrings(val ref.Val) ([]string, error) {
	if s, ok := val.(types.String); ok {
		return []string{string(s)}, nil
	}
	if list, ok := val.(traits.Lister); ok {
		native, err := list.ConvertToNative(reflect.TypeOf([]string{}))
		if err != nil {
			return nil, err
		}
		return native.([]string), nil
	}
	return nil, fmt.Errorf("expected string or list<string>, got %v", val.Type())
}

// TimelineEvaluator compiles and executes CEL expressions on TimelineData.
type TimelineEvaluator struct {
	mu              sync.Mutex
	env             *cel.Env
	program         cel.Program
	currentTimeline *TimelineData
}

// NewTimelineEvaluator creates a new TimelineEvaluator.
func NewTimelineEvaluator() (*TimelineEvaluator, error) {
	eval := &TimelineEvaluator{}

	matchBindingBinary := func(lhs, rhs ref.Val) ref.Val {
		key, ok := lhs.(types.String)
		if !ok {
			return types.False
		}
		patterns, err := extractStrings(rhs)
		if err != nil {
			return types.False
		}
		return types.Bool(MatchTimelinePath(eval.currentTimeline, string(key), patterns))
	}

	matchBindingUnary := func(arg ref.Val) ref.Val {
		patterns, err := extractStrings(arg)
		if err != nil {
			return types.False
		}
		return types.Bool(MatchTimelinePath(eval.currentTimeline, "*", patterns))
	}

	rbBindingBinary := func(lhs, rhs ref.Val) ref.Val {
		pathKey, ok := lhs.(types.String)
		if !ok {
			return types.False
		}
		patterns, err := extractStrings(rhs)
		if err != nil {
			return types.False
		}
		return types.Bool(MatchTimelineRevisionBodyField(eval.currentTimeline, string(pathKey), patterns))
	}

	rbBindingUnary := func(arg ref.Val) ref.Val {
		patterns, err := extractStrings(arg)
		if err != nil {
			return types.False
		}
		return types.Bool(MatchTimelineRevisionBodyField(eval.currentTimeline, "*", patterns))
	}

	minSeverityBinding := func(arg ref.Val) ref.Val {
		if eval.currentTimeline == nil {
			return types.False
		}
		minOrder, ok := arg.(types.Int)
		if !ok {
			return types.False
		}
		return types.Bool(eval.currentTimeline.MaxSeverity >= uint32(minOrder))
	}

	env, err := cel.NewEnv(
		cel.Variable("t", cel.MapType(cel.StringType, cel.DynType)),
		cel.Variable("name", cel.StringType),
		cel.Variable("timelineType", cel.StringType),
		cel.Variable("path", cel.MapType(cel.StringType, cel.StringType)),
		cel.Variable("events", cel.ListType(cel.MapType(cel.StringType, cel.DynType))),
		cel.Variable("revisions", cel.ListType(cel.MapType(cel.StringType, cel.DynType))),
		cel.Variable("UNKNOWN", cel.IntType),
		cel.Variable("INFO", cel.IntType),
		cel.Variable("WARNING", cel.IntType),
		cel.Variable("ERROR", cel.IntType),
		cel.Variable("FATAL", cel.IntType),
		// match functions
		cel.Function("match",
			cel.Overload("match_string_string", []*cel.Type{cel.StringType, cel.StringType}, cel.BoolType,
				cel.BinaryBinding(matchBindingBinary),
			),
			cel.Overload("match_string_list", []*cel.Type{cel.StringType, cel.ListType(cel.StringType)}, cel.BoolType,
				cel.BinaryBinding(matchBindingBinary),
			),
			cel.Overload("match_string", []*cel.Type{cel.StringType}, cel.BoolType,
				cel.UnaryBinding(matchBindingUnary),
			),
			cel.Overload("match_list", []*cel.Type{cel.ListType(cel.StringType)}, cel.BoolType,
				cel.UnaryBinding(matchBindingUnary),
			),
		),
		cel.Function("M",
			cel.Overload("m_string_string", []*cel.Type{cel.StringType, cel.StringType}, cel.BoolType,
				cel.BinaryBinding(matchBindingBinary),
			),
			cel.Overload("m_string_list", []*cel.Type{cel.StringType, cel.ListType(cel.StringType)}, cel.BoolType,
				cel.BinaryBinding(matchBindingBinary),
			),
			cel.Overload("m_string", []*cel.Type{cel.StringType}, cel.BoolType,
				cel.UnaryBinding(matchBindingUnary),
			),
			cel.Overload("m_list", []*cel.Type{cel.ListType(cel.StringType)}, cel.BoolType,
				cel.UnaryBinding(matchBindingUnary),
			),
		),
		// revision_body functions
		cel.Function("revision_body",
			cel.Overload("revision_body_string_string", []*cel.Type{cel.StringType, cel.StringType}, cel.BoolType,
				cel.BinaryBinding(rbBindingBinary),
			),
			cel.Overload("revision_body_string_list", []*cel.Type{cel.StringType, cel.ListType(cel.StringType)}, cel.BoolType,
				cel.BinaryBinding(rbBindingBinary),
			),
			cel.Overload("revision_body_string", []*cel.Type{cel.StringType}, cel.BoolType,
				cel.UnaryBinding(rbBindingUnary),
			),
			cel.Overload("revision_body_list", []*cel.Type{cel.ListType(cel.StringType)}, cel.BoolType,
				cel.UnaryBinding(rbBindingUnary),
			),
		),
		cel.Function("RB",
			cel.Overload("rb_string_string", []*cel.Type{cel.StringType, cel.StringType}, cel.BoolType,
				cel.BinaryBinding(rbBindingBinary),
			),
			cel.Overload("rb_string_list", []*cel.Type{cel.StringType, cel.ListType(cel.StringType)}, cel.BoolType,
				cel.BinaryBinding(rbBindingBinary),
			),
			cel.Overload("rb_string", []*cel.Type{cel.StringType}, cel.BoolType,
				cel.UnaryBinding(rbBindingUnary),
			),
			cel.Overload("rb_list", []*cel.Type{cel.ListType(cel.StringType)}, cel.BoolType,
				cel.UnaryBinding(rbBindingUnary),
			),
		),
		// minSeverity function
		cel.Function("minSeverity",
			cel.Overload("minSeverity_int", []*cel.Type{cel.IntType}, cel.BoolType,
				cel.UnaryBinding(minSeverityBinding),
			),
		),
	)
	if err != nil {
		return nil, err
	}

	eval.env = env
	return eval, nil
}

// Compile parses and compiles the CEL expression string. If expr is empty, evaluation always passes.
func (e *TimelineEvaluator) Compile(expr string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if expr == "" {
		e.program = nil
		return nil
	}

	ast, iss := e.env.Compile(expr)
	if iss.Err() != nil {
		return iss.Err()
	}

	prg, err := e.env.Program(ast)
	if err != nil {
		return err
	}
	e.program = prg
	return nil
}

// Evaluate evaluates the compiled CEL expression against the provided timeline.
func (e *TimelineEvaluator) Evaluate(ctx context.Context, t *TimelineData) (bool, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.program == nil {
		return true, nil
	}

	e.currentTimeline = t
	defer func() { e.currentTimeline = nil }()

	tVars := map[string]any{
		"name":         t.Name,
		"timelineType": t.TimelineType,
		"path":         t.Path,
		"UNKNOWN":      int64(0),
		"INFO":         int64(1),
		"WARNING":      int64(2),
		"ERROR":        int64(3),
		"FATAL":        int64(4),
	}
	tVars["t"] = tVars

	out, _, err := e.program.Eval(tVars)
	if err != nil {
		return false, err
	}

	if b, ok := out.Value().(bool); ok {
		return b, nil
	}
	return false, nil
}

// LogEvaluator compiles and executes CEL expressions on LogData.
type LogEvaluator struct {
	mu         sync.Mutex
	env        *cel.Env
	program    cel.Program
	currentLog *LogData
}

// NewLogEvaluator creates a new LogEvaluator.
func NewLogEvaluator() (*LogEvaluator, error) {
	eval := &LogEvaluator{}

	bodyBindingBinary := func(lhs, rhs ref.Val) ref.Val {
		pathKey, ok := lhs.(types.String)
		if !ok {
			return types.False
		}
		patterns, err := extractStrings(rhs)
		if err != nil {
			return types.False
		}
		return types.Bool(MatchLogField(eval.currentLog, string(pathKey), patterns))
	}

	bodyBindingUnary := func(arg ref.Val) ref.Val {
		patterns, err := extractStrings(arg)
		if err != nil {
			return types.False
		}
		return types.Bool(MatchLogField(eval.currentLog, "*", patterns))
	}

	env, err := cel.NewEnv(
		cel.Variable("l", cel.MapType(cel.StringType, cel.DynType)),
		cel.Variable("logType", cel.StringType),
		cel.Variable("severity", cel.IntType),
		cel.Variable("summary", cel.StringType),
		cel.Variable("body", cel.MapType(cel.StringType, cel.DynType)),
		cel.Variable("bodyYAML", cel.StringType),
		cel.Variable("UNKNOWN", cel.IntType),
		cel.Variable("INFO", cel.IntType),
		cel.Variable("WARNING", cel.IntType),
		cel.Variable("ERROR", cel.IntType),
		cel.Variable("FATAL", cel.IntType),
		cel.Function("body",
			cel.Overload("body_string_string", []*cel.Type{cel.StringType, cel.StringType}, cel.BoolType,
				cel.BinaryBinding(bodyBindingBinary),
			),
			cel.Overload("body_string_list", []*cel.Type{cel.StringType, cel.ListType(cel.StringType)}, cel.BoolType,
				cel.BinaryBinding(bodyBindingBinary),
			),
			cel.Overload("body_string", []*cel.Type{cel.StringType}, cel.BoolType,
				cel.UnaryBinding(bodyBindingUnary),
			),
			cel.Overload("body_list", []*cel.Type{cel.ListType(cel.StringType)}, cel.BoolType,
				cel.UnaryBinding(bodyBindingUnary),
			),
		),
		cel.Function("B",
			cel.Overload("b_string_string", []*cel.Type{cel.StringType, cel.StringType}, cel.BoolType,
				cel.BinaryBinding(bodyBindingBinary),
			),
			cel.Overload("b_string_list", []*cel.Type{cel.StringType, cel.ListType(cel.StringType)}, cel.BoolType,
				cel.BinaryBinding(bodyBindingBinary),
			),
			cel.Overload("b_string", []*cel.Type{cel.StringType}, cel.BoolType,
				cel.UnaryBinding(bodyBindingUnary),
			),
			cel.Overload("b_list", []*cel.Type{cel.ListType(cel.StringType)}, cel.BoolType,
				cel.UnaryBinding(bodyBindingUnary),
			),
		),
	)
	if err != nil {
		return nil, err
	}

	eval.env = env
	return eval, nil
}

// Compile parses and compiles the CEL expression string. If expr is empty, evaluation always passes.
func (e *LogEvaluator) Compile(expr string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if expr == "" {
		e.program = nil
		return nil
	}

	ast, iss := e.env.Compile(expr)
	if iss.Err() != nil {
		return iss.Err()
	}

	prg, err := e.env.Program(ast)
	if err != nil {
		return err
	}
	e.program = prg
	return nil
}

// Evaluate evaluates the compiled CEL expression against the provided log.
func (e *LogEvaluator) Evaluate(ctx context.Context, l *LogData) (bool, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.program == nil {
		return true, nil
	}

	e.currentLog = l
	defer func() { e.currentLog = nil }()

	lVars := map[string]any{
		"logType":  l.LogType,
		"severity": int64(l.Severity),
		"summary":  l.Summary,
		"body":     l.Body,
		"bodyYAML": l.BodyYAML,
		"UNKNOWN":  int64(0),
		"INFO":     int64(1),
		"WARNING":  int64(2),
		"ERROR":    int64(3),
		"FATAL":    int64(4),
	}
	lVars["l"] = lVars

	out, _, err := e.program.Eval(lVars)
	if err != nil {
		return false, err
	}

	if b, ok := out.Value().(bool); ok {
		return b, nil
	}
	return false, nil
}

// ValidateTimelineQuery validates a CEL timeline expression syntax and types.
func ValidateTimelineQuery(query string) error {
	if query == "" {
		return nil
	}
	eval, err := NewTimelineEvaluator()
	if err != nil {
		return err
	}
	return eval.Compile(query)
}

// ValidateLogQuery validates a CEL log expression syntax and types.
func ValidateLogQuery(query string) error {
	if query == "" {
		return nil
	}
	eval, err := NewLogEvaluator()
	if err != nil {
		return err
	}
	return eval.Compile(query)
}
