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

package apiv1

import (
	"context"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud/logestimator"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	apiv1 "github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1"
	"github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1/apiv1connect"
	"github.com/GoogleCloudPlatform/khi/pkg/server/upload"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"google.golang.org/protobuf/proto"
)

// InspectionServiceServer implements the apiv1connect.InspectionServiceHandler interface.
type InspectionServiceServer struct {
	inspectionServer    *coreinspection.InspectionTaskServer
	streamCycleDuration time.Duration
	updateInterval      time.Duration
}

var _ apiv1connect.InspectionServiceHandler = (*InspectionServiceServer)(nil)

// NewInspectionServiceServer creates a new InspectionServiceServer with default 30s stream cycle and 1s interval.
func NewInspectionServiceServer(inspectionServer *coreinspection.InspectionTaskServer) *InspectionServiceServer {
	return NewInspectionServiceServerWithIntervals(inspectionServer, 30*time.Second, 1*time.Second)
}

// NewInspectionServiceServerWithIntervals creates a new InspectionServiceServer with configurable stream intervals.
func NewInspectionServiceServerWithIntervals(
	inspectionServer *coreinspection.InspectionTaskServer,
	streamCycleDuration time.Duration,
	updateInterval time.Duration,
) *InspectionServiceServer {
	return &InspectionServiceServer{
		inspectionServer:    inspectionServer,
		streamCycleDuration: streamCycleDuration,
		updateInterval:      updateInterval,
	}
}

// GetInspectionTypes returns all supported inspection types registered on the server.
func (s *InspectionServiceServer) GetInspectionTypes(
	ctx context.Context,
	req *connect.Request[apiv1.GetInspectionTypesRequest],
) (*connect.Response[apiv1.GetInspectionTypesResponse], error) {
	types := s.inspectionServer.GetAllInspectionTypes()
	resTypes := make([]*apiv1.InspectionType, 0, len(types))
	for _, t := range types {
		resTypes = append(resTypes, &apiv1.InspectionType{
			Id:          proto.String(t.Id),
			Name:        proto.String(t.Name),
			Description: proto.String(t.Description),
			Icon:        proto.String(t.Icon),
			Labels:      t.Labels,
		})
	}
	return connect.NewResponse(&apiv1.GetInspectionTypesResponse{
		Types: resTypes,
	}), nil
}

// GetInspections returns the current snapshot of active inspection runner sessions.
func (s *InspectionServiceServer) GetInspections(
	ctx context.Context,
	req *connect.Request[apiv1.GetInspectionsRequest],
) (*connect.Response[apiv1.GetInspectionsResponse], error) {
	items, err := s.collectInspectionListItems()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&apiv1.GetInspectionsResponse{
		Inspections: items,
	}), nil
}

// WatchInspections streams active inspection runner session updates and closes after streamCycleDuration.
func (s *InspectionServiceServer) WatchInspections(
	ctx context.Context,
	req *connect.Request[apiv1.WatchInspectionsRequest],
	stream *connect.ServerStream[apiv1.WatchInspectionsResponse],
) error {
	// Send initial snapshot immediately upon connection.
	items, err := s.collectInspectionListItems()
	if err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}
	if err := stream.Send(&apiv1.WatchInspectionsResponse{Inspections: items}); err != nil {
		return err
	}

	cycleTimer := time.NewTimer(s.streamCycleDuration)
	defer cycleTimer.Stop()

	ticker := time.NewTicker(s.updateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-cycleTimer.C:
			// Gracefully close stream after cycle duration expires to prompt client reconnect.
			return nil
		case <-ticker.C:
			items, err := s.collectInspectionListItems()
			if err != nil {
				return connect.NewError(connect.CodeInternal, err)
			}
			if err := stream.Send(&apiv1.WatchInspectionsResponse{Inspections: items}); err != nil {
				return err
			}
		}
	}
}

// PullInspections pulls active inspection runner sessions snapshot without opening a persistent stream.
func (s *InspectionServiceServer) PullInspections(
	ctx context.Context,
	req *connect.Request[apiv1.PullInspectionsRequest],
) (*connect.Response[apiv1.PullInspectionsResponse], error) {
	items, err := s.collectInspectionListItems()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&apiv1.PullInspectionsResponse{
		Inspections: items,
	}), nil
}

