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

package k8s_node

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/inspection/ioconfig"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourceinfo/noderesource"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	log_test "github.com/GoogleCloudPlatform/khi/pkg/testutil/log"
	parser_test "github.com/GoogleCloudPlatform/khi/pkg/testutil/parser"

	"github.com/google/go-cmp/cmp"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

func TestParseSummary(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{{
		name: "parse pod status",
		input: `jsonPayload:
  MESSAGE: I0101 22:34:24.745733    1754 kubelet_getters.go:187] "Pod status updated" pod="kube-system/kube-proxy-foo" status=Running`,
		expected: "Pod status updated(status=Running)",
	}, {
		name: "PLEG event",
		input: `jsonPayload:
  MESSAGE: 'I0101 09:35:31.624702    1754 kubelet.go:2375] "SyncLoop (PLEG): event for pod" pod="kube-system/fluentbit-gke-rxrcn" event=&{ID:caecee7e-19ba-4463-aa9d-2a46275c077c Type:ContainerStarted Data:4a4f3a5cc34c7d2f39bdc22690b8a96b1241d4367c5c147f09413c44101755e9}'`,
		expected: "SyncLoop (PLEG): event for pod(ContainerStarted)",
	}, {
		name: "node event",
		input: `jsonPayload:
  MESSAGE: I0101 09:31:25.067393    1761 kubelet_node_status.go:669] "Recording event message for node" node="node name" event="NodeHasSufficientMemory"`,
		expected: "Recording event message for node(NodeHasSufficientMemory)",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			l := log_test.MustLogEntity(tc.input)
			summary, err := parseDefaultSummary(l)
			if err != nil {
				t.Fatal(err)
			}
			if summary != tc.expected {
				t.Errorf("expected %s,\nactual %s", tc.expected, summary)
			}
		})
	}
}

func TestParseRunPodSandboxLog(t *testing.T) {
	testCases := []struct {
		Name     string
		Input    string
		Expected *runPodSandboxLog
	}{
		{
			Name:  "standard run pod log",
			Input: "RunPodSandbox for &PodSandboxMetadata{Name:podname,Uid:b86b49f2431d244c613996c6472eb864,Namespace:kube-system,Attempt:0,} returns sandbox id \"6123c6aacf0c78dc38ec4f0ff72edd3cf04eb82ca0e3e7dddd3950ea9753bdf1\"",
			Expected: &runPodSandboxLog{
				PodName:      "podname",
				PodNamespace: "kube-system",
				PodSandboxID: "6123c6aacf0c78dc38ec4f0ff72edd3cf04eb82ca0e3e7dddd3950ea9753bdf1",
			},
		},
		{
			Name:     "missing container id",
			Input:    "RunPodSandbox for &PodSandboxMetadata{Name:podname,Uid:b86b49f2431d244c613996c6472eb864,Namespace:kube-system,Attempt:0,}",
			Expected: &runPodSandboxLog{PodName: "podname", PodNamespace: "kube-system"},
		},
		{
			Name:     "missing pod name in the struct",
			Input:    "RunPodSandbox for &PodSandboxMetadata{Uid:b86b49f2431d244c613996c6472eb864,Namespace:kube-system,Attempt:0,} returns sandbox id \"6123c6aacf0c78dc38ec4f0ff72edd3cf04eb82ca0e3e7dddd3950ea9753bdf1\"",
			Expected: nil,
		},
		{
			Name:     "missing pod metadata struct",
			Input:    "RunPodSandbox for returns sandbox id \"6123c6aacf0c78dc38ec4f0ff72edd3cf04eb82ca0e3e7dddd3950ea9753bdf1\"",
			Expected: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			sandbox, _ := parseRunPodSandboxLog(tc.Input)
			if diff := cmp.Diff(tc.Expected, sandbox); diff != "" {
				t.Errorf("the result sandbox is not matching with the expected result\n%s", diff)
			}
		})
	}
}

