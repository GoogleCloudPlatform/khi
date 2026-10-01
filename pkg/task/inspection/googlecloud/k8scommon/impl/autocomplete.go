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

package k8scommon_impl

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// autocompleteMetricsK8sContainerTask is the task to provide the default metrics type to collect the cluster names.
// The resource type "k8s_container" must be available on the returned metrics type.
// This task is overridden in GKE clusters.
// logging.googleapis.com/log_entry_count is better from the perspective of KHI's purpose, but use container metrics for longer retention period(24 months).
var autocompleteMetricsK8sContainerTask = coretask.DefineConstant(k8scommon.AutocompleteMetricsK8sContainerTaskID, "kubernetes.io/anthos/up")

// autocompleteMetricsK8sNodeTask provides the default metrics type to collect node names.
var autocompleteMetricsK8sNodeTask = coretask.DefineConstant(k8scommon.AutocompleteMetricsK8sNodeTaskID, "kubernetes.io/anthos/up")

// autocompleteClusterIdentityTask collects cluster name candidates as GoogleCloudClusterIdentity.
var autocompleteClusterIdentityTask = inspectiontaskbase.DefineCachedTask(k8scommon.AutocompleteClusterIdentityTaskID, func(b *coretask.Binder) inspectiontaskbase.CachedTaskSpec[*inspectioncore.AutocompleteResult[k8scommon.GoogleCloudClusterIdentity]] {
	prefixPolicy := coretask.Use(b, k8scommon.ClusterNamePrefixTaskRef)
	projectID := coretask.Use(b, gcpcommon.InputProjectIdTaskID.Ref())
	startTime := coretask.Use(b, gcpcommon.InputStartTimeTaskID.Ref())
	endTime := coretask.Use(b, gcpcommon.InputEndTimeTaskID.Ref())
	metricsType := coretask.Use(b, k8scommon.AutocompleteMetricsK8sContainerTaskID.Ref())
	clientFactory := coretask.Use(b, gcpcommon.APIClientFactoryTaskID.Ref())
	optionInjector := coretask.Use(b, gcpcommon.APIClientCallOptionsInjectorTaskID.Ref())

	return inspectiontaskbase.CachedTaskSpec[*inspectioncore.AutocompleteResult[k8scommon.GoogleCloudClusterIdentity]]{
		Scope: inspectiontaskbase.CacheScopeGlobal,
		InputDigest: func(ctx context.Context) string {
			return fmt.Sprintf("%s-%s-%d-%d", prefixPolicy.Get(ctx).PrefixFor(k8scommon.ClusterNameUsageK8sCluster), projectID.Get(ctx), startTime.Get(ctx).Unix(), endTime.Get(ctx).Unix())
		},
		Compute: func(ctx context.Context) (*inspectioncore.AutocompleteResult[k8scommon.GoogleCloudClusterIdentity], error) {
			pid := projectID.Get(ctx)
			if pid == "" {
				return &inspectioncore.AutocompleteResult[k8scommon.GoogleCloudClusterIdentity]{
					Values: []k8scommon.GoogleCloudClusterIdentity{},
					Error:  "",
					Hint:   "Cluster names are suggested after the project ID is provided.",
				}, nil
			}

			errorString := ""
			hintString := ""
			st := startTime.Get(ctx)
			et := endTime.Get(ctx)
			if et.Before(time.Now().Add(-time.Hour * 24 * 30 * 24)) {
				hintString = "The end time is more than 24 months ago. Suggested cluster names may not be complete."
			}

			client, err := clientFactory.Get(ctx).MonitoringMetricClient(ctx, googlecloud.Project(pid))
			if err != nil {
				return nil, fmt.Errorf("failed to create monitoring metric client: %w", err)
			}

			callCtx := optionInjector.Get(ctx).InjectToCallContext(ctx, googlecloud.Project(pid))
			filter := fmt.Sprintf(`metric.type="%s" AND resource.type="k8s_container"`, metricsType.Get(ctx))
			metricsLabels, err := googlecloud.QueryResourceLabelsFromMetrics(callCtx, client, pid, filter, st, et, []string{"resource.label.cluster_name", "resource.label.location"})
			if err != nil {
				errorString = err.Error()
			}
			policy := prefixPolicy.Get(ctx)
			metricsLabels = filterAndTrimPrefixFromClusterNames(metricsLabels, policy.PrefixFor(k8scommon.ClusterNameUsageK8sCluster))
			if hintString == "" && errorString == "" && len(metricsLabels) == 0 {
				hintString = fmt.Sprintf("No cluster names found between %s and %s. It is highly likely that the time range is incorrect. Please verify the time range, or proceed by manually entering the cluster name.", st.Format(time.RFC3339), et.Format(time.RFC3339))
			}

			identities := make([]k8scommon.GoogleCloudClusterIdentity, len(metricsLabels))
			for i, labels := range metricsLabels {
				identities[i] = k8scommon.GoogleCloudClusterIdentity{
					ProjectID:    pid,
					PrefixPolicy: policy,
					ClusterName:  labels["cluster_name"],
					Location:     labels["location"],
				}
			}

			return &inspectioncore.AutocompleteResult[k8scommon.GoogleCloudClusterIdentity]{
				Values: identities,
				Error:  errorString,
				Hint:   hintString,
			}, nil
		},
	}
})

