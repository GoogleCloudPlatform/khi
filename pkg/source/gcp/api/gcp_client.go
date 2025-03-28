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

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"strings"
	"sync"

	"github.com/GoogleCloudPlatform/khi/pkg/common/cache"
	"github.com/GoogleCloudPlatform/khi/pkg/common/httpclient"
	"github.com/GoogleCloudPlatform/khi/pkg/common/token"
)

var ErrorRateLimitExceeds = errors.New("ratelimit exceeds. retry it later")

var ErrorInternalServerError = errors.New("internal server error")

var MinWaitTimeOnRetriableError = 5
var MaxWaitTimeOnRetriableError = 60
var MaxRetryCount = 10
var RetriableHttpResponseCodes = []int{
	429, 500, 501, 502, 503,
}
var RetriableWithRefreshingTokenHttpResponseCodes = []int{
	401, 403,
}

type multicloudAPIEndpoint struct {
	Endpoint string
	Location string
}

var multicloudAPIEndpoints = []multicloudAPIEndpoint{
	{
		Endpoint: "https://asia-east2-gkemulticloud.googleapis.com",
		Location: "asia-east2",
	},
	{
		Endpoint: "https://asia-northeast2-gkemulticloud.googleapis.com",
		Location: "asia-northeast2",
	},
	{
		Endpoint: "https://asia-south1-gkemulticloud.googleapis.com",
		Location: "asia-south1",
	},
	{
		Endpoint: "https://asia-southeast1-gkemulticloud.googleapis.com",
		Location: "asia-southeast1",
	},
	{
		Endpoint: "https://asia-southeast2-gkemulticloud.googleapis.com",
		Location: "asia-southeast2",
	},
	{
		Endpoint: "https://australia-southeast1-gkemulticloud.googleapis.com",
		Location: "australia-southeast1",
	},
	{
		Endpoint: "https://europe-north1-gkemulticloud.googleapis.com",
		Location: "europe-north1",
	},
	{
		Endpoint: "https://europe-west1-gkemulticloud.googleapis.com",
		Location: "europe-west1",
	},
	{
		Endpoint: "https://europe-west2-gkemulticloud.googleapis.com",
		Location: "europe-west2",
	},
	{
		Endpoint: "https://europe-west3-gkemulticloud.googleapis.com",
		Location: "europe-west3",
	},
	{
		Endpoint: "https://europe-west4-gkemulticloud.googleapis.com",
		Location: "europe-west4",
	},
	{
		Endpoint: "https://europe-west6-gkemulticloud.googleapis.com",
		Location: "europe-west6",
	},
	{
		Endpoint: "https://europe-west9-gkemulticloud.googleapis.com",
		Location: "europe-west9",
	},
	{
		Endpoint: "https://northamerica-northeast1-gkemulticloud.googleapis.com",
		Location: "northamerica-northeast1",
	},
	{
		Endpoint: "https://southamerica-east1-gkemulticloud.googleapis.com",
		Location: "southamerica-east1",
	},
	{
		Endpoint: "https://us-east4-gkemulticloud.googleapis.com",
		Location: "us-east4",
	},
	{
		Endpoint: "https://us-west1-gkemulticloud.googleapis.com",
		Location: "us-west1",
	},
}

type GCPClientImpl struct {
	BaseClient httpclient.HTTPClient[*http.Response]
	// This is a parameter for limiting the result length of List log entries api call for testing purpose.
	MaxLogEntries int
}

// Digest implements task.CachableDependency.
func (pi *GCPClientImpl) Digest() string {
	return "singleton"
}

var _ cache.CacheDependency = (*GCPClientImpl)(nil)

var _ GCPClient = (*GCPClientImpl)(nil)

func NewGCPClient(refresher token.TokenRefresher, headerProviders []httpclient.HTTPHeaderProvider) (GCPClient, error) {
	return &GCPClientImpl{
		BaseClient: httpclient.NewRetryHttpClient(httpclient.NewBasicHttpClient().WithHeaderProvider(headerProviders...), MinWaitTimeOnRetriableError, MaxWaitTimeOnRetriableError, MaxRetryCount, RetriableHttpResponseCodes, RetriableWithRefreshingTokenHttpResponseCodes,
			refresher),
		MaxLogEntries: math.MaxInt,
	}, nil
}

