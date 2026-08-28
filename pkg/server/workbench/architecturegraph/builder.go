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

package architecturegraph

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	apiv1 "github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1"
	khifilev6model "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench/cel"
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench/sparsebitset"
	"github.com/RoaringBitmap/roaring/v2"
	"google.golang.org/protobuf/proto"
)

const (
	defaultDeletionThresholdSeconds = 180.0
)

var podPhaseChildRegex = regexp.MustCompile(`^([^/]+)/([^\[]+?)(?:\[([^\]]*)\])?$`)

// podPhaseInfo stores pod placement and phase metadata resolved from a node's child pod-phase timeline.
type podPhaseInfo struct {
	timelineID     uint32
	nodeName       string
	namespace      string
	name           string
	uid            string
	phase          string
	isPhaseHealthy bool
	updatedAtNs    int64
	deletedAtNs    int64
}

// isPodPhaseHealthy evaluates whether the given Kubernetes Pod phase indicates a healthy state.
func isPodPhaseHealthy(phase string) bool {
	return phase == "Running" || phase == "Succeeded" || phase == "Completed"
}

// parsePodPhaseState parses a PodPhase timeline revision state string into a Kubernetes Pod phase and health status.
func parsePodPhaseState(state string) (phase string, isHealthy bool) {
	lower := strings.ToLower(state)
	switch {
	case strings.Contains(lower, "running"):
		phase = "Running"
	case strings.Contains(lower, "succeeded"):
		phase = "Succeeded"
	case strings.Contains(lower, "pending"), strings.Contains(lower, "scheduled"):
		phase = "Pending"
	case strings.Contains(lower, "failed"):
		phase = "Failed"
	default:
		phase = "Unknown"
	}
	return phase, isPodPhaseHealthy(phase)
}

// Builder constructs the Kubernetes architecture graph at a specified timestamp.
type Builder struct {
	timelines   []*cel.TimelineData
	timelineMap map[uint32]*cel.TimelineData
	internPool  *khifilev6model.InternPool
}

// NewBuilder creates a new Builder instance.
func NewBuilder(
	timelines []*cel.TimelineData,
	timelineMap map[uint32]*cel.TimelineData,
	internPool *khifilev6model.InternPool,
) *Builder {
	return &Builder{
		timelines:   timelines,
		timelineMap: timelineMap,
		internPool:  internPool,
	}
}