// filterAndTrimPrefixFromClusterNames filters cluster names by prefix and trims the prefix from the filtered cluster names.
func filterAndTrimPrefixFromClusterNames(metricsLabels []map[string]string, prefix string) []map[string]string {
	filteredClusters := make([]map[string]string, 0, len(metricsLabels))
	for _, labels := range metricsLabels {
		clusterName := labels["cluster_name"]
		if prefix == "" {
			if !strings.Contains(clusterName, "/") {
				filteredClusters = append(filteredClusters, labels)
			}
		} else if strings.HasPrefix(clusterName, prefix) {
			labels["cluster_name"] = strings.TrimPrefix(clusterName, prefix)
			filteredClusters = append(filteredClusters, labels)
		}
	}
	return filteredClusters
}

// autocompleteLocationForClusterTask returns the location for the given cluster name.
var autocompleteLocationForClusterTask = inspectiontaskbase.DefineCachedTask(k8scommon.AutocompleteLocationForClusterTaskID, func(b *coretask.Binder) inspectiontaskbase.CachedTaskSpec[*inspectioncore.AutocompleteResult[string]] {
	clusterName := coretask.Use(b, k8scommon.InputClusterNameTaskID.Ref()) // This task must not depend on ClusterIdentity because this autocomplete will generate the source of it.
	projectID := coretask.Use(b, gcpcommon.InputProjectIdTaskID.Ref())
	startTime := coretask.Use(b, gcpcommon.InputStartTimeTaskID.Ref())
	endTime := coretask.Use(b, gcpcommon.InputEndTimeTaskID.Ref())
	clusterIdentities := coretask.Use(b, k8scommon.AutocompleteClusterIdentityTaskID.Ref())

	return inspectiontaskbase.CachedTaskSpec[*inspectioncore.AutocompleteResult[string]]{
		Scope: inspectiontaskbase.CacheScopeGlobal,
		InputDigest: func(ctx context.Context) string {
			return fmt.Sprintf("%s-%s-%d-%d", clusterName.Get(ctx), projectID.Get(ctx), startTime.Get(ctx).Unix(), endTime.Get(ctx).Unix())
		},
		Compute: func(ctx context.Context) (*inspectioncore.AutocompleteResult[string], error) {
			pid := projectID.Get(ctx)
			if pid == "" {
				return &inspectioncore.AutocompleteResult[string]{
					Values: []string{},
					Error:  "",
					Hint:   "Locations will be suggested after the project ID is provided.",
				}, nil
			}
			ci := clusterIdentities.Get(ctx)
			if ci.Error != "" {
				return &inspectioncore.AutocompleteResult[string]{
					Values: []string{},
					Error:  ci.Error,
					Hint:   ci.Hint,
				}, nil
			}
			cName := clusterName.Get(ctx)
			if cName == "" {
				return &inspectioncore.AutocompleteResult[string]{
					Values: []string{},
					Error:  "",
					Hint:   "Locations will be suggested after the cluster name is provided.",
				}, nil
			}
			result := &inspectioncore.AutocompleteResult[string]{
				Values: []string{},
				Error:  "",
				Hint:   "",
			}

			// Limit the location to the items which has the same cluster name.
			for _, identity := range ci.Values {
				if identity.ClusterName == cName {
					result.Values = append(result.Values, identity.Location)
				}
			}
			return result, nil
		},
	}
}, coretask.WithSelectionPriority(500))

// clusterScopedAutocompleteConfig defines parameters for querying cluster-scoped autocomplete suggestions from Cloud Monitoring metrics.
type clusterScopedAutocompleteConfig struct {
	resourceType       string
	resourceLabelKey   string
	targetNameSingular string
	targetNamePlural   string
}

// defineClusterScopedAutocompleteTask defines a cached task for cluster-scoped autocomplete suggestions.
func defineClusterScopedAutocompleteTask(
	id taskid.TaskImplementationID[*inspectioncore.AutocompleteResult[string]],
	metricsTypeRef taskid.TaskReference[string],
	cfg clusterScopedAutocompleteConfig,
) coretask.DefinedTask[*inspectioncore.AutocompleteResult[string]] {
	return inspectiontaskbase.DefineCachedTask(id, func(b *coretask.Binder) inspectiontaskbase.CachedTaskSpec[*inspectioncore.AutocompleteResult[string]] {
		cluster := coretask.Use(b, k8scommon.ClusterIdentityTaskID.Ref())
		startTime := coretask.Use(b, gcpcommon.InputStartTimeTaskID.Ref())
		endTime := coretask.Use(b, gcpcommon.InputEndTimeTaskID.Ref())
		clientFactory := coretask.Use(b, gcpcommon.APIClientFactoryTaskID.Ref())
		optionInjector := coretask.Use(b, gcpcommon.APIClientCallOptionsInjectorTaskID.Ref())
		metricsType := coretask.Use(b, metricsTypeRef)

		return inspectiontaskbase.CachedTaskSpec[*inspectioncore.AutocompleteResult[string]]{
			Scope: inspectiontaskbase.CacheScopeGlobal,
			InputDigest: func(ctx context.Context) string {
				c := cluster.Get(ctx)
				s := startTime.Get(ctx)
				e := endTime.Get(ctx)
				return fmt.Sprintf("%s-%d-%d", c.UniqueDigest(), s.Unix(), e.Unix())
			},
			Compute: func(ctx context.Context) (*inspectioncore.AutocompleteResult[string], error) {
				return queryClusterScopedAutocompleteMetrics(
					ctx,
					cluster.Get(ctx),
					startTime.Get(ctx),
					endTime.Get(ctx),
					clientFactory.Get(ctx),
					optionInjector.Get(ctx),
					metricsType.Get(ctx),
					cfg,
				)
			},
		}
	})
}