func (c *GCPClientImpl) CreateGCPHttpRequest(ctx context.Context, method string, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

/**
 * Get the list of GKE cluster names.
 */
func (c *GCPClientImpl) GetClusterNames(ctx context.Context, projectId string) ([]string, error) {
	type gkeCluster struct {
		Name string `json:"name"`
	}
	type clusterListResponse struct {
		Clusters      []*gkeCluster `json:"clusters"`
		NextPageToken string        `json:"nextPageToken"`
	}
	var result []string
	pc := NewPageClient[clusterListResponse](c.BaseClient)
	clusterListResponses, err := pc.GetAll(ctx, func(hasToken bool, nextPageToken string) (*http.Request, error) {
		endpoint := fmt.Sprintf("https://container.googleapis.com/v1/projects/%s/locations/-/clusters", projectId)
		if nextPageToken != "-" {
			endpoint += "?pageToken=" + nextPageToken
		}
		return c.CreateGCPHttpRequest(ctx, "GET", endpoint, nil)
	}, func(response *clusterListResponse) string {
		return response.NextPageToken
	})
	if err != nil {
		return nil, err
	}
	for _, response := range clusterListResponses {
		for i := 0; i < len(response.Clusters); i++ {
			result = append(result, response.Clusters[i].Name)
		}
	}

	return result, nil
}

// GetAnthosAWSClusterNames retrieves the list of Anthos on AWS cluster names.
func (c *GCPClientImpl) GetAnthosAWSClusterNames(ctx context.Context, projectId string) ([]string, error) {
	type awsCluster struct {
		Name string `json:"name"`
	}
	type clusterListResponse struct {
		AwsClusters   []*awsCluster `json:"awsClusters"`
		NextPageToken string        `json:"nextPageToken"`
	}

	var result []string
	var lock sync.Mutex
	var wg sync.WaitGroup

	for _, endpoint := range multicloudAPIEndpoints {
		wg.Add(1)
		go func(endpoint multicloudAPIEndpoint) {
			defer wg.Done()

			pc := NewPageClient[clusterListResponse](c.BaseClient)
			awsClusterLists, err := pc.GetAll(ctx, func(hasToken bool, nextPageToken string) (*http.Request, error) {
				ep := fmt.Sprintf("%s/v1/projects/%s/locations/%s/awsClusters", endpoint.Endpoint, projectId, endpoint.Location)
				if hasToken {
					ep += "?pageToken=" + nextPageToken
				}
				return c.CreateGCPHttpRequest(ctx, "GET", ep, nil)
			}, func(response *clusterListResponse) string {
				return response.NextPageToken
			})
			if err != nil {
				return
			}

			lock.Lock()
			for _, awsClusterList := range awsClusterLists {
				for _, awsCluster := range awsClusterList.AwsClusters {
					clusterNameSegments := strings.Split(awsCluster.Name, "/")
					result = append(result, clusterNameSegments[len(clusterNameSegments)-1])
				}
			}
			lock.Unlock()
		}(endpoint)
	}

	wg.Wait()

	return result, nil
}

// GetAnthosAzureClusterNames retrieves the list of Anthos on Azure cluster names.
func (c *GCPClientImpl) GetAnthosAzureClusterNames(ctx context.Context, projectId string) ([]string, error) {
	type azureCluster struct {
		Name string `json:"name"`
	}
	type clusterListResponse struct {
		AzureClusters []*azureCluster `json:"azureClusters"`
		NextPageToken string          `json:"nextPageToken"`
	}

	var result []string
	var lock sync.Mutex
	var wg sync.WaitGroup

	for _, endpoint := range multicloudAPIEndpoints {
		wg.Add(1)
		go func(endpoint multicloudAPIEndpoint) {
			defer wg.Done()

			pc := NewPageClient[clusterListResponse](c.BaseClient)
			azureClusterLists, err := pc.GetAll(ctx, func(hasToken bool, nextPageToken string) (*http.Request, error) {
				ep := fmt.Sprintf("%s/v1/projects/%s/locations/%s/azureClusters", endpoint.Endpoint, projectId, endpoint.Location)
				if hasToken {
					ep += "?pageToken=" + nextPageToken
				}
				return c.CreateGCPHttpRequest(ctx, "GET", ep, nil)
			}, func(response *clusterListResponse) string {
				return response.NextPageToken
			})
			if err != nil {
				return
			}

			lock.Lock()
			for _, azureClusterList := range azureClusterLists {
				for _, azureCluster := range azureClusterList.AzureClusters {
					clusterNameSegments := strings.Split(azureCluster.Name, "/")
					result = append(result, clusterNameSegments[len(clusterNameSegments)-1])
				}
			}
			lock.Unlock()
		}(endpoint)
	}

	wg.Wait()

	return result, nil
}

func (c *GCPClientImpl) GetAnthosOnBaremetalClusterNames(ctx context.Context, projectId string) ([]string, error) {
	type baremetalCluster struct {
		Name string `json:"name"`
		// Ignoreing the other fields...
	}
	type clusterListResponse struct {
		BaremetalClusters []*baremetalCluster `json:"bareMetalClusters"`
		NextPageToken     string              `json:"nextPageToken"`
	}
	type baremetalAdminCluster struct {
		Name string `json:"name"`
		// Ignoreing the other fields...
	}
	type clusterAdminListResponse struct {
		BaremetalAdminClusters []*baremetalAdminCluster `json:"bareMetalAdminClusters"`
		NextPageToken          string                   `json:"nextPageToken"`
	}
	wg := sync.WaitGroup{}
	wg.Add(3)
	resultLock := sync.Mutex{}
	result := make([]string, 0)
	go func() {
		defer wg.Done()
		// Admin cluster can be only registered on the fleet membership.
		// Query fleet membership status as well.
		fleets, err := c.GetFleetMembershipNames(ctx, projectId)
		if err != nil {
			return
		}
		resultLock.Lock()
		defer resultLock.Unlock()
		result = append(result, fleets...)
	}()
	go func() {
		defer wg.Done()
		pc := NewPageClient[clusterListResponse](c.BaseClient)
		clusterLists, err := pc.GetAll(ctx, func(hasToken bool, nextPageToken string) (*http.Request, error) {
			endpoint := fmt.Sprintf("https://gkeonprem.googleapis.com/v1/projects/%s/locations/-/bareMetalClusters", projectId)
			if hasToken {
				endpoint += "?pageToken=" + nextPageToken
			}
			return c.CreateGCPHttpRequest(ctx, "GET", endpoint, nil)
		}, func(response *clusterListResponse) string {
			return response.NextPageToken
		})
		if err != nil {
			return
		}
		resultLock.Lock()
		defer resultLock.Unlock()
		for _, clusters := range clusterLists {
			for _, cluster := range clusters.BaremetalClusters {
				nameSegments := strings.Split(cluster.Name, "/")
				result = append(result, nameSegments[len(nameSegments)-1])
			}
		}
	}()
	go func() {
		defer wg.Done()
		pac := NewPageClient[clusterAdminListResponse](c.BaseClient)
		clusterAdminLists, err := pac.GetAll(ctx, func(hasToken bool, nextPageToken string) (*http.Request, error) {
			endpoint := fmt.Sprintf("https://gkeonprem.googleapis.com/v1/projects/%s/locations/-/bareMetalAdminClusters", projectId)
			if hasToken {
				endpoint += "?pageToken=" + nextPageToken
			}
			return c.CreateGCPHttpRequest(ctx, "GET", endpoint, nil)
		}, func(response *clusterAdminListResponse) string {
			return response.NextPageToken
		})
		if err != nil {
			return
		}
		resultLock.Lock()
		defer resultLock.Unlock()
		for _, cluster := range clusterAdminLists {
			for _, cluster := range cluster.BaremetalAdminClusters {
				nameSegments := strings.Split(cluster.Name, "/")
				result = append(result, nameSegments[len(nameSegments)-1])
			}
		}
	}()
	wg.Wait()
	return result, nil
}

func (c *GCPClientImpl) GetAnthosOnVMWareClusterNames(ctx context.Context, projectId string) ([]string, error) {
	type vmwareCluster struct {
		Name string `json:"name"`
		// Ignoreing the other fields...
	}
	type clusterListResponse struct {
		VMWareClusters []*vmwareCluster `json:"vmwareClusters"`
		NextPageToken  string           `json:"nextPageToken"`
	}
	type vmwareAdminCluster struct {
		Name string `json:"name"`
		// Ignoreing the other fields...
	}
	type clusterAdminListResponse struct {
		VMWareAdminClusters []*vmwareAdminCluster `json:"vmwareAdminClusters"`
		NextPageToken       string                `json:"nextPageToken"`
	}
	wg := sync.WaitGroup{}
	wg.Add(3)
	resultLock := sync.Mutex{}
	result := make([]string, 0)
	go func() {
		defer wg.Done()
		// Admin cluster can be only registered on the fleet membership.
		// Query fleet membership status as well.
		fleets, err := c.GetFleetMembershipNames(ctx, projectId)
		if err != nil {
			return
		}
		resultLock.Lock()
		defer resultLock.Unlock()
		result = append(result, fleets...)
	}()
	go func() {
		defer wg.Done()
		pc := NewPageClient[clusterListResponse](c.BaseClient)
		clusterLists, err := pc.GetAll(ctx, func(hasToken bool, nextPageToken string) (*http.Request, error) {
			endpoint := fmt.Sprintf("https://gkeonprem.googleapis.com/v1/projects/%s/locations/-/vmwareClusters", projectId)
			if hasToken {
				endpoint += "?pageToken=" + nextPageToken
			}
			return c.CreateGCPHttpRequest(ctx, "GET", endpoint, nil)
		}, func(response *clusterListResponse) string {
			return response.NextPageToken
		})
		if err != nil {
			return
		}
		resultLock.Lock()
		defer resultLock.Unlock()
		for _, clusters := range clusterLists {
			for _, cluster := range clusters.VMWareClusters {
				nameSegments := strings.Split(cluster.Name, "/")
				result = append(result, nameSegments[len(nameSegments)-1])
			}
		}
	}()
	go func() {
		defer wg.Done()
		pac := NewPageClient[clusterAdminListResponse](c.BaseClient)
		clusterAdminLists, err := pac.GetAll(ctx, func(hasToken bool, nextPageToken string) (*http.Request, error) {
			endpoint := fmt.Sprintf("https://gkeonprem.googleapis.com/v1/projects/%s/locations/-/vmwareAdminClusters", projectId)
			if hasToken {
				endpoint += "?pageToken=" + nextPageToken
			}
			return c.CreateGCPHttpRequest(ctx, "GET", endpoint, nil)
		}, func(response *clusterAdminListResponse) string {
			return response.NextPageToken
		})
		if err != nil {
			return
		}
		resultLock.Lock()
		defer resultLock.Unlock()
		for _, cluster := range clusterAdminLists {
			for _, cluster := range cluster.VMWareAdminClusters {
				nameSegments := strings.Split(cluster.Name, "/")
				result = append(result, nameSegments[len(nameSegments)-1])
			}
		}
	}()
	wg.Wait()
	return result, nil
}

func (c *GCPClientImpl) GetFleetMembershipNames(ctx context.Context, projectId string) ([]string, error) {
	type membershipResource struct {
		Name string `json:"name"`
		// Ignoreing the other fields...
	}
	type clusterAdminListResponse struct {
		Resources     []*membershipResource `json:"resources"`
		NextPageToken string                `json:"nextPageToken"`
	}
	pc := NewPageClient[clusterAdminListResponse](c.BaseClient)
	membershipLists, err := pc.GetAll(ctx, func(hasToken bool, nextPageToken string) (*http.Request, error) {
		endpoint := fmt.Sprintf("https://gkehub.googleapis.com/v1/projects/%s/locations/-/memberships", projectId)
		if hasToken {
			endpoint += "?pageToken=" + nextPageToken
		}
		return c.CreateGCPHttpRequest(ctx, "GET", endpoint, nil)
	}, func(response *clusterAdminListResponse) string {
		return response.NextPageToken
	})
	if err != nil {
		return nil, err
	}
	result := make([]string, 0)
	for _, membershipList := range membershipLists {
		for _, membership := range membershipList.Resources {
			membershipFragments := strings.Split(membership.Name, "/")
			result = append(result, membershipFragments[len(membershipFragments)-1])
		}
	}
	return result, nil
}

// Get all composer environment names from composer.googleapis.com in a region
// refs: https://cloud.google.com/composer/docs/reference/rest/v1/projects.locations.environments/list
func (c *GCPClientImpl) GetComposerEnvironmentNames(ctx context.Context, projectId string, location string) ([]string, error) {
	type environment struct {
		Name string `json:"name"`
	}
	type environmentListResponse struct {
		Environments  []environment `json:"environments"`
		NextPageToken string        `json:"nextPageToken"`
	}

	var result []string
	for nextPageToken := "-"; nextPageToken != ""; {
		select {
		case <-ctx.Done():
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			continue
		default:
			endpoint := fmt.Sprintf("https://composer.googleapis.com/v1/projects/%s/locations/%s/environments", projectId, location)
			if nextPageToken != "-" {
				endpoint += "?pageToken=" + nextPageToken
			}
			req, err := c.CreateGCPHttpRequest(ctx, "GET", endpoint, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to create GCP HTTP request: %w", err)
			}

			client := httpclient.NewJsonResponseHttpClient[environmentListResponse](c.BaseClient)
			response, httpResponse, err := client.DoWithContext(ctx, req)
			if httpResponse != nil && httpResponse.Body != nil {
				defer httpResponse.Body.Close()
			}
			if err != nil {
				return nil, fmt.Errorf("failed to get JSON response: %w", err)
			}

			for _, environment := range response.Environments {
				// fullname: projects/PROJECT_ID/locations/LOCATION/environments/ENVIRONMENT_NAME
				fullname := environment.Name
				name := strings.Split(fullname, "/")[len(strings.Split(fullname, "/"))-1]
				result = append(result, name)
			}
			nextPageToken = response.NextPageToken
		}
	}

	return result, nil
}

/**
 * Query logs with specified filter
 */
func (c *GCPClientImpl) ListLogEntries(ctx context.Context, projectId string, filter string, logSink chan any) error {
	type logEntriesListRequest struct {
		ResourceNames []string `json:"resourceNames"`
		Filter        string   `json:"filter"`
		OrderBy       string   `json:"orderBy"`
		PageSize      int64    `json:"pageSize"`
		PageToken     string   `json:"pageToken,omitempty"`
	}

	type logEntriesListResponse struct {
		Entries       []any  `json:"entries"`
		NextPageToken string `json:"nextPageToken"`
	}

	defer close(logSink)

	ENDPOINT := "https://logging.googleapis.com/v2/entries:list"
	MAXIMUM_PAGE_SIZE := 1000

	nextPageToken := ""
	pageCount := 0
	for entryIndex := 0; entryIndex < c.MaxLogEntries; entryIndex += MAXIMUM_PAGE_SIZE {
		queryEnd := false
		select {
		case <-ctx.Done():
			if err := ctx.Err(); err != nil {
				// cancel operation
				return err
			}
		default:
			requestBody := logEntriesListRequest{
				ResourceNames: []string{fmt.Sprintf("projects/%s", projectId)},
				Filter:        filter,
				OrderBy:       "timestamp asc",
				PageSize:      int64(math.Min(float64(MAXIMUM_PAGE_SIZE), float64(c.MaxLogEntries-entryIndex))), // logging API can take 1000 entries at most.
				PageToken:     nextPageToken,
			}
			requestBytes, err := json.Marshal(requestBody)
			if err != nil {
				return err
			}
			req, err := c.CreateGCPHttpRequest(ctx, "POST", ENDPOINT, bytes.NewReader(requestBytes))
			if err != nil {
				return err
			}
			client := httpclient.NewJsonResponseHttpClient[logEntriesListResponse](c.BaseClient)
			response, httpResponse, err := client.DoWithContext(ctx, req)
			if httpResponse != nil && httpResponse.Body != nil {
				defer httpResponse.Body.Close()
			}
			if err != nil {
				if httpResponse != nil {
					slog.ErrorContext(ctx, fmt.Sprintf("Unretriable error found: %d:%s", httpResponse.StatusCode, httpResponse.Status))
				}
				return err
			}
			for _, entry := range response.Entries {
				logSink <- entry
			}

			if response.NextPageToken == "" {
				queryEnd = true
				break
			}
			nextPageToken = response.NextPageToken
			pageCount += 1
		}
		if queryEnd {
			break
		}
	}
	return nil
}
