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

package task

import (
	"context"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"
)

type cachableData struct{}

// Digest implements CachableDependency.
func (*cachableData) Digest() string {
	return "foo-digest"
}

var _ CachableDependency = (*cachableData)(nil)

func TestCachedProcessorStoreValue(t *testing.T) {
	cache := NewLocalTaskVariableCache()
	callCount := 0
	cachableTaskFunc := func(ctx context.Context, taskMode int, v *VariableSet) (any, error) {
		callCount += 1
		return "foo-value", nil
	}
	// First task is the cachable task.
	taskSeries := []struct {
		ShouldUseCache bool
		Tasks          []Definition
	}{
		{
			ShouldUseCache: false,
			Tasks: []Definition{
				NewCachedProcessor("foo", []string{}, cachableTaskFunc),
			},
		},
		{
			ShouldUseCache: true,
			Tasks: []Definition{
				NewCachedProcessor("foo", []string{}, cachableTaskFunc),
			},
		},
		{
			ShouldUseCache: false,
			Tasks: []Definition{
				NewCachedProcessor("bar", []string{"qux", "quux"}, cachableTaskFunc),
				NewProcessorTask("qux", []string{}, func(ctx context.Context, taskMode int, v *VariableSet) (any, error) {
					return "qux-value", nil
				}),
				NewProcessorTask("quux", []string{}, func(ctx context.Context, taskMode int, v *VariableSet) (any, error) {
					return "quux-value", nil
				}),
			},
		},
		{
			ShouldUseCache: false,
			Tasks: []Definition{
				NewCachedProcessor("bar", []string{"qux", "quux"}, cachableTaskFunc),
				NewProcessorTask("qux", []string{}, func(ctx context.Context, taskMode int, v *VariableSet) (any, error) {
					return "qux-value2", nil
				}),
				NewProcessorTask("quux", []string{}, func(ctx context.Context, taskMode int, v *VariableSet) (any, error) {
					return "quux-value", nil
				}),
			},
		},
		{
			ShouldUseCache: true,
			Tasks: []Definition{
				NewCachedProcessor("bar", []string{"qux", "quux"}, cachableTaskFunc),
				NewProcessorTask("qux", []string{}, func(ctx context.Context, taskMode int, v *VariableSet) (any, error) {
					return "qux-value2", nil
				}),
				NewProcessorTask("quux", []string{}, func(ctx context.Context, taskMode int, v *VariableSet) (any, error) {
					return "quux-value", nil
				}),
			},
		},
		{
			ShouldUseCache: true,
			Tasks: []Definition{
				NewCachedProcessor("bar", []string{"qux", "quux"}, cachableTaskFunc),
				NewProcessorTask("qux", []string{}, func(ctx context.Context, taskMode int, v *VariableSet) (any, error) {
					return "qux-value", nil
				}),
				NewProcessorTask("quux", []string{}, func(ctx context.Context, taskMode int, v *VariableSet) (any, error) {
					return "quux-value", nil
				}),
			},
		},
		{
			ShouldUseCache: false,
			Tasks: []Definition{
				NewCachedProcessor("hoge", []string{"fuga"}, cachableTaskFunc),
				NewProcessorTask("fuga", []string{}, func(ctx context.Context, taskMode int, v *VariableSet) (any, error) {
					return &cachableData{}, nil
				}),
			},
		},
		{
			ShouldUseCache: true,
			Tasks: []Definition{
				NewCachedProcessor("hoge", []string{"fuga"}, cachableTaskFunc),
				NewProcessorTask("fuga", []string{}, func(ctx context.Context, taskMode int, v *VariableSet) (any, error) {
					return &cachableData{}, nil
				}),
			},
		},
	}

	for _, series := range taskSeries {
		previousCallCount := callCount
		lr, err := newLocalCachedTaskRunnerForSingleTask(series.Tasks[0], cache, series.Tasks...)
		if err != nil {
			t.Errorf("unexpected error\n%v", err)
		}
		err = lr.Run(context.Background(), 0, map[string]any{})
		if err != nil {
			t.Errorf("unexpected error\n%v", err)
		}
		<-lr.Wait()
		v, err := lr.Result()
		if err != nil {
			t.Errorf("unexpected error\n%v", err)
		}

		storedValue, err := v.Get(series.Tasks[0].ID().String())
		if err != nil {
			t.Errorf("unexpected error\n%v", err)
		}

		if storedValue.(string) != "foo-value" {
			t.Errorf("stored value is not expected value\n%s", storedValue)
		}

		cacheUsed := callCount == previousCallCount

		if cacheUsed && !series.ShouldUseCache {
			t.Errorf("expect the task shouldn't use the cache, but used")
		}
		if !cacheUsed && series.ShouldUseCache {
			t.Errorf("expected the task should use the cache but didn't")
		}
	}
}

func TestCacheProcessorWithMultiThreadNotToCallRunnableMultipleTime(t *testing.T) {
	// cache processor is expected to run a single instance and others should wait
	// if the other processor uses the same cached runner.
	RUNNER_COUNT := 100
	cache := NewLocalTaskVariableCache()
	runners := []*LocalRunner{}
	results := make([]string, RUNNER_COUNT)
	callCount := 0
	task := NewCachedProcessor("foo", []string{}, func(ctx context.Context, taskMode int, v *VariableSet) (any, error) {
		<-time.After(time.Second)
		callCount += 1
		return "foo-value", nil
	})
	for i := 0; i < RUNNER_COUNT; i++ {
		runner, err := newLocalCachedTaskRunnerForSingleTask(task, cache)
		if err != nil {
			t.Errorf("unexpected error\n%v", err)
		}
		runners = append(runners, runner)
	}
	errGrp := errgroup.Group{}
	for i, runner := range runners {
		captureRunner := runner
		captureIndex := i
		errGrp.Go(func() error {
			err := captureRunner.Run(context.Background(), 0, map[string]any{})
			if err != nil {
				return err
			}
			<-captureRunner.Wait()
			vs, err := captureRunner.Result()
			if err != nil {
				return err
			}
			resultAny, err := vs.Get("foo")
			if err != nil {
				return err
			}
			results[captureIndex] = resultAny.(string)
			return nil
		})
	}

	err := errGrp.Wait()
	if err != nil {
		t.Errorf("errgroup end with errors\n%v", err)
	}
	if callCount != 1 {
		t.Errorf("task runnable was called 2 or more times")
	}
}