// CreateInspection instantiates a new inspection runner session for a given inspection type.
func (s *InspectionServiceServer) CreateInspection(
	ctx context.Context,
	req *connect.Request[apiv1.CreateInspectionRequest],
) (*connect.Response[apiv1.CreateInspectionResponse], error) {
	typeID := req.Msg.GetInspectionTypeId()
	inspectionID, err := s.inspectionServer.CreateInspection(typeID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(&apiv1.CreateInspectionResponse{
		InspectionId: proto.String(inspectionID),
	}), nil
}

// UpdateInspection updates metadata of an inspection runner session.
func (s *InspectionServiceServer) UpdateInspection(
	ctx context.Context,
	req *connect.Request[apiv1.UpdateInspectionRequest],
) (*connect.Response[apiv1.UpdateInspectionResponse], error) {
	inspectionID := req.Msg.GetInspectionId()
	task := s.inspectionServer.GetInspection(inspectionID)
	if task == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("inspection %s was not found", inspectionID))
	}
	md, err := task.GetCurrentMetadata()
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	header, found := typedmap.Get(md, inspectionmetadata.HeaderMetadataKey)
	if !found || header == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("header not found"))
	}
	header.InspectionName = req.Msg.GetName()
	header.SuggestedFileName = fmt.Sprintf("%s.khi", req.Msg.GetName())
	return connect.NewResponse(&apiv1.UpdateInspectionResponse{}), nil
}

// GetInspectionFeatures retrieves the list of toggleable features for an inspection runner.
func (s *InspectionServiceServer) GetInspectionFeatures(
	ctx context.Context,
	req *connect.Request[apiv1.GetInspectionFeaturesRequest],
) (*connect.Response[apiv1.GetInspectionFeaturesResponse], error) {
	inspectionID := req.Msg.GetInspectionId()
	task := s.inspectionServer.GetInspection(inspectionID)
	if task == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("inspection %s was not found", inspectionID))
	}
	features, err := task.FeatureList()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resFeatures := make([]*apiv1.InspectionFeature, 0, len(features))
	for _, f := range features {
		resFeatures = append(resFeatures, &apiv1.InspectionFeature{
			Id:          proto.String(f.Id),
			Label:       proto.String(f.Label),
			Description: proto.String(f.Description),
			Enabled:     proto.Bool(f.Enabled),
		})
	}
	return connect.NewResponse(&apiv1.GetInspectionFeaturesResponse{
		Features: resFeatures,
	}), nil
}

// UpdateInspectionFeatures updates the enabled states of features for an inspection runner.
func (s *InspectionServiceServer) UpdateInspectionFeatures(
	ctx context.Context,
	req *connect.Request[apiv1.UpdateInspectionFeaturesRequest],
) (*connect.Response[apiv1.UpdateInspectionFeaturesResponse], error) {
	inspectionID := req.Msg.GetInspectionId()
	task := s.inspectionServer.GetInspection(inspectionID)
	if task == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("inspection %s was not found", inspectionID))
	}
	if len(req.Msg.GetEnabledFeatureIds()) > 0 {
		if err := task.SetFeatureList(req.Msg.GetEnabledFeatureIds()); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}
	if len(req.Msg.GetFeatureStates()) > 0 {
		if err := task.UpdateFeatureMap(req.Msg.GetFeatureStates()); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}
	return connect.NewResponse(&apiv1.UpdateInspectionFeaturesResponse{}), nil
}

