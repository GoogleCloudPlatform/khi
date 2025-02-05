package noderesource

import (
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
)

type ResourceBinding interface {
	// GetUniqueIdentifier returns ids/names that needs to be included in the log body when the resource associates with it.
	GetUniqueIdentifier() string
	// GetResourcePath returns the path on where this resource should have the event.
	GetResourcePath() resourcepath.ResourcePath
	// RewriteLogSummary receives summary from original or privious another resource association and return rewritten summary.
	RewriteLogSummary(summary string) string
}

// PodResourceBinding is a ResourceBinding for a Pod resource on a node.
type PodResourceBinding struct {
	PodSandboxId string
	PodName      string
	PodNamespace string
}

// NewPodResourceBinding returns a new PodResourceBinding instance.
func NewPodResourceBinding(podSandboxId string, podName string, podNamespace string) *PodResourceBinding {
	return &PodResourceBinding{
		PodSandboxId: podSandboxId,
		PodName:      podName,
		PodNamespace: podNamespace,
	}
}

// GetResourcePath implements ResourceBinding.
func (p *PodResourceBinding) GetResourcePath() resourcepath.ResourcePath {
	return resourcepath.Pod(p.PodNamespace, p.PodName)
}

// GetUniqueIdentifier implements ResourceBinding.
func (p *PodResourceBinding) GetUniqueIdentifier() string {
	return p.PodSandboxId
}

// RewriteLogSummary implements ResourceBinding.
func (p *PodResourceBinding) RewriteLogSummary(summary string) string {
	return rewriteIdWithReadableName(p.PodSandboxId, fmt.Sprintf("%s/%s", p.PodNamespace, p.PodName), fmt.Sprintf("%s【%s/%s】", summary, p.PodNamespace, p.PodName))
}

// NewContainerResourceBinding returns an instance of ContainerRersourceBinding that is a child of this Pod.
func (p *PodResourceBinding) NewContainerResourceBinding(containerId string, containerName string) *ContainerResourceBinding {
	return &ContainerResourceBinding{
		ConainerId:    containerId,
		ContainerName: containerName,
		PodNamespace:  p.PodNamespace,
		PodName:       p.PodName,
	}
}

var _ ResourceBinding = (*PodResourceBinding)(nil)

// ContainerResourceBinding is a ResourceBinding for a container on a node.
type ContainerResourceBinding struct {
	ConainerId    string
	ContainerName string
	PodNamespace  string
	PodName       string
}

// GetResourcePath implements ResourceBinding.
func (c *ContainerResourceBinding) GetResourcePath() resourcepath.ResourcePath {
	return resourcepath.Container(c.PodNamespace, c.PodName, c.ContainerName)
}

// GetUniqueIdentifier implements ResourceBinding.
func (c *ContainerResourceBinding) GetUniqueIdentifier() string {
	return c.ConainerId
}

// RewriteLogSummary implements ResourceBinding.
func (c *ContainerResourceBinding) RewriteLogSummary(summary string) string {
	return rewriteIdWithReadableName(c.ConainerId, fmt.Sprintf("%s in %s/%s", c.PodNamespace, c.PodName, c.ContainerName), fmt.Sprintf("%s 【%s in %s/%s】", summary, c.ContainerName, c.PodNamespace, c.PodName))
}

var _ ResourceBinding = (*ContainerResourceBinding)(nil)

func rewriteIdWithReadableName(replaceTarget string, readableName string, originalMessage string) string {
	converted := fmt.Sprintf("%s...(%s)", replaceTarget[:min(len(replaceTarget), 7)], readableName)
	return strings.ReplaceAll(originalMessage, replaceTarget, converted)
}
