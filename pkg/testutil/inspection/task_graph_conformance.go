// Copyright 2025 Google LLC
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

package inspection_test

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/logger"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/generated"
	"github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6/style"
	"github.com/GoogleCloudPlatform/khi/pkg/server/upload"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// RunTaskGraphConformance runs the full suite of automated conformance tests across all registered tasks.
func RunTaskGraphConformance(t *testing.T) {
	logger.InitGlobalKHILogger()
	oldStore := upload.DefaultUploadFileStore
	upload.DefaultUploadFileStore = upload.NewUploadFileStore(upload.NewLocalUploadFileStoreProvider(t.TempDir()))
	t.Cleanup(func() {
		upload.DefaultUploadFileStore = oldStore
	})

	ioConfig, err := inspectioncore_contract.NewIOConfigForTest()
	if err != nil {
		t.Fatalf("unexpected error creating IOConfig: %v", err)
	}
	server, err := coreinspection.NewServer(ioConfig)
	if err != nil {
		t.Fatalf("unexpected error creating server: %v", err)
	}
	if err := generated.RegisterAllInspectionTasks(server); err != nil {
		t.Fatalf("failed to register inspection tasks: %v", err)
	}
	style.LockRegistry()

	allTasks := server.GetAllRegisteredTasks()

	// Layer 1: Global static conformance checks on all registered tasks.
	t.Run("Layer1_GlobalStaticConformance", func(t *testing.T) {
		runGlobalStaticConformance(t, allTasks)
	})

	reachedTaskImplIDs := make(map[string]struct{})
	completedInspectionTypeIDs := make(map[string]struct{})

	// Layer 2, 3, 4: InspectionType-specific conformance checks.
	for _, it := range server.GetAllInspectionTypes() {
		t.Run(fmt.Sprintf("InspectionType_%s", it.Id), func(t *testing.T) {
			t.Run("Layer2_InspectionTypeConformance", func(t *testing.T) {
				runInspectionTypeConformance(t, server, it)
			})

			t.Run("Layer3_FeatureCombinations", func(t *testing.T) {
				runFeatureCombinationsConformance(t, server, it, reachedTaskImplIDs)
				completedInspectionTypeIDs[it.Id] = struct{}{}
			})

			t.Run("Layer4_FormTaskAndTypeContracts", func(t *testing.T) {
				runFormTaskAndTypeContracts(t, server, it)
			})
		})
	}

	// Layer 5: Reachability check across all inspection types.
	t.Run("Layer5_Reachability", func(t *testing.T) {
		if len(completedInspectionTypeIDs) < len(server.GetAllInspectionTypes()) || len(reachedTaskImplIDs) == 0 {
			t.Skip("skipping reachability check: feature resolution was not evaluated for all inspection types (subtest filter applied)")
		}
		runReachabilityConformance(t, allTasks, reachedTaskImplIDs)
	})
}

// runGlobalStaticConformance validates global invariants on all registered tasks.
func runGlobalStaticConformance(t *testing.T, allTasks []coretask.UntypedTask) {
	t.Run("NoDanglingReferences", func(t *testing.T) {
		registeredRefs := make(map[string]struct{}, len(allTasks))
		for _, task := range allTasks {
			registeredRefs[task.UntypedID().ReferenceIDString()] = struct{}{}
		}

		for _, task := range allTasks {
			for _, dep := range task.Dependencies() {
				if ptp, ok := dep.(taskid.PointToPointDescriptor); ok {
					refID := ptp.ReferenceID()
					if _, exists := registeredRefs[refID]; !exists {
						t.Errorf("task %q depends on unregistered reference ID %q", task.UntypedID().String(), refID)
					}
				}
			}
		}
	})

	t.Run("ResultTypeConsistencyPerReferenceID", func(t *testing.T) {
		variantsByRef := make(map[string][]coretask.UntypedTask)
		for _, task := range allTasks {
			refID := task.UntypedID().ReferenceIDString()
			variantsByRef[refID] = append(variantsByRef[refID], task)
		}

		for refID, variants := range variantsByRef {
			if len(variants) <= 1 {
				continue
			}
			firstType := variants[0].ResultType()
			for _, variant := range variants[1:] {
				if variant.ResultType() != firstType {
					t.Errorf("reference ID %q has conflicting ResultType: %q returns %v, but %q returns %v",
						refID, variants[0].UntypedID().String(), firstType, variant.UntypedID().String(), variant.ResultType())
				}
			}
		}
	})

	t.Run("NoSelfDependency", func(t *testing.T) {
		for _, task := range allTasks {
			taskRef := task.UntypedID().ReferenceIDString()
			for _, dep := range task.Dependencies() {
				if ptp, ok := dep.(taskid.PointToPointDescriptor); ok {
					if ptp.ReferenceID() == taskRef {
						t.Errorf("task %q has self-dependency on its own reference ID %q", task.UntypedID().String(), taskRef)
					}
				}
			}
		}
	})
}