func TestParseCreateContainerLog(t *testing.T) {
	testCases := []struct {
		Name     string
		Input    string
		Expected *createContainerLog
	}{
		{
			Name:  "standard create container log",
			Input: "CreateContainer within sandbox \"e175052cada9b999c5d9fabc8dc2276effc92b564aff74633eee122bcd4c8097\" for &ContainerMetadata{Name:config-init,Attempt:0,} returns container id \"14a996c61131027c75cc9e454acd8244c23ff7ddd236ee4ebbd0dd18d7d637d8\"",
			Expected: &createContainerLog{
				PodSandboxID:  "e175052cada9b999c5d9fabc8dc2276effc92b564aff74633eee122bcd4c8097",
				ContainerID:   "14a996c61131027c75cc9e454acd8244c23ff7ddd236ee4ebbd0dd18d7d637d8",
				ContainerName: "config-init",
			},
		},
		{
			Name:  "standard create container log without container id",
			Input: "CreateContainer within sandbox \"e175052cada9b999c5d9fabc8dc2276effc92b564aff74633eee122bcd4c8097\" for &ContainerMetadata{Name:config-init,Attempt:0,}",
			Expected: &createContainerLog{
				PodSandboxID:  "e175052cada9b999c5d9fabc8dc2276effc92b564aff74633eee122bcd4c8097",
				ContainerID:   "",
				ContainerName: "config-init",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			createContainerLog, _ := parseCreateContainerLog(tc.Input)
			if diff := cmp.Diff(tc.Expected, createContainerLog); diff != "" {
				t.Errorf("the result  is not matching with the expected result\n%s", diff)
			}
		})
	}
}

func TestReadGoStructFromString(t *testing.T) {
	testCases := []struct {
		Name       string
		Input      string
		StructName string
		Expected   map[string]string
	}{
		{
			Name:       "An example RunPodSandbox log",
			Input:      "RunPodSandbox for &PodSandboxMetadata{Name:podname,Uid:b86b49f2431d244c613996c6472eb864,Namespace:kube-system,Attempt:0,} returns sandbox id \"6123c6aacf0c78dc38ec4f0ff72edd3cf04eb82ca0e3e7dddd3950ea9753bdf1\"",
			StructName: "PodSandboxMetadata",
			Expected: map[string]string{
				"Name":      "podname",
				"Namespace": "kube-system",
				"Attempt":   "0",
				"Uid":       "b86b49f2431d244c613996c6472eb864",
			},
		},
		{
			Name:       "An example CreateContainer log",
			Input:      "CreateContainer within sandbox \"573208ed2827243aa3db0db52e8f5a8d6fe65fcf67d93ecc76f5a4d92378af83\" for &ContainerMetadata{Name:fluentbit-gke-init,Attempt:0,} returns container id \"fc3e6702e38e918ec02567358c4c889b38fc628838645222d9a08b0b68c90256\"",
			StructName: "ContainerMetadata",
			Expected: map[string]string{
				"Attempt": "0",
				"Name":    "fluentbit-gke-init",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			result := readGoStructFromString(tc.Input, tc.StructName)
			if diff := cmp.Diff(tc.Expected, result); diff != "" {
				t.Errorf("result is not matching with the expected result\n%s", diff)
			}
		})
	}
}

