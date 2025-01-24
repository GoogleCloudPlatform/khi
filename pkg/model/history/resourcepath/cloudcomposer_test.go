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

package resourcepath

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/model"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
)

func TestComposerTaskInstance(t *testing.T) {
	expectedParentRelationship := enum.RelationshipChild
	tests := []struct {
		name string
		ti   *model.AirflowTaskInstance
		want string
	}{
		{
			name: "basic",
			ti:   model.NewAirflowTaskInstance("my_dag", "my_task", "my_run", "0", "my_host", "my_status"),
			want: "Cloud Composer#Task Instance#my_dag#my_run#my_task+0",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := ComposerTaskInstance(test.ti)
			if got.Path != test.want {
				t.Errorf("ComposerTaskInstance(%v).Path = %v, want %v", test.ti, got.Path, test.want)
			}
			if got.ParentRelationship != expectedParentRelationship {
				t.Errorf("ComposerTaskInstance(%v).Parentrelationship = %v, want %v", test.ti, got.ParentRelationship, expectedParentRelationship)
			}
		})
	}
}

func TestComposerAirflowWorker(t *testing.T) {
	expectedParentRelationship := enum.RelationshipChild
	tests := []struct {
		name string
		wo   *model.AirflowWorker
		want string
	}{
		{
			name: "basic",
			wo:   model.NewAirflowWorker("my_host"),
			want: "Cloud Composer#Airflow Worker#cluster-scope#my_host",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := ComposerAirflowWorker(test.wo)
			if got.Path != test.want {
				t.Errorf("ComposerAirflowWorker(%v).Path = %v, want %v", test.wo, got.Path, test.want)
			}
			if got.ParentRelationship != expectedParentRelationship {
				t.Errorf("ComposerAirflowWorker(%v).Parentrelationship = %v, want %v", test.wo, got.ParentRelationship, expectedParentRelationship)
			}
		})
	}
}

func TestDagFileProcessorStats(t *testing.T) {
	expectedParentRelationship := enum.RelationshipChild
	tests := []struct {
		name  string
		stats *model.DagFileProcessorStats
		want  string
	}{
		{
			name:  "basic",
			stats: model.NewDagFileProcessorStats("my_dag_file_path", "my_dag_file_path", "10", "10"),
			want:  "Cloud Composer#Dag File Processor Stats#cluster-scope#my_dag_file_path",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := DagFileProcessorStats(test.stats)
			if got.Path != test.want {
				t.Errorf("DagFileProcessorStats(%v).Path = %v, want %v", test.stats, got.Path, test.want)
			}
			if got.ParentRelationship != expectedParentRelationship {
				t.Errorf("DagFileProcessorStats(%v).Parentrelationship = %v, want %v", test.stats, got.ParentRelationship, expectedParentRelationship)
			}
		})
	}
}