// runInspectionTypeConformance validates type-level compatibility and priority uniqueness.
func runInspectionTypeConformance(t *testing.T, server *coreinspection.InspectionTaskServer, it *coreinspection.InspectionType) {
	availableTasks := getAvailableTasksForInspectionType(server, it)
	availableMap := make(map[string]coretask.UntypedTask, len(availableTasks))
	for _, task := range availableTasks {
		availableMap[task.UntypedID().ReferenceIDString()] = task
	}

	t.Run("MandatoryDependencyCompleteness", func(t *testing.T) {
		for _, task := range availableTasks {
			for _, dep := range task.Dependencies() {
				if dep.DescriptorScope() == taskid.ScopeAll && dep.DescriptorCardinality() == taskid.CardinalityPointToPoint {
					ptp, ok := dep.(taskid.PointToPointDescriptor)
					if !ok {
						continue
					}
					if _, exists := availableMap[ptp.ReferenceID()]; !exists {
						t.Errorf("task %q requires mandatory dependency %q which is missing from available tasks",
							task.UntypedID().String(), ptp.ReferenceID())
					}
				}
			}
		}
	})

	t.Run("NoPriorityCollision", func(t *testing.T) {
		var compatibleTasks []coretask.UntypedTask
		for _, task := range server.GetAllRegisteredTasks() {
			if compat, _ := coreinspection.EvaluateTaskCompatibility(task, it); compat {
				compatibleTasks = append(compatibleTasks, task)
			}
		}

		compatByRef := make(map[string][]coretask.UntypedTask)
		for _, task := range compatibleTasks {
			refID := task.UntypedID().ReferenceIDString()
			compatByRef[refID] = append(compatByRef[refID], task)
		}

		for refID, variants := range compatByRef {
			if len(variants) <= 1 {
				continue
			}
			priorityMap := make(map[int]coretask.UntypedTask)
			for _, v := range variants {
				p := typedmap.GetOrDefault(v.Labels(), coretask.LabelKeyTaskSelectionPriority, 0)
				if existing, exists := priorityMap[p]; exists {
					t.Errorf("reference ID %q has ambiguous priority collision: %q and %q both have priority %d",
						refID, existing.UntypedID().String(), v.UntypedID().String(), p)
				} else {
					priorityMap[p] = v
				}
			}
		}
	})

	t.Run("RequiredTasksPresent", func(t *testing.T) {
		for _, task := range server.GetAllRegisteredTasks() {
			if typedmap.GetOrDefault(task.Labels(), coretask.LabelKeyRequiredTask, false) {
				if compat, _ := coreinspection.EvaluateTaskCompatibility(task, it); compat {
					refID := task.UntypedID().ReferenceIDString()
					if _, exists := availableMap[refID]; !exists {
						t.Errorf("required task %q is compatible with %q but missing from available tasks",
							task.UntypedID().String(), it.Id)
					}
				}
			}
		}
	})
}