// Build generates the complete GetArchitectureGraphResponse based on the provided request parameters.
func (b *Builder) Build(
	ctx context.Context,
	req *apiv1.GetArchitectureGraphRequest,
) (*apiv1.GetArchitectureGraphResponse, error) {
	thresholdSec := req.GetDeletionThresholdSeconds()
	if thresholdSec <= 0 {
		thresholdSec = defaultDeletionThresholdSeconds
	}
	timeNs := req.GetTimestampNs()

	var allowedTimelines *roaring.Bitmap
	if req.GetTimelineBitset() != nil {
		allowedTimelines = sparsebitset.Decode(req.GetTimelineBitset())
	}

	nodesMap := make(map[string]*apiv1.GraphNode)
	podsMap := make(map[string]*apiv1.GraphPod)
	servicesMap := make(map[string]*apiv1.GraphService)
	serviceSelectors := make(map[string]map[string]string)
	podOwnersMap := make(map[string]*apiv1.GraphPodOwner)
	podOwnerOwnersMap := make(map[string]*apiv1.GraphPodOwnerOwner)
	podPhaseMap := make(map[string]*podPhaseInfo)

	for _, tl := range b.timelines {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		path := b.computeTimelinePath(tl)

		// Collect pod phase information recorded under Node timelines:
		// [cluster, "core/v1", "node", "cluster-scope", <nodeName>, "<namespace>/<podName>[<uid>]"]
		if len(path) == 6 && strings.ToLower(path[2]) == "node" && path[3] == "cluster-scope" {
			match := podPhaseChildRegex.FindStringSubmatch(path[5])
			if len(match) >= 3 {
				rev, _ := b.lookupRevisionAtNs(tl, timeNs)
				if rev != nil {
					updatedAtNs, deletedAtNs, ok := resolveResourceRetention(timeNs, rev, thresholdSec)
					if ok {
						ns := match[1]
						pName := match[2]
						uid := ""
						if len(match) >= 4 {
							uid = match[3]
						}
						podID := fmt.Sprintf("pod/%s/%s", ns, pName)
						phase, isPhaseHealthy := parsePodPhaseState(rev.State)

						if existing, exists := podPhaseMap[podID]; !exists || existing.updatedAtNs < updatedAtNs {
							podPhaseMap[podID] = &podPhaseInfo{
								timelineID:     tl.ID,
								nodeName:       path[4],
								namespace:      ns,
								name:           pName,
								uid:            uid,
								phase:          phase,
								isPhaseHealthy: isPhaseHealthy,
								updatedAtNs:    updatedAtNs,
								deletedAtNs:    deletedAtNs,
							}
						}
					}
				}
			}
			continue
		}

		if allowedTimelines != nil && !allowedTimelines.Contains(tl.ID) {
			continue
		}

		if len(path) != 5 {
			continue
		}

		kind := strings.ToLower(path[2])
		namespace := path[3]
		name := path[4]

		rev, revIdx := b.lookupRevisionAtNs(tl, timeNs)
		if rev == nil {
			continue
		}

		updatedAtNs, deletedAtNs, ok := resolveResourceRetention(timeNs, rev, thresholdSec)
		if !ok {
			continue
		}

		manifestReader := b.resolveManifestReader(tl, revIdx)

		switch kind {
		case "node":
			if namespace == "cluster-scope" {
				node := b.parseNode(tl.ID, name, updatedAtNs, deletedAtNs, manifestReader)
				nodesMap[node.GetId()] = node
			}
		case "pod":
			pod := b.parsePod(tl, name, namespace, timeNs, updatedAtNs, deletedAtNs, manifestReader)
			podsMap[pod.GetId()] = pod
		case "service":
			svc, selector := b.parseService(tl.ID, name, namespace, updatedAtNs, deletedAtNs, manifestReader)
			servicesMap[svc.GetId()] = svc
			if len(selector) > 0 {
				serviceSelectors[svc.GetId()] = selector
			}
		case "daemonset", "job", "replicaset":
			owner := b.parsePodOwner(tl.ID, kind, name, namespace, updatedAtNs, deletedAtNs, manifestReader)
			podOwnersMap[owner.GetId()] = owner
		case "deployment", "cronjob":
			ownerOwner := b.parsePodOwnerOwner(tl.ID, kind, name, namespace, updatedAtNs, deletedAtNs, manifestReader)
			podOwnerOwnersMap[ownerOwner.GetId()] = ownerOwner
		}
	}

	// Merge podPhaseMap into podsMap.
	// Node child timelines (core/v1/node/cluster-scope/<node>/<namespace>/<pod>[<uid>]) serve two purposes:
	// 1. As evidence to resolve node placement, phase, and UID for an existing pod (even if the child timeline itself was filtered out).
	// 2. To synthesize a pod when no primary pod timeline exists, in which case the child timeline must be allowed by allowedTimelines.
	for podID, info := range podPhaseMap {
		pod, exists := podsMap[podID]
		if !exists {
			if allowedTimelines != nil && !allowedTimelines.Contains(info.timelineID) {
				continue
			}
			pod = &apiv1.GraphPod{
				Id:             proto.String(podID),
				TimelineId:     proto.Uint32(info.timelineID),
				Name:           proto.String(info.name),
				Namespace:      proto.String(info.namespace),
				NodeName:       proto.String(info.nodeName),
				Uid:            proto.String(info.uid),
				PodIp:          proto.String("-"),
				Phase:          proto.String(info.phase),
				IsPhaseHealthy: proto.Bool(info.isPhaseHealthy),
				UpdatedAtNs:    proto.Int64(info.updatedAtNs),
				DeletedAtNs:    proto.Int64(info.deletedAtNs),
				Labels:         make(map[string]string),
			}
			podsMap[podID] = pod
		} else {
			if pod.GetNodeName() == "" && info.nodeName != "" {
				pod.NodeName = proto.String(info.nodeName)
			}
			if pod.GetUid() == "" && info.uid != "" {
				pod.Uid = proto.String(info.uid)
			}
			if (pod.GetPhase() == "" || pod.GetPhase() == "Unknown") && info.phase != "Unknown" {
				pod.Phase = proto.String(info.phase)
				pod.IsPhaseHealthy = proto.Bool(info.isPhaseHealthy)
			}
		}
	}

	for _, pod := range podsMap {
		if pod.GetNodeName() != "" {
			nodeID := fmt.Sprintf("node/cluster-scope/%s", pod.GetNodeName())
			if _, exists := nodesMap[nodeID]; !exists {
				nodesMap[nodeID] = &apiv1.GraphNode{
					Id:         proto.String(nodeID),
					TimelineId: proto.Uint32(0),
					Name:       proto.String(pod.GetNodeName()),
					PodCidr:    proto.String("-"),
					InternalIp: proto.String("-"),
					ExternalIp: proto.String("-"),
					Labels:     make(map[string]string),
				}
			}
		}
	}

	edges := b.buildEdges(nodesMap, podsMap, servicesMap, serviceSelectors, podOwnersMap, podOwnerOwnersMap)

	nodes := sortedValuesByID(nodesMap)
	pods := sortedValuesByID(podsMap)
	services := sortedValuesByID(servicesMap)
	podOwners := sortedValuesByID(podOwnersMap)
	podOwnerOwners := sortedValuesByID(podOwnerOwnersMap)

	sort.Slice(edges, func(i, j int) bool {
		if edges[i].GetType() != edges[j].GetType() {
			return edges[i].GetType() < edges[j].GetType()
		}
		if edges[i].GetSourceId() != edges[j].GetSourceId() {
			return edges[i].GetSourceId() < edges[j].GetSourceId()
		}
		return edges[i].GetTargetId() < edges[j].GetTargetId()
	})

	return &apiv1.GetArchitectureGraphResponse{
		TimestampNs:    proto.Int64(timeNs),
		Nodes:          nodes,
		Pods:           pods,
		Services:       services,
		PodOwners:      podOwners,
		PodOwnerOwners: podOwnerOwners,
		Edges:          edges,
	}, nil
}