// DryRunInspection dry-runs an inspection task graph and evaluates form fields with the given parameters.
func (s *InspectionServiceServer) DryRunInspection(
	ctx context.Context,
	req *connect.Request[apiv1.DryRunInspectionRequest],
) (*connect.Response[apiv1.DryRunInspectionResponse], error) {
	inspectionID := req.Msg.GetInspectionId()
	task := s.inspectionServer.GetInspection(inspectionID)
	if task == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("inspection %s was not found", inspectionID))
	}
	values := convertParametersToMap(req.Msg.GetParameters())
	result, err := task.DryRun(ctx, &inspectioncore_contract.InspectionRequest{
		Values: values,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	mdMap, ok := result.Metadata.(map[string]interface{})
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("unexpected dryrun metadata format"))
	}

	resp := &apiv1.DryRunInspectionResponse{}
	if fields, ok := mdMap["form"].([]inspectionmetadata.ParameterFormField); ok && fields != nil {
		resp.Form = convertFormFields(fields)
	}
	if queries, ok := mdMap["query"].([]*inspectionmetadata.QueryItem); ok && queries != nil {
		resp.Queries = convertQueries(queries)
	}
	if plan, ok := mdMap["plan"].(*inspectionmetadata.InspectionPlanMetadata); ok && plan != nil {
		resp.Plan = &apiv1.InspectionPlan{TaskGraph: proto.String(plan.TaskGraph)}
	}
	if cmd, ok := mdMap["jobCommand"].(*inspectionmetadata.JobModeCommandSerializable); ok && cmd != nil {
		resp.JobCommand = &apiv1.InspectionJobCommand{Command: proto.String(cmd.Command)}
	}

	return connect.NewResponse(resp), nil
}

// RunInspection kicks off asynchronous execution of an inspection task graph with the given parameters.
func (s *InspectionServiceServer) RunInspection(
	ctx context.Context,
	req *connect.Request[apiv1.RunInspectionRequest],
) (*connect.Response[apiv1.RunInspectionResponse], error) {
	inspectionID := req.Msg.GetInspectionId()
	task := s.inspectionServer.GetInspection(inspectionID)
	if task == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("inspection %s was not found", inspectionID))
	}
	values := convertParametersToMap(req.Msg.GetParameters())
	err := task.Run(ctx, &inspectioncore_contract.InspectionRequest{
		Values: values,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&apiv1.RunInspectionResponse{}), nil
}

// CancelInspection aborts an in-progress inspection run.
func (s *InspectionServiceServer) CancelInspection(
	ctx context.Context,
	req *connect.Request[apiv1.CancelInspectionRequest],
) (*connect.Response[apiv1.CancelInspectionResponse], error) {
	inspectionID := req.Msg.GetInspectionId()
	task := s.inspectionServer.GetInspection(inspectionID)
	if task == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("inspection %s was not found", inspectionID))
	}
	if err := task.Cancel(); err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(&apiv1.CancelInspectionResponse{}), nil
}

// GetInspectionMetadata returns serializable metadata for a completed inspection run.
func (s *InspectionServiceServer) GetInspectionMetadata(
	ctx context.Context,
	req *connect.Request[apiv1.GetInspectionMetadataRequest],
) (*connect.Response[apiv1.GetInspectionMetadataResponse], error) {
	inspectionID := req.Msg.GetInspectionId()
	task := s.inspectionServer.GetInspection(inspectionID)
	if task == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("inspection %s was not found", inspectionID))
	}
	md, err := task.Metadata()
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	resp := &apiv1.GetInspectionMetadataResponse{}
	if h, ok := md["header"].(*inspectionmetadata.HeaderMetadata); ok && h != nil {
		resp.Header = &apiv1.InspectionHeader{
			InspectionType:         proto.String(h.InspectionType),
			InspectionName:         proto.String(h.InspectionName),
			InspectionTypeIconPath: proto.String(h.InspectionTypeIconPath),
			InspectTimeUnixSeconds: proto.Int64(h.InspectTimeUnixSeconds),
			StartTimeUnixSeconds:   proto.Int64(h.StartTimeUnixSeconds),
			EndTimeUnixSeconds:     proto.Int64(h.EndTimeUnixSeconds),
			SuggestedFilename:      proto.String(h.SuggestedFileName),
			FileSize:               proto.Int64(int64(h.FileSize)),
		}
	}
	if p, ok := md["plan"].(*inspectionmetadata.InspectionPlanMetadata); ok && p != nil {
		resp.Plan = &apiv1.InspectionPlan{TaskGraph: proto.String(p.TaskGraph)}
	}
	if q, ok := md["query"].([]*inspectionmetadata.QueryItem); ok && q != nil {
		resp.Queries = convertQueries(q)
	}
	if l, ok := md["log"].([]inspectionmetadata.SerializableLogItem); ok && l != nil {
		resp.Logs = make([]*apiv1.InspectionLog, 0, len(l))
		for _, item := range l {
			resp.Logs = append(resp.Logs, &apiv1.InspectionLog{
				Id:   proto.String(item.Id),
				Name: proto.String(item.Name),
				Log:  proto.String(item.Log),
			})
		}
	}
	if e, ok := md["error"].(*inspectionmetadata.ErrorMessageSetMetadata); ok && e != nil {
		resp.Error = convertErrorSet(e)
	}
	return connect.NewResponse(resp), nil
}

