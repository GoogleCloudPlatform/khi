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

package k8s

import (
	"reflect"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/log/structure/merger"
	"github.com/GoogleCloudPlatform/khi/pkg/model/k8s/configsource"
	admissionv1 "k8s.io/api/admission/v1"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	certificatesv1 "k8s.io/api/certificates/v1"
	coordinationv1 "k8s.io/api/coordination/v1"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	networkingv1 "k8s.io/api/networking/v1"
	nodev1 "k8s.io/api/node/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type defaultMergeConfigSourceItem struct {
	group     schema.GroupVersion
	Resources []interface{}
}

var defaultMergeConfigSources []defaultMergeConfigSourceItem = []defaultMergeConfigSourceItem{
	{
		group: admissionv1.SchemeGroupVersion,
		Resources: []interface{}{
			admissionv1.AdmissionReview{},
		},
	},
	{
		group: admissionregistrationv1.SchemeGroupVersion,
		Resources: []interface{}{
			admissionregistrationv1.ValidatingWebhookConfiguration{},
			admissionregistrationv1.ValidatingWebhookConfigurationList{},
			admissionregistrationv1.MutatingWebhookConfiguration{},
			admissionregistrationv1.MutatingWebhookConfigurationList{}},
	},
	{
		group: autoscalingv1.SchemeGroupVersion,
		Resources: []interface{}{
			autoscalingv1.HorizontalPodAutoscaler{},
			autoscalingv1.HorizontalPodAutoscalerList{},
			autoscalingv1.Scale{}},
	},
	{
		group: autoscalingv2.SchemeGroupVersion,
		Resources: []interface{}{
			autoscalingv2.HorizontalPodAutoscaler{},
			autoscalingv2.HorizontalPodAutoscalerList{},
		},
	},
	{
		group: certificatesv1.SchemeGroupVersion,
		Resources: []interface{}{
			certificatesv1.CertificateSigningRequest{},
			certificatesv1.CertificateSigningRequestList{},
		},
	},
	{
		group: coordinationv1.SchemeGroupVersion,
		Resources: []interface{}{
			coordinationv1.Lease{},
			coordinationv1.LeaseList{},
		},
	},
	{
		group: corev1.SchemeGroupVersion,
		Resources: []interface{}{
			corev1.Pod{},
			corev1.PodList{},
			corev1.PodStatusResult{},
			corev1.PodTemplate{},
			corev1.PodTemplateList{},
			corev1.ReplicationController{},
			corev1.ReplicationControllerList{},
			corev1.Service{},
			corev1.ServiceProxyOptions{},
			corev1.ServiceList{},
			corev1.Endpoints{},
			corev1.EndpointsList{},
			corev1.Node{},
			corev1.NodeList{},
			corev1.NodeProxyOptions{},
			corev1.Binding{},
			corev1.Event{},
			corev1.EventList{},
			corev1.List{},
			corev1.LimitRange{},
			corev1.LimitRangeList{},
			corev1.ResourceQuota{},
			corev1.ResourceQuotaList{},
			corev1.Namespace{},
			corev1.NamespaceList{},
			corev1.Secret{},
			corev1.SecretList{},
			corev1.ServiceAccount{},
			corev1.ServiceAccountList{},
			corev1.PersistentVolume{},
			corev1.PersistentVolumeList{},
			corev1.PersistentVolumeClaim{},
			corev1.PersistentVolumeClaimList{},
			corev1.PodAttachOptions{},
			corev1.PodLogOptions{},
			corev1.PodExecOptions{},
			corev1.PodPortForwardOptions{},
			corev1.PodProxyOptions{},
			corev1.ComponentStatus{},
			corev1.ComponentStatusList{},
			corev1.SerializedReference{},
			corev1.RangeAllocation{},
			corev1.ConfigMap{},
			corev1.ConfigMapList{},
		},
	},
	{
		group: discoveryv1.SchemeGroupVersion,
		Resources: []interface{}{
			discoveryv1.EndpointSlice{},
			discoveryv1.EndpointSliceList{},
		},
	},
	{
		group: policyv1.SchemeGroupVersion,
		Resources: []interface{}{
			policyv1.PodDisruptionBudget{},
			policyv1.PodDisruptionBudgetList{},
			policyv1.Eviction{},
		},
	},
	{
		group: rbacv1.SchemeGroupVersion,
		Resources: []interface{}{
			rbacv1.Role{},
			rbacv1.RoleBinding{},
			rbacv1.RoleBindingList{},
			rbacv1.RoleList{},

			rbacv1.ClusterRole{},
			rbacv1.ClusterRoleBinding{},
			rbacv1.ClusterRoleBindingList{},
			rbacv1.ClusterRoleList{},
		},
	},
	{
		group: schedulingv1.SchemeGroupVersion,
		Resources: []interface{}{
			schedulingv1.PriorityClass{},
			schedulingv1.PriorityClassList{},
		},
	},
	{
		group: appsv1.SchemeGroupVersion,
		Resources: []interface{}{
			appsv1.Deployment{},
			appsv1.DeploymentList{},
			appsv1.StatefulSet{},
			appsv1.StatefulSetList{},
			appsv1.DaemonSet{},
			appsv1.DaemonSetList{},
			appsv1.ReplicaSet{},
			appsv1.ReplicaSetList{},
			appsv1.ControllerRevision{},
			appsv1.ControllerRevisionList{},
		},
	},
	{
		group: networkingv1.SchemeGroupVersion,
		Resources: []interface{}{
			networkingv1.Ingress{},
			networkingv1.IngressList{},
			networkingv1.IngressClass{},
			networkingv1.IngressClassList{},
			networkingv1.NetworkPolicy{},
			networkingv1.NetworkPolicyList{},
		},
	},
	{
		group: nodev1.SchemeGroupVersion,
		Resources: []interface{}{
			nodev1.RuntimeClass{},
			nodev1.RuntimeClassList{},
		},
	},
	{
		group: storagev1.SchemeGroupVersion,
		Resources: []interface{}{
			storagev1.StorageClass{},
			storagev1.StorageClassList{},
			storagev1.VolumeAttachment{},
			storagev1.VolumeAttachmentList{},
			storagev1.CSINode{},
			storagev1.CSINodeList{},
			storagev1.CSIDriver{},
			storagev1.CSIDriverList{},
			storagev1.CSIStorageCapacity{},
			storagev1.CSIStorageCapacityList{},
		},
	},
}

func GenerateDefaultMergeConfig() (*MergeConfigRegistry, error) {
	type defaultResource struct {
		metav1.ObjectMeta `json:"metadata,omitempty"`
	}
	defaultResovler, err := configsource.FromResourceTypeReflection(defaultResource{})
	if err != nil {
		return nil, err
	}
	registry := &MergeConfigRegistry{
		defaultResolver:      defaultResovler,
		mergeConfigResolvers: make(map[string]*merger.MergeConfigResolver),
	}

	for _, config := range defaultMergeConfigSources {
		apiVersion := config.group.Identifier()
		if apiVersion == "v1" {
			apiVersion = "core/v1"
		}
		for _, resource := range config.Resources {
			refType := reflect.TypeOf(resource)
			kind := strings.ToLower(refType.Name())
			resolver, err := configsource.FromResourceTypeReflection(resource)
			if err != nil {
				return nil, err
			}
			registry.Register(apiVersion, kind, resolver)
		}
	}

	return registry, nil
}