type identifiable interface {
	GetId() string
}

func sortedValuesByID[T identifiable](m map[string]T) []T {
	res := make([]T, 0, len(m))
	for _, v := range m {
		res = append(res, v)
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].GetId() < res[j].GetId()
	})
	return res
}

func (b *Builder) buildEdges(
	nodesMap map[string]*apiv1.GraphNode,
	podsMap map[string]*apiv1.GraphPod,
	servicesMap map[string]*apiv1.GraphService,
	serviceSelectors map[string]map[string]string,
	podOwnersMap map[string]*apiv1.GraphPodOwner,
	podOwnerOwnersMap map[string]*apiv1.GraphPodOwnerOwner,
) []*apiv1.GraphEdge {
	var edges []*apiv1.GraphEdge

	ownerUIDToPodOwners := make(map[string][]*apiv1.GraphPodOwner)
	for _, po := range podOwnersMap {
		if po.GetUid() != "" {
			ownerUIDToPodOwners[po.GetUid()] = append(ownerUIDToPodOwners[po.GetUid()], po)
		}
	}

	ownerUIDToTopLevelOwners := make(map[string][]*apiv1.GraphPodOwnerOwner)
	for _, ownerOwner := range podOwnerOwnersMap {
		if ownerOwner.GetUid() != "" {
			ownerUIDToTopLevelOwners[ownerOwner.GetUid()] = append(ownerUIDToTopLevelOwners[ownerOwner.GetUid()], ownerOwner)
		}
	}

	for _, pod := range podsMap {
		for _, ownerUID := range pod.OwnerUids {
			if owners, exists := ownerUIDToPodOwners[ownerUID]; exists {
				for _, owner := range owners {
					edges = append(edges, &apiv1.GraphEdge{
						Type:     apiv1.GraphEdge_EDGE_TYPE_POD_OWNER_TO_POD.Enum(),
						SourceId: proto.String(owner.GetId()),
						TargetId: proto.String(pod.GetId()),
					})
				}
			}
		}
	}

	for _, po := range podOwnersMap {
		for _, ownerUID := range po.OwnerUids {
			if ownerOwners, exists := ownerUIDToTopLevelOwners[ownerUID]; exists {
				for _, oo := range ownerOwners {
					edges = append(edges, &apiv1.GraphEdge{
						Type:     apiv1.GraphEdge_EDGE_TYPE_POD_OWNER_OWNER_TO_POD_OWNER.Enum(),
						SourceId: proto.String(oo.GetId()),
						TargetId: proto.String(po.GetId()),
					})
				}
			}
		}
	}

	for svcID, selector := range serviceSelectors {
		svc := servicesMap[svcID]
		if svc == nil {
			continue
		}
		for _, pod := range podsMap {
			if pod.GetNamespace() != svc.GetNamespace() {
				continue
			}
			matched := true
			for k, v := range selector {
				if podVal, ok := pod.Labels[k]; !ok || podVal != v {
					matched = false
					break
				}
			}
			if matched {
				edges = append(edges, &apiv1.GraphEdge{
					Type:     apiv1.GraphEdge_EDGE_TYPE_SERVICE_TO_POD.Enum(),
					SourceId: proto.String(svc.GetId()),
					TargetId: proto.String(pod.GetId()),
				})
			}
		}
	}

	return edges
}