// GetInspectionDataChunk downloads a slice of binary data from the generated .khi archive.
func (s *InspectionServiceServer) GetInspectionDataChunk(
	ctx context.Context,
	req *connect.Request[apiv1.GetInspectionDataChunkRequest],
) (*connect.Response[apiv1.GetInspectionDataChunkResponse], error) {
	inspectionID := req.Msg.GetInspectionId()
	task := s.inspectionServer.GetInspection(inspectionID)
	if task == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("inspection %s was not found", inspectionID))
	}
	result, err := task.Result()
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	fileSize, err := result.ResultStore.GetInspectionResultSizeInBytes()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	offset := req.Msg.GetOffsetBytes()
	if offset < 0 || offset > int64(fileSize) {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("offset %d is out of bounds (file size: %d)", offset, fileSize))
	}
	maxSize := req.Msg.GetMaxSizeBytes()
	if maxSize <= 0 {
		maxSize = 16 * 1024 * 1024
	}

	reader, err := result.ResultStore.GetRangeReader(offset, maxSize)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer reader.Close()

	readSize := min(maxSize, int64(fileSize)-offset)
	if readSize < 0 {
		readSize = 0
	}
	buf := make([]byte, readSize)
	if len(buf) > 0 {
		_, err = io.ReadFull(reader, buf)
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	return connect.NewResponse(&apiv1.GetInspectionDataChunkResponse{
		Data:               buf,
		TotalFileSizeBytes: proto.Int64(int64(fileSize)),
	}), nil
}

