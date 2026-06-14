/**
 * Copyright 2026 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { TestBed } from '@angular/core/testing';
import { SelectionManagerV2 } from 'src/app/services/selection-manager-v2.service';
import { InspectionDataStoreV2 } from 'src/app/services/inspection-data-store-v2.service';
import { InspectionDataV2 } from 'src/app/store/domain/inspection-data';
import { InternPoolStore } from 'src/app/store/domain/intern-pool-store';
import { StyleStore } from 'src/app/store/domain/style-store';
import { LogStore } from 'src/app/store/domain/log-store';
import { TimelineStore } from 'src/app/store/domain/timeline-store';

describe('SelectionManagerV2', () => {
  let service: SelectionManagerV2;
  let dataStore: InspectionDataStoreV2;
  let logStore: LogStore;
  let timelineStore: TimelineStore;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [InspectionDataStoreV2, SelectionManagerV2],
    });

    dataStore = TestBed.inject(InspectionDataStoreV2);
    service = TestBed.inject(SelectionManagerV2);

    const internPool = InternPoolStore.create();
    const styleStore = new StyleStore();
    logStore = LogStore.create(internPool, styleStore);
    timelineStore = TimelineStore.create(internPool, styleStore, logStore);

    styleStore.addSeverities([
      {
        id: 1,
        label: 'Info',
        shortLabel: 'I',
        backgroundColor: { r: 0, g: 0, b: 1, a: 1 },
        foregroundColor: { r: 1, g: 1, b: 1, a: 1 },
        order: 1,
      },
    ]);

    styleStore.addLogTypes([
      {
        id: 2,
        label: 'K8sEvent',
        description: 'Kubernetes Event',
        backgroundColor: { r: 0, g: 1, b: 0, a: 1 },
        foregroundColor: { r: 1, g: 1, b: 1, a: 1 },
      },
    ]);

    styleStore.addTimelineTypes([
      {
        id: 1,
        label: 'APIVersion',
        description: 'Kubernetes API Version',
        icon: 'settings',
        backgroundColor: { r: 0, g: 0, b: 1, a: 1 },
        foregroundColor: { r: 1, g: 1, b: 1, a: 1 },
        typeChipBackgroundColor: { r: 0, g: 0, b: 1, a: 1 },
        typeChipForegroundColor: { r: 1, g: 1, b: 1, a: 1 },
        visible: true,
        sortPriority: 1,
        height: 1,
      },
      {
        id: 2,
        label: 'Kind',
        description: 'Kubernetes Resource Kind',
        icon: 'workspaces',
        backgroundColor: { r: 0, g: 1, b: 0, a: 1 },
        foregroundColor: { r: 1, g: 1, b: 1, a: 1 },
        typeChipBackgroundColor: { r: 0, g: 1, b: 0, a: 1 },
        typeChipForegroundColor: { r: 1, g: 1, b: 1, a: 1 },
        visible: true,
        sortPriority: 2,
        height: 1,
      },
      {
        id: 3,
        label: 'Namespace',
        description: 'Kubernetes Namespace',
        icon: 'folder',
        backgroundColor: { r: 0.5, g: 0, b: 0.5, a: 1 },
        foregroundColor: { r: 1, g: 1, b: 1, a: 1 },
        typeChipBackgroundColor: { r: 0.5, g: 0, b: 0.5, a: 1 },
        typeChipForegroundColor: { r: 1, g: 1, b: 1, a: 1 },
        visible: true,
        sortPriority: 3,
        height: 1,
      },
      {
        id: 4,
        label: 'Resource',
        description: 'Kubernetes Resource Instance',
        icon: 'description',
        backgroundColor: { r: 1, g: 0.5, b: 0, a: 1 },
        foregroundColor: { r: 1, g: 1, b: 1, a: 1 },
        typeChipBackgroundColor: { r: 1, g: 0.5, b: 0, a: 1 },
        typeChipForegroundColor: { r: 1, g: 1, b: 1, a: 1 },
        visible: true,
        sortPriority: 4,
        height: 1,
      },
    ]);

    styleStore.addVerbs([
      {
        id: 1,
        label: 'CREATE',
        backgroundColor: { r: 0, g: 0.5, b: 1, a: 1 },
        foregroundColor: { r: 1, g: 1, b: 1, a: 1 },
        visible: true,
      },
    ]);

    styleStore.addRevisionStates([
      {
        id: 1,
        label: 'Existing',
        icon: '',
        description: 'Resource exists',
        backgroundColor: { r: 0, g: 0, b: 1, a: 1 },
        style: 0,
      },
    ]);

    internPool.addStrings([
      { id: 1, value: 'v1' },
      { id: 2, value: 'Pod' },
      { id: 3, value: 'default' },
      { id: 4, value: 'mock-pod-1' },
      { id: 5, value: 'Pod created successfully' },
      { id: 6, value: 'system:serviceaccount:kube-system:generic' },
    ]);

    logStore.initialize(
      [
        {
          id: 1,
          ts: 1700000000000000000n,
          logTypeId: 2,
          severityTypeId: 1,
          summaryStringId: 5,
        },
      ],
      1,
    );

    timelineStore.initialize(
      [
        {
          id: 1,
          timelineTypeId: 1,
          nameStringId: 1,
          parentTimelineId: 0,
          revisionIds: [],
          eventIds: [],
        },
        {
          id: 2,
          timelineTypeId: 2,
          nameStringId: 2,
          parentTimelineId: 1,
          revisionIds: [],
          eventIds: [],
        },
        {
          id: 3,
          timelineTypeId: 3,
          nameStringId: 3,
          parentTimelineId: 2,
          revisionIds: [],
          eventIds: [],
        },
        {
          id: 4,
          timelineTypeId: 4,
          nameStringId: 4,
          parentTimelineId: 3,
          revisionIds: [1],
          eventIds: [1],
        },
      ],
      4,
      [
        {
          id: 1,
          logId: 1,
          changedTime: 1700000000000000000n,
          principalStringId: 6,
          verbTypeId: 1,
          stateTypeId: 1,
        },
      ],
      1,
      [
        {
          id: 1,
          logId: 1,
        },
      ],
      1,
    );

    const mockData: InspectionDataV2 = {
      internPool,
      styleStore,
      logStore,
      timelineStore,
    };
    dataStore.setNewInspectionData(mockData);
  });

  it('should initialize with default null selections', () => {
    expect(service.selectedTimeline()).toBeNull();
    expect(service.highlightedTimeline()).toBeNull();
    expect(service.selectedRevision()).toBeNull();
    expect(service.selectedLog()).toBeNull();
    expect(service.selectedLogIndex()).toBe(-1);
  });

  it('should handle timeline selection', () => {
    const timelines = timelineStore.timelines;
    const targetTimeline = timelines.find((t) => t.id === 4)!;

    service.onSelectTimeline(targetTimeline);
    expect(service.selectedTimeline()).toBe(targetTimeline);
  });

  it('should include descendants in selectedTimelinesWithChildren when timelineSelectionShouldIncludeChildren is true', () => {
    const timelines = timelineStore.timelines;
    // Timeline 3 is 'Namespace: default', which parent of Timeline 4 ('Resource: mock-pod-1')
    const timeline3 = timelines.find((t) => t.id === 3)!;
    const timeline4 = timelines.find((t) => t.id === 4)!;

    service.timelineSelectionShouldIncludeChildren.set(true);
    service.onSelectTimeline(timeline3);

    const selection = service.selectedTimelinesWithChildren();
    expect(selection).toContain(timeline3);
    expect(selection).toContain(timeline4);
  });

  it('should NOT include descendants in selectedTimelinesWithChildren when timelineSelectionShouldIncludeChildren is false', () => {
    const timelines = timelineStore.timelines;
    const timeline3 = timelines.find((t) => t.id === 3)!;
    const timeline4 = timelines.find((t) => t.id === 4)!;

    service.timelineSelectionShouldIncludeChildren.set(false);
    service.onSelectTimeline(timeline3);

    const selection = service.selectedTimelinesWithChildren();
    expect(selection).toContain(timeline3);
    expect(selection).not.toContain(timeline4);
  });

  it('should sync timeline and revision selections when a log is selected', () => {
    const logs = Array.from(logStore.logs());
    const targetLog = logs[0];

    // When log is selected, it should automatically resolve target resource/revision in the hierarchy
    service.onSelectLog(targetLog);

    expect(service.selectedLog()?.id).toBe(targetLog.id);
    // Log 1 resides on Resource Timeline (ID 4) inside Revision 1
    expect(service.selectedTimeline()?.id).toBe(4);
    expect(service.selectedRevision()?.logIndex).toBe(targetLog.logIndex);
  });

  it('should automatically clear log/revision selection if newly selected timeline does not contain them (sync effect)', () => {
    const logs = Array.from(logStore.logs());
    const targetLog = logs[0];
    const timelines = timelineStore.timelines;
    const unrelatedTimeline = timelines.find((t) => t.id === 1)!;

    // Select log (will select Log 1, Timeline 4, Revision 1)
    service.onSelectLog(targetLog);
    expect(service.selectedLog()?.id).toBe(targetLog.id);
    expect(service.selectedTimeline()?.id).toBe(4);

    // Select unrelated timeline
    service.onSelectTimeline(unrelatedTimeline);

    // Expectations after effect propagates
    TestBed.tick();

    expect(service.selectedTimeline()).toBe(unrelatedTimeline);
    // Since Log 1 is not in Timeline 1, selections must be cleared
    expect(service.selectedLog()).toBeNull();
    expect(service.selectedRevision()).toBeNull();
  });
});