func TestReadNextQuotedString(t *testing.T) {
	testCases := []struct {
		Name     string
		Input    string
		Expected string
	}{
		{
			Name:     "standard input obtained from RunPodSandbox",
			Input:    "returns sandbox id \"6123c6aacf0c78dc38ec4f0ff72edd3cf04eb82ca0e3e7dddd3950ea9753bdf1\"",
			Expected: "6123c6aacf0c78dc38ec4f0ff72edd3cf04eb82ca0e3e7dddd3950ea9753bdf1",
		},
		{
			Name:     "not containing quote",
			Input:    "foo bar",
			Expected: "",
		},
		{
			Name:     "contains single double quote",
			Input:    "\"foo bar",
			Expected: "",
		},
		{
			Name:     "contains more than 3 double quote",
			Input:    "\"foo bar\" \"qux baz\"",
			Expected: "foo bar",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			nextQuoted := readNextQuotedString(tc.Input)
			if nextQuoted != tc.Expected {
				t.Errorf("expected:%s\nactual:%s", tc.Expected, nextQuoted)
			}
		})
	}
}
func TestGetSyslogIdentifier(t *testing.T) {
	testCases := []struct {
		Name                     string
		InputYaml                string
		ExpectedSyslogIdentifier string
	}{
		{
			Name: "jsonPayload.SYSLOG_IDENTIFIER exists without bracket",
			InputYaml: `jsonPayload:
  SYSLOG_IDENTIFIER: containerd`,
			ExpectedSyslogIdentifier: "containerd",
		},
		{
			Name: "jsonPayload.SYSLOG_IDENTIFIER exists with bracket",
			InputYaml: `jsonPayload:
  SYSLOG_IDENTIFIER: (dockerd)`,
			ExpectedSyslogIdentifier: "dockerd",
		},
		{
			Name:                     "jsonPayload.SYSLOG_IDENTIFIER doesn't exist",
			InputYaml:                `jsonPayload: {}`,
			ExpectedSyslogIdentifier: "Unknown",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			l := log_test.MustLogEntity(tc.InputYaml)
			syslogIdentifier := (&k8sNodeParser{}).GetSyslogIdentifier(l)
			if syslogIdentifier != tc.ExpectedSyslogIdentifier {
				t.Errorf("GetSyslogIdentifier() = %q, want:%q", syslogIdentifier, tc.ExpectedSyslogIdentifier)
			}
		})

	}
}

func TestK8sNodeParser_ParseKubeletLogWithPodNameButNotWithContainerName(t *testing.T) {
	wantLogSummary := "MountVolume.SetUp succeeded for volume \"kube-dns-config\" (UniqueName: \"kubernetes.io/configmap/34a3f9e5-4363-47a9-8bd9-3b37c60d107b-kube-dns-config\") pod \"kube-dns-58f547fd74-swzzt\" (UID: \"34a3f9e5-4363-47a9-8bd9-3b37c60d107b\") 【kube-system/kube-dns-58f547fd74-swzzt】"

	builder := history.NewBuilder(&ioconfig.IOConfig{
		ApplicationRoot: "/",
		DataDestination: "/tmp/",
		TemporaryFolder: "/tmp/",
	})
	cs, err := parser_test.ParseFromYamlLogFile("test/logs/k8s_node/kubelet_only_pod_name.yaml", &k8sNodeParser{}, builder)
	if err != nil {
		t.Errorf("got error %v, want nil", err)
	}

	event := cs.GetEvents(resourcepath.NodeComponent("gke-sample-cluster-default-abcdefgh-abcd", "kubelet"))
	if len(event) != 1 {
		t.Errorf("got %d events, want 1", len(event))
	}
	event = cs.GetEvents(resourcepath.Pod("kube-system", "kube-dns-58f547fd74-swzzt"))
	if len(event) != 1 {
		t.Errorf("got %d events, want 1", len(event))
	}

	gotLogSummary := cs.GetLogSummary()
	if gotLogSummary != wantLogSummary {
		t.Errorf("got %q log summary, want %q", gotLogSummary, wantLogSummary)
	}
}