// queryClusterScopedAutocompleteMetrics queries cluster-scoped autocomplete suggestions from Cloud Monitoring metrics.
func queryClusterScopedAutocompleteMetrics(
	ctx context.Context,
	cluster k8scommon.GoogleCloudClusterIdentity,
	startTime time.Time,
	endTime time.Time,
	clientFactory *googlecloud.ClientFactory,
	optionInjector *googlecloud.CallOptionInjector,
	metricsType string,
	cfg clusterScopedAutocompleteConfig,
) (*inspectioncore.AutocompleteResult[string], error) {
	if !cluster.IsComplete() {
		capitalizedPlural := strings.ToUpper(cfg.targetNamePlural[:1]) + cfg.targetNamePlural[1:]
		return &inspectioncore.AutocompleteResult[string]{
			Values: []string{},
			Error:  "",
			Hint:   fmt.Sprintf("%s are suggested after the project ID, cluster name, and location are provided.", capitalizedPlural),
		}, nil
	}

	errorString := ""
	hintString := ""
	if endTime.Before(time.Now().Add(-time.Hour * 24 * 30 * 24)) {
		hintString = fmt.Sprintf("The end time is more than 24 months ago. Suggested %s may not be complete.", cfg.targetNamePlural)
	}

	client, err := clientFactory.MonitoringMetricClient(ctx, googlecloud.Project(cluster.ProjectID))
	if err != nil {
		return nil, fmt.Errorf("failed to create monitoring metric client: %w", err)
	}

	callCtx := optionInjector.InjectToCallContext(ctx, googlecloud.Project(cluster.ProjectID))
	filter := fmt.Sprintf(`metric.type="%s" AND resource.type="%s" AND resource.labels.cluster_name="%s" AND resource.labels.location="%s"`, metricsType, cfg.resourceType, cluster.ClusterName, cluster.Location)
	groupByKey := "resource.labels." + cfg.resourceLabelKey
	values, err := googlecloud.QueryDistinctStringLabelValuesFromMetrics(callCtx, client, cluster.ProjectID, filter, startTime, endTime, groupByKey, cfg.resourceLabelKey)
	if err != nil {
		errorString = err.Error()
	}
	if hintString == "" && errorString == "" && len(values) == 0 {
		hintString = fmt.Sprintf("No %s found between %s and %s. It is highly likely that the time range is incorrect. Please verify the time range, or proceed by manually entering the %s.", cfg.targetNamePlural, startTime.Format(time.RFC3339), endTime.Format(time.RFC3339), cfg.targetNameSingular)
	}
	return &inspectioncore.AutocompleteResult[string]{
		Values: values,
		Error:  errorString,
		Hint:   hintString,
	}, nil
}

// autocompleteNamespacesTask provides namespace suggestions for autocomplete.
var autocompleteNamespacesTask = defineClusterScopedAutocompleteTask(
	k8scommon.AutocompleteNamespacesTaskID,
	k8scommon.AutocompleteMetricsK8sContainerTaskID.Ref(),
	clusterScopedAutocompleteConfig{
		resourceType:       "k8s_container",
		resourceLabelKey:   "namespace_name",
		targetNameSingular: "namespace name",
		targetNamePlural:   "namespace names",
	},
)

// autocompletePodNamesTask provides pod name suggestions for autocomplete.
var autocompletePodNamesTask = defineClusterScopedAutocompleteTask(
	k8scommon.AutocompletePodNamesTaskID,
	k8scommon.AutocompleteMetricsK8sContainerTaskID.Ref(),
	clusterScopedAutocompleteConfig{
		resourceType:       "k8s_container",
		resourceLabelKey:   "pod_name",
		targetNameSingular: "pod name",
		targetNamePlural:   "pod names",
	},
)

// autocompleteNodeNamesTask provides node name suggestions for autocomplete.
var autocompleteNodeNamesTask = defineClusterScopedAutocompleteTask(
	k8scommon.AutocompleteNodeNamesTaskID,
	k8scommon.AutocompleteMetricsK8sNodeTaskID.Ref(),
	clusterScopedAutocompleteConfig{
		resourceType:       "k8s_node",
		resourceLabelKey:   "node_name",
		targetNameSingular: "node name",
		targetNamePlural:   "node names",
	},
)