// runFeatureCombinationsConformance tests resolving the task graph under various feature selections.
func runFeatureCombinationsConformance(
	t *testing.T,
	server *coreinspection.InspectionTaskServer,
	it *coreinspection.InspectionType,
	reachedTaskImplIDs map[string]struct{},
) {
	inspectionID, err := server.CreateInspection(it.Id)
	if err != nil {
		t.Fatalf("failed to create inspection for %q: %v", it.Id, err)
	}
	runner := server.GetInspection(inspectionID)
	features, err := runner.FeatureList()
	if err != nil {
		t.Fatalf("failed to get feature list for %q: %v", it.Id, err)
	}

	allFeatureIDs := make([]string, len(features))
	for i, f := range features {
		allFeatureIDs[i] = f.Id
	}

	resolve := func(t *testing.T, featureIDs []string, configName string) {
		if err := runner.SetFeatureList(featureIDs); err != nil {
			t.Errorf("failed to set feature list for %s: %v", configName, err)
			return
		}
		taskSet, err := runner.ResolveTaskGraph()
		if err != nil {
			t.Errorf("failed to resolve task graph for %s: %v", configName, err)
			return
		}
		for _, task := range taskSet.GetAll() {
			reachedTaskImplIDs[task.UntypedID().String()] = struct{}{}
		}
	}

	t.Run("DefaultFeatures", func(t *testing.T) {
		availableTasks := getAvailableTasksForInspectionType(server, it)
		var defaultIDs []string
		for _, task := range availableTasks {
			if typedmap.GetOrDefault(task.Labels(), inspectioncore_contract.LabelKeyInspectionDefaultFeatureFlag, false) {
				defaultIDs = append(defaultIDs, task.UntypedID().String())
			}
		}
		resolve(t, defaultIDs, "default-features")
	})

	t.Run("AllFeaturesDisabled", func(t *testing.T) {
		resolve(t, []string{}, "all-features-disabled")
	})

	t.Run("AllFeaturesEnabled", func(t *testing.T) {
		resolve(t, allFeatureIDs, "all-features-enabled")
	})

	t.Run("SingleFeatureIsolation", func(t *testing.T) {
		for _, f := range features {
			t.Run(f.Id, func(t *testing.T) {
				resolve(t, []string{f.Id}, fmt.Sprintf("single-feature-%s", f.Id))
			})
		}
	})

	t.Run("AllExceptOneFeature", func(t *testing.T) {
		for _, f := range features {
			t.Run(fmt.Sprintf("without-%s", f.Id), func(t *testing.T) {
				remaining := make([]string, 0, len(allFeatureIDs)-1)
				for _, id := range allFeatureIDs {
					if id != f.Id {
						remaining = append(remaining, id)
					}
				}
				resolve(t, remaining, fmt.Sprintf("without-%s", f.Id))
			})
		}
	})

	t.Run("PairwiseFeatures", func(t *testing.T) {
		for i := 0; i < len(features); i++ {
			for j := i + 1; j < len(features); j++ {
				pair := []string{features[i].Id, features[j].Id}
				testName := fmt.Sprintf("%s+AND+%s", features[i].Id, features[j].Id)
				t.Run(testName, func(t *testing.T) {
					resolve(t, pair, testName)
				})
			}
		}
	})

	// If feature count is small (<= 8), test all 2^N combinations exhaustively.
	if len(features) <= 8 && len(features) > 2 {
		t.Run("ExhaustivePowerset", func(t *testing.T) {
			n := len(features)
			totalCombinations := 1 << n
			for mask := 0; mask < totalCombinations; mask++ {
				var subset []string
				for bit := 0; bit < n; bit++ {
					if (mask & (1 << bit)) != 0 {
						subset = append(subset, features[bit].Id)
					}
				}
				resolve(t, subset, fmt.Sprintf("powerset-mask-%08b", mask))
			}
		})
	}
}

// runFormTaskAndTypeContracts validates form task parameters and producer-consumer ResultType contracts.
func runFormTaskAndTypeContracts(t *testing.T, server *coreinspection.InspectionTaskServer, it *coreinspection.InspectionType) {
	availableTasks := getAvailableTasksForInspectionType(server, it)

	t.Run("FormFieldLabelUniqueness", func(t *testing.T) {
		formFieldsByLabel := make(map[string]coretask.UntypedTask)
		for _, task := range availableTasks {
			if isFormTask := typedmap.GetOrDefault(task.Labels(), inspectioncore_contract.TaskLabelKeyIsFormTask, false); isFormTask {
				label := typedmap.GetOrDefault(task.Labels(), inspectioncore_contract.TaskLabelKeyFormFieldLabel, "")
				if label == "" {
					t.Errorf("form task %q is missing a form field label", task.UntypedID().String())
					continue
				}
				if existing, exists := formFieldsByLabel[label]; exists {
					t.Errorf("duplicate form field label %q: implemented by both %q and %q",
						label, existing.UntypedID().String(), task.UntypedID().String())
				} else {
					formFieldsByLabel[label] = task
				}
			}
		}
	})

	t.Run("ResultTypeAssignmentCompatibility", func(t *testing.T) {
		tasksByRef := make(map[string]coretask.UntypedTask)
		for _, task := range availableTasks {
			tasksByRef[task.UntypedID().ReferenceIDString()] = task
		}

		for _, task := range availableTasks {
			for _, dep := range task.Dependencies() {
				if ptp, ok := dep.(taskid.PointToPointDescriptor); ok {
					expectedType := dep.ResultType()
					if expectedType == nil {
						continue
					}
					producer, exists := tasksByRef[ptp.ReferenceID()]
					if !exists {
						continue
					}
					producerType := producer.ResultType()
					if producerType != nil && !producerType.AssignableTo(expectedType) {
						t.Errorf("type mismatch: task %q expects %v for dependency %q, but producer %q returns %v",
							task.UntypedID().String(), expectedType, ptp.ReferenceID(), producer.UntypedID().String(), producerType)
					}
				}
			}
		}
	})

	t.Run("TagFanInTypeCompatibility", func(t *testing.T) {
		for _, task := range availableTasks {
			for _, dep := range task.Dependencies() {
				fanInDesc, ok := dep.(taskid.FanInDescriptor)
				if !ok {
					continue
				}
				expectedType := dep.ResultType()
				if expectedType == nil {
					continue
				}
				tagKey := coretask.LabelKeyProvidedTag(fanInDesc.Tag())
				for _, candidate := range availableTasks {
					if typedmap.GetOrDefault(candidate.Labels(), tagKey, false) {
						candidateType := candidate.ResultType()
						if candidateType != nil && !candidateType.AssignableTo(expectedType) {
							t.Errorf("tag type mismatch: task %q expects %v for tag %q, but producer %q returns %v",
								task.UntypedID().String(), expectedType, fanInDesc.Tag(), candidate.UntypedID().String(), candidateType)
						}
					}
				}
			}
		}
	})
}