func (b *Builder) parseNode(
	timelineID uint32,
	name string,
	updatedAtNs int64,
	deletedAtNs int64,
	reader *structured.NodeReader,
) *apiv1.GraphNode {
	node := &apiv1.GraphNode{
		Id:          proto.String(fmt.Sprintf("node/cluster-scope/%s", name)),
		TimelineId:  proto.Uint32(timelineID),
		Name:        proto.String(name),
		PodCidr:     proto.String("-"),
		InternalIp:  proto.String("-"),
		ExternalIp:  proto.String("-"),
		UpdatedAtNs: proto.Int64(updatedAtNs),
		DeletedAtNs: proto.Int64(deletedAtNs),
		Labels:      make(map[string]string),
	}

	if reader == nil {
		return node
	}

	node.Uid = proto.String(reader.ReadStringOrDefault("metadata.uid", ""))
	node.Labels = readMap(reader, "metadata.labels")
	node.PodCidr = proto.String(reader.ReadStringOrDefault("spec.podCIDR", "-"))

	if taintsReader, err := reader.GetReader("spec.taints"); err == nil {
		for _, tReader := range taintsReader.Children() {
			key := tReader.ReadStringOrDefault("key", "")
			effect := tReader.ReadStringOrDefault("effect", "")
			if key != "" {
				node.Taints = append(node.Taints, fmt.Sprintf("%s(%s)", key, effect))
			}
		}
	}

	if addrsReader, err := reader.GetReader("status.addresses"); err == nil {
		for _, aReader := range addrsReader.Children() {
			addrType := aReader.ReadStringOrDefault("type", "")
			addr := aReader.ReadStringOrDefault("address", "")
			if addrType == "InternalIP" && addr != "" {
				node.InternalIp = proto.String(addr)
			} else if addrType == "ExternalIP" && addr != "" {
				node.ExternalIp = proto.String(addr)
			}
		}
	}

	node.Conditions = readConditions(reader, false)

	return node
}

