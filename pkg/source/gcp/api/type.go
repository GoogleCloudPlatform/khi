package api

// ref: https://cloud.google.com/kubernetes-engine/docs/reference/rest/v1/projects.locations.clusters#Cluster
type Cluster struct {
	Name           string            `json:"name"`
	ResourceLabels map[string]string `json:"resourceLabels"`
}