// knownFallbackTaskImplIDs records task implementations that serve as default fallbacks
// in generic packages (e.g. googlecloudcommon), but are intentionally shadowed by higher-priority
// tasks in all currently registered inspection types (such as Kubernetes cluster or Composer types).
var knownFallbackTaskImplIDs = map[string]string{
	"cloud.google.com/common/autocomplete-location#default": "Default generic GCP location autocompleter, shadowed by cluster (priority 500) and composer (priority 1000) autocompleters in all current inspection types.",
	"cloud.google.com/common/location-fetcher#default":      "Underlying dependency of generic autocomplete-location#default, shadowed along with it.",
}

// runReachabilityConformance verifies that all registered tasks are reachable in at least one inspection type's graph.
func runReachabilityConformance(t *testing.T, allTasks []coretask.UntypedTask, reachedTaskImplIDs map[string]struct{}) {
	var unreachableTasks []string
	for _, task := range allTasks {
		implID := task.UntypedID().String()
		if reason, isFallback := knownFallbackTaskImplIDs[implID]; isFallback {
			t.Logf("skipping known fallback task %q: %s", implID, reason)
			continue
		}
		if _, reached := reachedTaskImplIDs[implID]; !reached {
			unreachableTasks = append(unreachableTasks, implID)
		}
	}
	if len(unreachableTasks) > 0 {
		sort.Strings(unreachableTasks)
		for _, id := range unreachableTasks {
			t.Errorf("unreachable task: %q is never included in any resolved task graph across all inspection types and feature configurations", id)
		}
	}
}

// getAvailableTasksForInspectionType returns the deduplicated set of tasks compatible with the inspection type,
// ordered by priority as determined by deduplicateTasksByPriority.
func getAvailableTasksForInspectionType(server *coreinspection.InspectionTaskServer, it *coreinspection.InspectionType) []coretask.UntypedTask {
	var compatible []coretask.UntypedTask
	for _, task := range server.GetAllRegisteredTasks() {
		if ok, _ := coreinspection.EvaluateTaskCompatibility(task, it); ok {
			compatible = append(compatible, task)
		}
	}
	return deduplicateTasksByPriority(compatible)
}

// deduplicateTasksByPriority retains only the task with the highest LabelKeyTaskSelectionPriority for each TaskRef, sorted by reference name.
func deduplicateTasksByPriority(tasks []coretask.UntypedTask) []coretask.UntypedTask {
	bestTaskForRef := map[string]coretask.UntypedTask{}
	for _, task := range tasks {
		refID := task.UntypedID().ReferenceIDString()
		existing, found := bestTaskForRef[refID]
		if !found {
			bestTaskForRef[refID] = task
			continue
		}

		priorityExisting := typedmap.GetOrDefault(existing.Labels(), coretask.LabelKeyTaskSelectionPriority, 0)
		priorityNew := typedmap.GetOrDefault(task.Labels(), coretask.LabelKeyTaskSelectionPriority, 0)
		if priorityNew > priorityExisting {
			bestTaskForRef[refID] = task
		} else if priorityNew == priorityExisting {
			if strings.Compare(task.UntypedID().String(), existing.UntypedID().String()) > 0 {
				bestTaskForRef[refID] = task
			}
		}
	}

	deduplicated := make([]coretask.UntypedTask, 0, len(bestTaskForRef))
	for _, task := range bestTaskForRef {
		deduplicated = append(deduplicated, task)
	}
	slices.SortFunc(deduplicated, func(a, b coretask.UntypedTask) int {
		return strings.Compare(a.UntypedID().ReferenceIDString(), b.UntypedID().ReferenceIDString())
	})
	return deduplicated
}