func (b *Builder) parsePod(
	tl *cel.TimelineData,
	name string,
	namespace string,
	timeNs int64,
	updatedAtNs int64,
	deletedAtNs int64,
	reader *structured.NodeReader,
) *apiv1.GraphPod {
	pod := &apiv1.GraphPod{
		Id:             proto.String(fmt.Sprintf("pod/%s/%s", namespace, name)),
		TimelineId:     proto.Uint32(tl.ID),
		Name:           proto.String(name),
		Namespace:      proto.String(namespace),
		PodIp:          proto.String("-"),
		Phase:          proto.String("Unknown"),
		IsPhaseHealthy: proto.Bool(false),
		UpdatedAtNs:    proto.Int64(updatedAtNs),
		DeletedAtNs:    proto.Int64(deletedAtNs),
		Labels:         make(map[string]string),
	}

	containersMap := make(map[string]*apiv1.GraphContainer)

	if reader != nil {
		pod.Uid = proto.String(reader.ReadStringOrDefault("metadata.uid", ""))
		pod.Labels = readMap(reader, "metadata.labels")
		pod.PodIp = proto.String(reader.ReadStringOrDefault("status.podIP", "-"))
		phase := reader.ReadStringOrDefault("status.phase", "Unknown")
		pod.Phase = proto.String(phase)
		pod.IsPhaseHealthy = proto.Bool(isPodPhaseHealthy(phase))
		pod.OwnerUids = readOwnerUIDs(reader)

		nodeName := reader.ReadStringOrDefault("spec.nodeName", "")
		if nodeName == "" {
			nodeName = b.resolveNodeNameFromBinding(tl, timeNs)
		}
		if nodeName != "" {
			pod.NodeName = proto.String(nodeName)
		}

		loadContainers := func(fieldPath string, isInit bool) {
			if cReader, err := reader.GetReader(fieldPath); err == nil {
				for _, cr := range cReader.Children() {
					cName := cr.ReadStringOrDefault("name", "")
					if cName != "" {
						containersMap[cName] = &apiv1.GraphContainer{
							Name:            proto.String(cName),
							IsInitContainer: proto.Bool(isInit),
							Status:          proto.String("Unknown"),
							Reason:          proto.String("Unknown"),
						}
					}
				}
			}
		}
		loadContainers("spec.initContainers", true)
		loadContainers("spec.containers", false)

		updateContainerStatuses := func(path string) {
			if csReader, err := reader.GetReader(path); err == nil {
				for _, itemReader := range csReader.Children() {
					cName := itemReader.ReadStringOrDefault("name", "")
					c, exists := containersMap[cName]
					if !exists {
						c = &apiv1.GraphContainer{
							Name:   proto.String(cName),
							Status: proto.String("Unknown"),
							Reason: proto.String("Unknown"),
						}
						containersMap[cName] = c
					}
					c.StatusReadFromManifest = proto.Bool(true)
					c.Ready = proto.Bool(itemReader.ReadBoolOrDefault("ready", false))

					if _, err := itemReader.GetReader("state.running"); err == nil {
						c.Status = proto.String("Running")
						c.IsStatusHealthy = proto.Bool(true)
					} else if termReader, err := itemReader.GetReader("state.terminated"); err == nil {
						reason := termReader.ReadStringOrDefault("reason", "")
						c.Code = proto.Int32(int32(termReader.ReadIntOrDefault("exitCode", 0)))
						c.Reason = proto.String(reason)
						if reason != "" {
							c.Status = proto.String(reason)
						} else {
							c.Status = proto.String("Terminated")
						}
						c.IsStatusHealthy = proto.Bool(reason == "Completed")
					} else if waitReader, err := itemReader.GetReader("state.waiting"); err == nil {
						reason := waitReader.ReadStringOrDefault("reason", "")
						c.Reason = proto.String(reason)
						if reason != "" {
							c.Status = proto.String(reason)
						} else {
							c.Status = proto.String("Waiting")
						}
						c.IsStatusHealthy = proto.Bool(false)
					}
				}
			}
		}

		updateContainerStatuses("status.initContainerStatuses")
		updateContainerStatuses("status.containerStatuses")
		pod.Conditions = readConditions(reader, true)
	} else {
		nodeName := b.resolveNodeNameFromBinding(tl, timeNs)
		if nodeName != "" {
			pod.NodeName = proto.String(nodeName)
		}
	}

	// Fallback to child timelines if containers or conditions were not parsed from manifest.
	fallbackContainers := len(containersMap) == 0
	fallbackConditions := len(pod.Conditions) == 0
	if fallbackContainers || fallbackConditions {
		for _, childID := range tl.ChildrenIDs {
			child := b.timelineMap[childID]
			if child == nil {
				continue
			}
			if fallbackContainers && child.TimelineType == "container" {
				c := &apiv1.GraphContainer{
					Name:            proto.String(child.Name),
					IsInitContainer: proto.Bool(false),
					Status:          proto.String("Unknown"),
					Reason:          proto.String("Unknown"),
				}
				cRev, _ := b.lookupRevisionAtNs(child, timeNs)
				if cRev != nil {
					stateLower := strings.ToLower(cRev.State)
					switch {
					case strings.Contains(stateLower, "running"):
						c.Status = proto.String("Running")
						c.IsStatusHealthy = proto.Bool(true)
					case strings.Contains(stateLower, "terminated"), strings.Contains(stateLower, "completed"):
						c.Status = proto.String("Terminated")
						c.IsStatusHealthy = proto.Bool(strings.Contains(stateLower, "completed"))
					case strings.Contains(stateLower, "waiting"):
						c.Status = proto.String("Waiting")
						c.IsStatusHealthy = proto.Bool(false)
					}
				}
				containersMap[child.Name] = c
			} else if fallbackConditions && child.TimelineType == "condition" {
				condRev, _ := b.lookupRevisionAtNs(child, timeNs)
				if condRev != nil {
					stateLower := strings.ToLower(condRev.State)
					status := "Unknown"
					isPositive := false
					if strings.Contains(stateLower, "true") {
						status = "True"
						isPositive = true
					} else if strings.Contains(stateLower, "false") {
						status = "False"
					}
					pod.Conditions = append(pod.Conditions, &apiv1.GraphCondition{
						Type:       proto.String(child.Name),
						Status:     proto.String(status),
						IsPositive: proto.Bool(isPositive),
					})
				}
			}
		}
	}

	for _, c := range containersMap {
		pod.Containers = append(pod.Containers, c)
	}
	sort.Slice(pod.Containers, func(i, j int) bool {
		return pod.Containers[i].GetName() < pod.Containers[j].GetName()
	})

	return pod
}