func (s *InspectionServiceServer) collectInspectionListItems() ([]*apiv1.InspectionListItem, error) {
	runners := s.inspectionServer.GetAllRunners()
	items := make([]*apiv1.InspectionListItem, 0, len(runners))
	for _, runner := range runners {
		if !runner.Started() {
			continue
		}
		item, err := runnerToInspectionListItem(runner)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	slices.SortFunc(items, func(a, b *apiv1.InspectionListItem) int {
		return strings.Compare(a.GetId(), b.GetId())
	})
	return items, nil
}

func runnerToInspectionListItem(runner *coreinspection.InspectionTaskRunner) (*apiv1.InspectionListItem, error) {
	md, err := runner.GetCurrentMetadata()
	if err != nil {
		return nil, err
	}
	item := &apiv1.InspectionListItem{
		Id: proto.String(runner.ID),
	}
	if header, found := typedmap.Get(md, inspectionmetadata.HeaderMetadataKey); found && header != nil {
		item.Header = &apiv1.InspectionHeader{
			InspectionType:         proto.String(header.InspectionType),
			InspectionName:         proto.String(header.InspectionName),
			InspectionTypeIconPath: proto.String(header.InspectionTypeIconPath),
			InspectTimeUnixSeconds: proto.Int64(header.InspectTimeUnixSeconds),
			StartTimeUnixSeconds:   proto.Int64(header.StartTimeUnixSeconds),
			EndTimeUnixSeconds:     proto.Int64(header.EndTimeUnixSeconds),
			SuggestedFilename:      proto.String(header.SuggestedFileName),
			FileSize:               proto.Int64(int64(header.FileSize)),
		}
	}
	if progress, found := typedmap.Get(md, inspectionmetadata.ProgressMetadataKey); found && progress != nil {
		item.Progress = convertProgress(progress)
	}
	if errorSet, found := typedmap.Get(md, inspectionmetadata.ErrorMessageSetMetadataKey); found && errorSet != nil {
		item.Error = convertErrorSet(errorSet)
	}
	return item, nil
}

func convertProgress(progress *inspectionmetadata.Progress) *apiv1.InspectionProgress {
	p := &apiv1.InspectionProgress{
		Phase: convertInspectionPhase(progress.Phase).Enum(),
	}
	if progress.TotalProgress != nil {
		p.TotalProgress = &apiv1.TaskProgressElement{
			Id:            proto.String(progress.TotalProgress.Id),
			Label:         proto.String(progress.TotalProgress.Label),
			Message:       proto.String(progress.TotalProgress.Message),
			Percentage:    proto.Float32(progress.TotalProgress.Percentage),
			Indeterminate: proto.Bool(progress.TotalProgress.Indeterminate),
		}
	}
	p.Progresses = make([]*apiv1.TaskProgressElement, 0, len(progress.TaskProgresses))
	for _, tp := range progress.TaskProgresses {
		p.Progresses = append(p.Progresses, &apiv1.TaskProgressElement{
			Id:            proto.String(tp.Id),
			Label:         proto.String(tp.Label),
			Message:       proto.String(tp.Message),
			Percentage:    proto.Float32(tp.Percentage),
			Indeterminate: proto.Bool(tp.Indeterminate),
		})
	}
	return p
}

func convertInspectionPhase(phase inspectionmetadata.TaskProgressPhase) apiv1.InspectionPhase {
	switch phase {
	case inspectionmetadata.TaskPhaseRunning:
		return apiv1.InspectionPhase_INSPECTION_PHASE_RUNNING
	case inspectionmetadata.TaskPhaseDone:
		return apiv1.InspectionPhase_INSPECTION_PHASE_DONE
	case inspectionmetadata.TaskPhaseError:
		return apiv1.InspectionPhase_INSPECTION_PHASE_ERROR
	case inspectionmetadata.TaskPhaseCancelled:
		return apiv1.InspectionPhase_INSPECTION_PHASE_CANCELLED
	default:
		return apiv1.InspectionPhase_INSPECTION_PHASE_UNSPECIFIED
	}
}

func convertErrorSet(errorSet *inspectionmetadata.ErrorMessageSetMetadata) *apiv1.InspectionErrorSet {
	res := &apiv1.InspectionErrorSet{
		ErrorMessages: make([]*apiv1.InspectionError, 0, len(errorSet.ErrorMessages)),
	}
	for _, msg := range errorSet.ErrorMessages {
		res.ErrorMessages = append(res.ErrorMessages, &apiv1.InspectionError{
			ErrorId: proto.String(fmt.Sprintf("%d", msg.ErrorId)),
			Message: proto.String(msg.Message),
			Link:    proto.String(msg.Link),
		})
	}
	return res
}

func convertPreset(preset logestimator.EstimatedCountPreset) apiv1.EstimatedCountPreset {
	switch preset {
	case logestimator.EstimatedCountPresetFew:
		return apiv1.EstimatedCountPreset_ESTIMATED_COUNT_PRESET_FEW
	default:
		return apiv1.EstimatedCountPreset_ESTIMATED_COUNT_PRESET_UNSPECIFIED
	}
}

func convertQueries(queries []*inspectionmetadata.QueryItem) []*apiv1.InspectionQuery {
	res := make([]*apiv1.InspectionQuery, 0, len(queries))
	for _, q := range queries {
		item := &apiv1.InspectionQuery{
			Id:                   proto.String(q.Id),
			Name:                 proto.String(q.Name),
			Query:                proto.String(q.Query),
			Incomplete:           proto.Bool(q.Incomplete),
			Pending:              proto.Bool(q.Pending),
			EstimatedCountPreset: convertPreset(q.Preset).Enum(),
		}
		if q.EstimatedCount != nil {
			item.EstimatedCount = proto.Int64(*q.EstimatedCount)
		}
		res = append(res, item)
	}
	return res
}

func convertFormFields(fields []inspectionmetadata.ParameterFormField) []*apiv1.FormField {
	res := make([]*apiv1.FormField, 0, len(fields))
	for _, field := range fields {
		base := inspectionmetadata.GetParameterFormFieldBase(field)
		f := &apiv1.FormField{
			Id:          proto.String(base.ID),
			Label:       proto.String(base.Label),
			Description: proto.String(base.Description),
			Hint:        proto.String(base.Hint),
			HintType:    convertHintType(base.HintType).Enum(),
		}
		switch v := field.(type) {
		case inspectionmetadata.GroupParameterFormField:
			f.Kind = &apiv1.FormField_Group{
				Group: &apiv1.GroupFormField{
					Children:           convertFormFields(v.Children),
					Collapsible:        proto.Bool(v.Collapsible),
					CollapsedByDefault: proto.Bool(v.CollapsedByDefault),
				},
			}
		case inspectionmetadata.TextParameterFormField:
			f.Kind = &apiv1.FormField_Text{
				Text: &apiv1.TextFormField{
					Readonly:         proto.Bool(v.Readonly),
					DefaultValue:     proto.String(v.Default),
					Suggestions:      v.Suggestions,
					ValidationTiming: convertValidationTiming(v.ValidationTiming).Enum(),
				},
			}
		case inspectionmetadata.FileParameterFormField:
			tokenID := ""
			if v.Token != nil {
				tokenID = v.Token.GetID()
			}
			f.Kind = &apiv1.FormField_File{
				File: &apiv1.FileFormField{
					TokenId: proto.String(tokenID),
					Status:  convertUploadStatus(v.Status).Enum(),
				},
			}
		case inspectionmetadata.SetParameterFormField:
			options := make([]*apiv1.SetOption, 0, len(v.Options))
			for _, opt := range v.Options {
				options = append(options, &apiv1.SetOption{
					Id:          proto.String(opt.ID),
					Description: proto.String(opt.Description),
				})
			}
			f.Kind = &apiv1.FormField_Set{
				Set: &apiv1.SetFormField{
					Options:          options,
					DefaultValues:    v.Default,
					AllowCustomValue: proto.Bool(v.AllowCustomValue),
					AllowAddAll:      proto.Bool(v.AllowAddAll),
					AllowRemoveAll:   proto.Bool(v.AllowRemoveAll),
				},
			}
		}
		res = append(res, f)
	}
	return res
}

func convertHintType(ht inspectionmetadata.ParameterHintType) apiv1.ParameterHintType {
	switch ht {
	case inspectionmetadata.None:
		return apiv1.ParameterHintType_PARAMETER_HINT_TYPE_NONE
	case inspectionmetadata.Error:
		return apiv1.ParameterHintType_PARAMETER_HINT_TYPE_ERROR
	case inspectionmetadata.Warning:
		return apiv1.ParameterHintType_PARAMETER_HINT_TYPE_WARNING
	case inspectionmetadata.Info:
		return apiv1.ParameterHintType_PARAMETER_HINT_TYPE_INFO
	default:
		return apiv1.ParameterHintType_PARAMETER_HINT_TYPE_UNSPECIFIED
	}
}

func convertValidationTiming(timing inspectionmetadata.TextFormValidationTimingType) apiv1.ValidationTiming {
	switch timing {
	case inspectionmetadata.Change:
		return apiv1.ValidationTiming_VALIDATION_TIMING_CHANGE
	case inspectionmetadata.Blur:
		return apiv1.ValidationTiming_VALIDATION_TIMING_BLUR
	default:
		return apiv1.ValidationTiming_VALIDATION_TIMING_UNSPECIFIED
	}
}

func convertUploadStatus(st upload.UploadStatus) apiv1.UploadStatus {
	switch st {
	case upload.UploadStatusWaiting:
		return apiv1.UploadStatus_UPLOAD_STATUS_WAITING
	case upload.UploadStatusUploading:
		return apiv1.UploadStatus_UPLOAD_STATUS_UPLOADING
	case upload.UploadStatusVerifying:
		return apiv1.UploadStatus_UPLOAD_STATUS_VERIFYING
	case upload.UploadStatusCompleted:
		return apiv1.UploadStatus_UPLOAD_STATUS_DONE
	default:
		return apiv1.UploadStatus_UPLOAD_STATUS_UNSPECIFIED
	}
}

func convertParametersToMap(params *apiv1.InspectionParameters) map[string]any {
	values := map[string]any{}
	if params == nil {
		return values
	}
	for _, p := range params.GetParameters() {
		id := p.GetId()
		switch v := p.GetValue().(type) {
		case *apiv1.ParameterValue_TextValue:
			values[id] = v.TextValue.GetValue()
		case *apiv1.ParameterValue_SetValue:
			values[id] = v.SetValue.GetValues()
		case *apiv1.ParameterValue_FileValue:
			values[id] = v.FileValue.GetToken()
		}
	}
	if params.GetTimezoneShiftHours() != 0 {
		values["timezoneShiftHours"] = params.GetTimezoneShiftHours()
	}
	return values
}
