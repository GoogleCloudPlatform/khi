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

package googlecloudbigquery_impl

import (
	"context"
	"fmt"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/legacyparser"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/grouper"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudbigquery_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudbigquery/contract"
	"gopkg.in/yaml.v3"
)

type bigqueryJobParser struct{}

// Dependencies implements parser.Parser.
func (b *bigqueryJobParser) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// Description implements parser.Parser.
func (b *bigqueryJobParser) Description() string {
	return "Gather completedEvent logs and visualize when did jobs create/start/finish."
}

// GetParserName implements parser.Parser.
func (b *bigqueryJobParser) GetParserName() string {
	return "BigQuery CompletedEvent Parser"
}

// Grouper implements parser.Parser.
func (b *bigqueryJobParser) Grouper() grouper.LogGrouper {
	return grouper.AllDependentLogGrouper
}

// LogTask implements parser.Parser.
func (b *bigqueryJobParser) LogTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudbigquery_contract.BigQueryCompletedEventQueryID.Ref()
}

// Parse implements parser.Parser.
func (b *bigqueryJobParser) Parse(ctx context.Context, l *log.Log, cs *history.ChangeSet, builder *history.Builder) error {
	// Parse BigQuery job from LogEntity
	job, err := NewBigQueryJobFromYamlStrings(l)
	if err != nil {
		return err
	}

	// Execution User
	requester := l.ReadStringOrDefault("protoPayload.authenticationInfo.principalEmail", "unknown")

	// BigQuery resource path
	resourcePath := job.ToResourcePath()
	body, _ := l.Serialize("protoPayload.serviceData.jobCompletedEvent.job", &structured.YAMLNodeSerializer{})

	// Record New Revision: Inserted BQ job
	cs.AddRevision(resourcePath, &history.StagingResourceRevision{
		Verb:       enum.RevisionVerbBigQuryJobCreate,
		State:      enum.RevisionStateBigQueryJobPending,
		Requestor:  requester,
		ChangeTime: parseTime(job.Statistics.CreateTime),
		Partial:    false,
		Body:       string(body),
	})

	// Record New Revision: Started BQ job
	cs.AddRevision(resourcePath, &history.StagingResourceRevision{
		Verb:       enum.RevisionVerbBigQuryJobStart,
		State:      enum.RevisionStateBigQueryJobRunning,
		Requestor:  requester,
		ChangeTime: parseTime(job.Statistics.StartTime),
		Partial:    false,
		Body:       string(body),
	})

	var state enum.RevisionState
	// status.error must be "" when the job succeeded
	// otherwise status.error must be an object
	isSuccess := job.Status.Error.Code == 0 && job.Status.Error.Message == ""
	if isSuccess {
		state = enum.RevisionStateBigQueryJobSuccess
	} else {
		state = enum.RevisionStateBigQueryJobFailed
	}

	// Record New Revision: Finished BQ job
	cs.AddRevision(resourcePath, &history.StagingResourceRevision{
		Verb:       enum.RevisionVerbBigQuryJobDone,
		State:      state,
		Requestor:  requester,
		ChangeTime: parseTime(job.Statistics.EndTime),
		Partial:    false,
		Body:       string(body),
	})

	var str string
	if isSuccess {
		str = "success"
	} else {
		str = "failed"
		cs.SetLogSeverity(enum.SeverityError)
	}

	// This summary shows on right panel
	cs.SetLogSummary(fmt.Sprintf("BigQuery Job(%s) finished with status %s", job.Name.JobId, str))

	// This event shows on the timeline view as ♢
	cs.AddEvent(resourcePath)
	return nil
}

// TargetLogType implements parser.Parser.
func (b *bigqueryJobParser) TargetLogType() enum.LogType {
	return enum.LogTypeEvent
}

var _ legacyparser.Parser = (*bigqueryJobParser)(nil)

func NewBigQueryJobFromYamlStrings(l *log.Log) (*googlecloudbigquery_contract.BigQueryJob, error) {
	jobYaml, err := l.Serialize("protoPayload.serviceData.jobCompletedEvent.job", &structured.YAMLNodeSerializer{})
	if err != nil {
		return nil, err
	}
	var job = &googlecloudbigquery_contract.BigQueryJob{}
	err = yaml.Unmarshal(jobYaml, job)
	if err != nil {
		return nil, err
	}
	return job, nil
}

func parseTime(timeString string) time.Time {
	t, err := time.Parse(time.RFC3339, timeString)
	if err != nil {
		panic(err)
	}

	return t
}

var BigQueryJobParserTask = legacyparser.NewParserTaskFromParser(googlecloudbigquery_contract.BigQueryJobParserTaskID, &bigqueryJobParser{}, 1000, true, []string{googlecloudbigquery_contract.InspectionTypeId})