func (b *Builder) resolveNodeNameFromBinding(tl *cel.TimelineData, timeNs int64) string {
	for _, childID := range tl.ChildrenIDs {
		child := b.timelineMap[childID]
		if child != nil && child.Name == "binding" {
			bRev, bRevIdx := b.lookupRevisionAtNs(child, timeNs)
			if bRev != nil {
				bReader := b.resolveManifestReader(child, bRevIdx)
				if bReader != nil {
					targetName := bReader.ReadStringOrDefault("target.name", "")
					if targetName == "" {
						targetName = bReader.ReadStringOrDefault("spec.target.name", "")
					}
					if targetName != "" {
						return targetName
					}
				}
			}
		}
	}
	return ""
}

func (b *Builder) parseService(
	timelineID uint32,
	name string,
	namespace string,
	updatedAtNs int64,
	deletedAtNs int64,
	reader *structured.NodeReader,
) (*apiv1.GraphService, map[string]string) {
	svc := &apiv1.GraphService{
		Id:          proto.String(fmt.Sprintf("service/%s/%s", namespace, name)),
		TimelineId:  proto.Uint32(timelineID),
		Name:        proto.String(name),
		Namespace:   proto.String(namespace),
		Type:        proto.String("Unknown"),
		ClusterIp:   proto.String("-"),
		UpdatedAtNs: proto.Int64(updatedAtNs),
		DeletedAtNs: proto.Int64(deletedAtNs),
		Labels:      make(map[string]string),
	}

	if reader == nil {
		return svc, nil
	}

	svc.Uid = proto.String(reader.ReadStringOrDefault("metadata.uid", ""))
	svc.Labels = readMap(reader, "metadata.labels")
	svc.Type = proto.String(reader.ReadStringOrDefault("spec.type", "Unknown"))

	clusterIP := reader.ReadStringOrDefault("status.clusterIp", "")
	if clusterIP == "" {
		clusterIP = reader.ReadStringOrDefault("spec.clusterIP", "-")
	}
	svc.ClusterIp = proto.String(clusterIP)

	selector := readMap(reader, "spec.selector")

	return svc, selector
}

func (b *Builder) parsePodOwner(
	timelineID uint32,
	kind string,
	name string,
	namespace string,
	updatedAtNs int64,
	deletedAtNs int64,
	reader *structured.NodeReader,
) *apiv1.GraphPodOwner {
	owner := &apiv1.GraphPodOwner{
		Id:          proto.String(fmt.Sprintf("%s/%s/%s", kind, namespace, name)),
		TimelineId:  proto.Uint32(timelineID),
		Name:        proto.String(name),
		Namespace:   proto.String(namespace),
		Kind:        proto.String(kind),
		UpdatedAtNs: proto.Int64(updatedAtNs),
		DeletedAtNs: proto.Int64(deletedAtNs),
		Labels:      make(map[string]string),
	}

	if reader == nil {
		return owner
	}

	owner.Uid = proto.String(reader.ReadStringOrDefault("metadata.uid", ""))
	owner.Labels = readMap(reader, "metadata.labels")
	owner.OwnerUids = readOwnerUIDs(reader)

	return owner
}

func (b *Builder) parsePodOwnerOwner(
	timelineID uint32,
	kind string,
	name string,
	namespace string,
	updatedAtNs int64,
	deletedAtNs int64,
	reader *structured.NodeReader,
) *apiv1.GraphPodOwnerOwner {
	ownerOwner := &apiv1.GraphPodOwnerOwner{
		Id:          proto.String(fmt.Sprintf("%s/%s/%s", kind, namespace, name)),
		TimelineId:  proto.Uint32(timelineID),
		Name:        proto.String(name),
		Namespace:   proto.String(namespace),
		Kind:        proto.String(kind),
		UpdatedAtNs: proto.Int64(updatedAtNs),
		DeletedAtNs: proto.Int64(deletedAtNs),
		Labels:      make(map[string]string),
	}

	if reader == nil {
		return ownerOwner
	}

	ownerOwner.Uid = proto.String(reader.ReadStringOrDefault("metadata.uid", ""))
	ownerOwner.Labels = readMap(reader, "metadata.labels")

	return ownerOwner
}