func TestK8sNodeParser_ParseKubeletLogWithPodNameAndContainerName(t *testing.T) {
	wantLogSummary := "Killing container with a grace period(gracePeriod=30s) 【sidecar in kube-system/kube-dns-58f547fd74-swzzt】"

	builder := history.NewBuilder(&ioconfig.IOConfig{
		ApplicationRoot: "/",
		DataDestination: "/tmp/",
		TemporaryFolder: "/tmp/",
	})
	nodeName := "gke-sample-cluster-default-abcdefgh-abcd"
	podID := "foo"
	containerID := "5e0d5f0eab7a1ee243894fe769d690840243de4d53f5cb139094c395d8186881"
	podResourceBinding := noderesource.NewPodResourceBinding(podID, "kube-system", "kube-dns-58f547fd74-swzzt")
	containerResourceBinding := podResourceBinding.NewContainerResourceBinding(containerID, "sidecar")
	builder.ClusterResource.NodeResourceLogBinder.AddResourceBinding(nodeName, podResourceBinding)
	builder.ClusterResource.NodeResourceLogBinder.AddResourceBinding(nodeName, containerResourceBinding)

	cs, err := parser_test.ParseFromYamlLogFile("test/logs/k8s_node/kubelet_pod_and_container_name.yaml", &k8sNodeParser{}, builder)
	if err != nil {
		t.Errorf("got error %v, want nil", err)
	}

	event := cs.GetEvents(resourcepath.NodeComponent("gke-sample-cluster-default-abcdefgh-abcd", "kubelet"))
	if len(event) != 1 {
		t.Errorf("got %d events, want 1", len(event))
	}
	event = cs.GetEvents(resourcepath.Container("kube-system", "kube-dns-58f547fd74-swzzt", "sidecar"))
	if len(event) != 1 {
		t.Errorf("got %d events, want 1", len(event))
	}

	gotLogSummary := cs.GetLogSummary()
	if gotLogSummary != wantLogSummary {
		t.Errorf("got %q log summary, want %q", gotLogSummary, wantLogSummary)
	}
}

func TestK8sNodeParser_ParseContainerdRunPod(t *testing.T) {
	wantLogSummary := "RunPodSandbox for &PodSandboxMetadata{Name:kube-dns-58f547fd74-swzzt,Uid:34a3f9e5-4363-47a9-8bd9-3b37c60d107b,Namespace:kube-system,Attempt:0,} returns sandbox id \"e4b03e2...(kube-system/kube-dns-58f547fd74-swzzt)\"【kube-system/kube-dns-58f547fd74-swzzt】"

	builder := history.NewBuilder(&ioconfig.IOConfig{
		ApplicationRoot: "/",
		DataDestination: "/tmp/",
		TemporaryFolder: "/tmp/",
	})
	podSandboxID := "e4b03e280958b847e92e22b7a1570bdf63cb35432514b9a8f12f4b9adfe49714"
	nodeName := "gke-sample-cluster-default-abcdefgh-abcd"
	podNamespace := "kube-system"
	podName := "kube-dns-58f547fd74-swzzt"
	podResourceBinding := noderesource.NewPodResourceBinding(podSandboxID, podNamespace, podName)
	builder.ClusterResource.NodeResourceLogBinder.AddResourceBinding(nodeName, podResourceBinding)
	cs, err := parser_test.ParseFromYamlLogFile("test/logs/k8s_node/containerd_run_pod_sandbox.yaml", &k8sNodeParser{}, builder)
	if err != nil {
		t.Errorf("got error %v, want nil", err)
	}

	event := cs.GetEvents(resourcepath.NodeComponent("gke-sample-cluster-default-abcdefgh-abcd", "containerd"))
	if len(event) != 1 {
		t.Errorf("got %d events, want 1", len(event))
	}

	gotLogSummary := cs.GetLogSummary()
	if gotLogSummary != wantLogSummary {
		t.Errorf("got %q log summary, want %q", gotLogSummary, wantLogSummary)
	}
}

