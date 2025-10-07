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

package googlecloudclustergkeonaws_contract

import (
	"testing"

	"cloud.google.com/go/gkemulticloud/apiv1/gkemulticloudpb"
)

func TestAwsClusterToClusterName(t *testing.T) {
	testCases := []struct {
		name    string
		cluster *gkemulticloudpb.AwsCluster
		want    string
	}{
		{
			name: "valid cluster name",
			cluster: &gkemulticloudpb.AwsCluster{
				Name: "projects/123456/locations/us-west1/awsClusters/my-aws-cluster",
			},
			want: "my-aws-cluster",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := awsClusterToClusterName(tc.cluster); got != tc.want {
				t.Errorf("awsClusterToClusterName() = %v, want %v", got, tc.want)
			}
		})
	}
}