func (b *Builder) computeTimelinePath(tl *cel.TimelineData) []string {
	var segs []string
	curr := tl
	visited := make(map[uint32]struct{})
	for curr != nil {
		if _, seen := visited[curr.ID]; seen {
			break
		}
		visited[curr.ID] = struct{}{}
		segs = append(segs, curr.Name)
		if curr.ParentID == 0 || b.timelineMap == nil {
			break
		}
		curr = b.timelineMap[curr.ParentID]
	}
	for i, j := 0, len(segs)-1; i < j; i, j = i+1, j-1 {
		segs[i], segs[j] = segs[j], segs[i]
	}
	return segs
}

func (b *Builder) lookupRevisionAtNs(tl *cel.TimelineData, timeNs int64) (*cel.RevisionInfo, int) {
	if tl == nil || len(tl.Revisions) == 0 {
		return nil, -1
	}
	idx := sort.Search(len(tl.Revisions), func(i int) bool {
		return tl.Revisions[i].ChangedTime > timeNs
	})
	if idx == 0 {
		return nil, -1
	}
	return &tl.Revisions[idx-1], idx - 1
}

func (b *Builder) resolveManifestReader(tl *cel.TimelineData, revIdx int) *structured.NodeReader {
	if b.internPool == nil || tl == nil || revIdx < 0 || revIdx >= len(tl.Revisions) {
		return nil
	}

	for i := revIdx; i >= 0; i-- {
		structID := tl.Revisions[i].ResourceBodyStructID
		if structID == 0 {
			continue
		}
		s := b.internPool.ResolveStructFromID(structID)
		if s == nil {
			continue
		}
		node, err := khifilev6model.FromInternedStruct(s, b.internPool)
		if err != nil {
			continue
		}
		return structured.NewNodeReader(node)
	}

	return nil
}

func resolveResourceRetention(
	timeNs int64,
	rev *cel.RevisionInfo,
	thresholdSec float64,
) (updatedAtNs, deletedAtNs int64, retain bool) {
	diffSec := float64(timeNs-rev.ChangedTime) / 1e9
	isDelete := rev.Verb == "Delete" || rev.Verb == "DeleteCollection"

	if isDelete {
		if diffSec > thresholdSec {
			return 0, 0, false
		}
		return 0, rev.ChangedTime, true
	}
	return rev.ChangedTime, 0, true
}

func readMap(reader *structured.NodeReader, fieldPath string) map[string]string {
	result := make(map[string]string)
	if reader == nil {
		return result
	}
	mapReader, err := reader.GetReader(fieldPath)
	if err != nil {
		return result
	}
	for key, valReader := range mapReader.Children() {
		val, err := valReader.NodeScalarValue()
		if err == nil && val != nil {
			result[key.Key] = fmt.Sprint(val)
		}
	}
	return result
}

func readOwnerUIDs(reader *structured.NodeReader) []string {
	var uids []string
	if reader == nil {
		return uids
	}
	ownersReader, err := reader.GetReader("metadata.ownerReferences")
	if err != nil {
		return uids
	}
	for _, oReader := range ownersReader.Children() {
		uid := oReader.ReadStringOrDefault("uid", "")
		if uid != "" {
			uids = append(uids, uid)
		}
	}
	return uids
}

func readConditions(reader *structured.NodeReader, isPod bool) []*apiv1.GraphCondition {
	var conditions []*apiv1.GraphCondition
	if reader == nil {
		return conditions
	}
	condsReader, err := reader.GetReader("status.conditions")
	if err != nil {
		return conditions
	}
	for _, cReader := range condsReader.Children() {
		cType := cReader.ReadStringOrDefault("type", "")
		status := cReader.ReadStringOrDefault("status", "")
		message := cReader.ReadStringOrDefault("message", "")

		var isPositive bool
		if isPod {
			isPositive = status == "True"
		} else {
			if cType == "Ready" {
				isPositive = status == "True"
			} else {
				isPositive = status == "False"
			}
		}

		conditions = append(conditions, &apiv1.GraphCondition{
			Type:       proto.String(cType),
			Status:     proto.String(status),
			Message:    proto.String(message),
			IsPositive: proto.Bool(isPositive),
		})
	}
	return conditions
}