func TestK8sNodeParser_ParseCreateContainer(t *testing.T) {
	wantLogSummary := "CreateContainer within sandbox \"e4b03e2...(kube-system/kube-dns-58f547fd74-swzzt)\" for &ContainerMetadata{Name:kubedns,Attempt:0,} returns container id \"eea48bc...(kubedns in kube-system/kube-dns-58f547fd74-swzzt)\"【kube-system/kube-dns-58f547fd74-swzzt】 【kubedns in kube-system/kube-dns-58f547fd74-swzzt】"

	builder := history.NewBuilder(&ioconfig.IOConfig{
		ApplicationRoot: "/",
		DataDestination: "/tmp/",
		TemporaryFolder: "/tmp/",
	})
	wantNodeName := "gke-sample-cluster-default-abcdefgh-abcd"
	podSandboxID := "e4b03e280958b847e92e22b7a1570bdf63cb35432514b9a8f12f4b9adfe49714"
	podNamespace := "kube-system"
	podName := "kube-dns-58f547fd74-swzzt"
	containerName := "kubedns"
	containerID := "eea48bce362bdf290ff0d41655c9e580a41acd354cc845c7b7163d9dd9980bd9"
	podResourceBinding := noderesource.NewPodResourceBinding(podSandboxID, podNamespace, podName)
	containerResourceBinding := podResourceBinding.NewContainerResourceBinding(containerID, containerName)
	builder.ClusterResource.NodeResourceLogBinder.AddResourceBinding(wantNodeName, podResourceBinding)
	builder.ClusterResource.NodeResourceLogBinder.AddResourceBinding(wantNodeName, containerResourceBinding)
	cs, err := parser_test.ParseFromYamlLogFile("test/logs/k8s_node/containerd_create_container.yaml", &k8sNodeParser{}, builder)
	if err != nil {
		t.Errorf("got error %v, want nil", err)
	}

	event := cs.GetEvents(resourcepath.NodeComponent("gke-sample-cluster-default-abcdefgh-abcd", "containerd"))
	if len(event) != 1 {
		t.Errorf("got %d events, want 1", len(event))
	}
	event = cs.GetEvents(resourcepath.Container(podNamespace, podName, containerName))
	if len(event) != 1 {
		t.Errorf("got %d events, want 1", len(event))
	}

	gotLogSummary := cs.GetLogSummary()
	if gotLogSummary != wantLogSummary {
		t.Errorf("got %q log summary, want %q", gotLogSummary, wantLogSummary)
	}
}

func TestK8sNodeParser_ParseContainerdIncludingContainerIdOnly(t *testing.T) {
	wantLogSummary := "Stop container \"eea48bc...(kubedns in kube-system/kube-dns-58f547fd74-swzzt)\" with signal terminated 【kubedns in kube-system/kube-dns-58f547fd74-swzzt】"

	builder := history.NewBuilder(&ioconfig.IOConfig{
		ApplicationRoot: "/",
		DataDestination: "/tmp/",
		TemporaryFolder: "/tmp/",
	})
	nodeName := "gke-sample-cluster-default-abcdefgh-abcd"
	podSandboxID := "e4b03e280958b847e92e22b7a1570bdf63cb35432514b9a8f12f4b9adfe49714"
	containerID := "eea48bce362bdf290ff0d41655c9e580a41acd354cc845c7b7163d9dd9980bd9"
	podNamespace := "kube-system"
	podName := "kube-dns-58f547fd74-swzzt"
	containerName := "kubedns"
	podResourceBinding := noderesource.NewPodResourceBinding(podSandboxID, podNamespace, podName)
	containerResourceBinding := podResourceBinding.NewContainerResourceBinding(containerID, containerName)
	builder.ClusterResource.NodeResourceLogBinder.AddResourceBinding(nodeName, podResourceBinding)
	builder.ClusterResource.NodeResourceLogBinder.AddResourceBinding(nodeName, containerResourceBinding)

	cs, err := parser_test.ParseFromYamlLogFile("test/logs/k8s_node/containerd_only_container_id.yaml", &k8sNodeParser{}, builder)
	if err != nil {
		t.Errorf("got error %v, want nil", err)
	}

	event := cs.GetEvents(resourcepath.NodeComponent(nodeName, "containerd"))
	if len(event) != 1 {
		t.Errorf("got %d events, want 1", len(event))
	}
	event = cs.GetEvents(resourcepath.Container(podNamespace, podName, containerName))
	if len(event) != 1 {
		t.Errorf("got %d events, want 1", len(event))
	}

	gotLogSummary := cs.GetLogSummary()
	if gotLogSummary != wantLogSummary {
		t.Errorf("got %q log summary, want %q", gotLogSummary, wantLogSummary)
	}
}
