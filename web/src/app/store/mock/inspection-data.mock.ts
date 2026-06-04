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

import { InspectionDataV2 } from 'src/app/store/domain/inspection-data';
import { InternPoolStore } from 'src/app/store/domain/intern-pool-store';
import { StyleStore } from 'src/app/store/domain/style-store';
import { LogStore } from 'src/app/store/domain/log-store';
import {
  TimelineStore,
  TimelineDTO,
  RevisionDTO,
  EventDTO,
} from 'src/app/store/domain/timeline-store';
import { LogDTO } from 'src/app/store/domain/log-store';
import { MetadataStore } from 'src/app/store/domain/metadata-store';
import { toBinary } from '@bufbuild/protobuf';
import { InternedStructSchema } from 'src/app/generated/khifile/shared_pb';
import {
  parseTimestampString,
  parseUnixSeconds,
  parseHexColor,
  objectToInternedStruct,
  initializeMockIconAtlas,
  MockInternIdState,
} from 'src/app/store/mock/mock-util';

/**
 * Creates a mock instance of InspectionDataV2 for debugging and testing purposes.
 *
 * This function encapsulates standard dummy timelines, logs, and revisions
 * to quickly visualize and verify component interactions.
 *
 * @returns A promise resolving to a populated mock inspection data instance.
 */
export async function createMockInspectionDataV2(): Promise<InspectionDataV2> {
  const internPool = InternPoolStore.create();
  const styleStore = new StyleStore();
  const logStore = new LogStore(internPool, styleStore);
  const timelineStore = new TimelineStore(internPool, styleStore, logStore);

  const idState: MockInternIdState = {
    nextStringId: 1,
    nextFieldSetId: 1,
  };

  // --- 1. Setup Styles (Ordered as requested & matched with Go enum colors) ---

  // 1-1. Severity Definitions
  styleStore.addSeverities([
    {
      id: 1,
      label: 'Info',
      shortLabel: 'I',
      backgroundColor: parseHexColor('#0000FF'),
      foregroundColor: parseHexColor('#FFFFFF'),
      order: 1,
    },
    {
      id: 2,
      label: 'Warning',
      shortLabel: 'W',
      backgroundColor: parseHexColor('#FFAA44'),
      foregroundColor: parseHexColor('#FFFFFF'),
      order: 2,
    },
    {
      id: 3,
      label: 'Error',
      shortLabel: 'E',
      backgroundColor: parseHexColor('#FF3935'),
      foregroundColor: parseHexColor('#FFFFFF'),
      order: 3,
    },
    {
      id: 4,
      label: 'Fatal',
      shortLabel: 'F',
      backgroundColor: parseHexColor('#AA66AA'),
      foregroundColor: parseHexColor('#FFFFFF'),
      order: 4,
    },
  ]);

  // 1-2. LogType Definitions
  styleStore.addLogTypes([
    {
      id: 1,
      label: 'K8sAudit',
      description: 'Kubernetes Audit Log',
      backgroundColor: parseHexColor('#000000'),
      foregroundColor: parseHexColor('#FFFFFF'),
    },
    {
      id: 2,
      label: 'K8sEvent',
      description: 'Kubernetes Event',
      backgroundColor: parseHexColor('#3fb549'),
      foregroundColor: parseHexColor('#FFFFFF'),
    },
  ]);

  // 1-3. TimelineType Definitions
  styleStore.addTimelineTypes([
    {
      id: 1,
      label: 'APIVersion',
      description: 'Kubernetes API Version',
      icon: 'settings',
      backgroundColor: parseHexColor('#0078D7'),
      foregroundColor: parseHexColor('#FFFFFF'),
      typeChipBackgroundColor: parseHexColor('#0078D7'),
      visible: true,
      sortPriority: 1,
      height: 0.7,
    },
    {
      id: 2,
      label: 'Kind',
      description: 'Kubernetes Resource Kind',
      icon: 'workspaces',
      backgroundColor: parseHexColor('#3f51b5'),
      foregroundColor: parseHexColor('#FFFFFF'),
      typeChipBackgroundColor: parseHexColor('#3f51b5'),
      visible: true,
      sortPriority: 2,
      height: 0.7,
    },
    {
      id: 3,
      label: 'Namespace',
      description: 'Kubernetes Namespace',
      icon: 'folder',
      backgroundColor: parseHexColor('#646464'),
      foregroundColor: parseHexColor('#FFFFFF'),
      typeChipBackgroundColor: parseHexColor('#646464'),
      visible: true,
      sortPriority: 3,
      height: 0.7,
    },
    {
      id: 4,
      label: 'Resource',
      description: 'Kubernetes Resource Instance',
      icon: 'description',
      backgroundColor: parseHexColor('#c8c8c8'),
      foregroundColor: parseHexColor('#323232'),
      typeChipBackgroundColor: parseHexColor('#c8c8c8'),
      visible: true,
      sortPriority: 4,
      height: 1.0,
    },
    {
      id: 5,
      label: 'Subresource',
      description: 'Kubernetes Subresource',
      icon: 'page_info',
      backgroundColor: parseHexColor('#f5f5f5'),
      foregroundColor: parseHexColor('#646464'),
      typeChipBackgroundColor: parseHexColor('#ffffff'),
      visible: true,
      sortPriority: 5,
      height: 0.6,
    },
    {
      id: 6,
      label: 'Cloud Composer',
      description: 'Cloud Composer related resources',
      icon: 'page_info',
      backgroundColor: parseHexColor('#ff1111'),
      foregroundColor: parseHexColor('#ffffff'),
      typeChipBackgroundColor: parseHexColor('#ffffff'),
      visible: true,
      sortPriority: 0,
      height: 0.7,
    },
    {
      id: 7,
      label: 'Airflow worker logs',
      description: 'Cloud Composer worker logs',
      icon: 'page_info',
      backgroundColor: parseHexColor('#f0fff0'),
      foregroundColor: parseHexColor('#000000'),
      typeChipBackgroundColor: parseHexColor('#ffffff'),
      visible: true,
      sortPriority: 0,
      height: 1,
    },
  ]);

  // 1-4. Verb Definitions
  styleStore.addVerbs([
    {
      id: 1,
      label: 'CREATE',
      backgroundColor: parseHexColor('#1E88E5'),
      foregroundColor: parseHexColor('#FFFFFF'),
      visible: true,
    },
    {
      id: 2,
      label: 'UPDATE',
      backgroundColor: parseHexColor('#FDD835'),
      foregroundColor: parseHexColor('#FFFFFF'),
      visible: true,
    },
    {
      id: 3,
      label: 'DELETE',
      backgroundColor: parseHexColor('#F54945'),
      foregroundColor: parseHexColor('#FFFFFF'),
      visible: true,
    },
  ]);

  // 1-5. RevisionState Definitions
  styleStore.addRevisionStates([
    {
      id: 1,
      label: 'Existing',
      icon: 'step_over',
      description: 'Resource exists',
      backgroundColor: parseHexColor('#0000FF'),
      style: 0,
    },
    {
      id: 2,
      label: 'Deleted',
      icon: 'skull',
      description: 'Resource deleted',
      backgroundColor: parseHexColor('#CC0000'),
      style: 0,
    },
  ]);

  // Initialize real IconAtlas safely
  await initializeMockIconAtlas(styleStore);

  // --- 2. Define Static Strings ---
  const apiVersionStringId = idState.nextStringId++;
  const kindStringId = idState.nextStringId++;
  const namespaceStringId = idState.nextStringId++;
  const podNameStringId = idState.nextStringId++;
  const subresourceStringId = idState.nextStringId++;

  const logSummaryStringId = idState.nextStringId++;
  const principalStringId = idState.nextStringId++;

  // Additional virtual timeline strings
  const podNameMockPod2StringId = idState.nextStringId++;
  const apiVersionAppsV1StringId = idState.nextStringId++;
  const kindDeploymentStringId = idState.nextStringId++;
  const namespaceKubeSystemStringId = idState.nextStringId++;
  const deploymentNameCorednsStringId = idState.nextStringId++;
  const subresourceStatusStringId = idState.nextStringId++;

  const composerTimelineStringId = idState.nextStringId++;
  const airflowWorkerTimelineStringId = idState.nextStringId++;

  // Dynamic timeline strings
  const apiStrings: number[] = [];
  const kindStrings: number[] = [];
  const nsStrings: number[] = [];
  const resStrings: number[] = [];
  const subStrings: number[] = [];

  const stringsToRegister: { id: number; value: string }[] = [
    { id: apiVersionStringId, value: 'v1' },
    { id: kindStringId, value: 'Pod' },
    { id: namespaceStringId, value: 'default' },
    { id: podNameStringId, value: 'mock-pod-1' },
    { id: subresourceStringId, value: 'scale' },
    { id: logSummaryStringId, value: 'Pod created successfully' },
    {
      id: principalStringId,
      value: 'system:serviceaccount:kube-system:generic',
    },
    { id: podNameMockPod2StringId, value: 'mock-pod-2' },
    { id: apiVersionAppsV1StringId, value: 'apps/v1' },
    { id: kindDeploymentStringId, value: 'Deployment' },
    { id: namespaceKubeSystemStringId, value: 'kube-system' },
    { id: deploymentNameCorednsStringId, value: 'coredns' },
    { id: subresourceStatusStringId, value: 'status' },
    { id: composerTimelineStringId, value: 'Cloud Composer' },
    { id: airflowWorkerTimelineStringId, value: 'Airflow worker logs' },
  ];

  for (let i = 0; i < 10; i++) {
    const id = idState.nextStringId++;
    apiStrings.push(id);
    stringsToRegister.push({ id, value: `apiversion-${i}` });
  }
  for (let i = 0; i < 10; i++) {
    const id = idState.nextStringId++;
    kindStrings.push(id);
    stringsToRegister.push({ id, value: `kind-${i}` });
  }
  for (let i = 0; i < 10; i++) {
    const id = idState.nextStringId++;
    nsStrings.push(id);
    stringsToRegister.push({ id, value: `ns-${i}` });
  }
  for (let i = 0; i < 10; i++) {
    const id = idState.nextStringId++;
    resStrings.push(id);
    stringsToRegister.push({ id, value: `res-${i}` });
  }
  for (let i = 0; i < 10; i++) {
    const id = idState.nextStringId++;
    subStrings.push(id);
    stringsToRegister.push({ id, value: `sub-${i}` });
  }

  internPool.addStrings(stringsToRegister);

  // --- 3. Define Mock Entities ---
  const timestampString = '2026-05-13T09:00:00Z';
  const timestamp = parseTimestampString(timestampString);
  const unixSeconds = parseUnixSeconds(timestampString);

  const logBodyStruct = objectToInternedStruct(
    {
      message: 'Pod created successfully',
      reason: 'Created',
      source: { component: 'kubelet', host: 'node-1' },
    },
    internPool,
    idState,
  );
  const logBody = toBinary(InternedStructSchema, logBodyStruct);

  const mockLogs: LogDTO[] = [
    {
      id: 1,
      ts: timestamp - 65_000_000_000n, // Set before dynamic start (timestamp - 60s) to maintain ascending order
      logTypeId: 2,
      severityTypeId: 1,
      summaryStringId: logSummaryStringId,
      body: logBody,
    },
  ];

  const revisionBodyStruct = objectToInternedStruct(
    {
      spec: { containers: [{ name: 'app', image: 'nginx:latest' }] },
      status: { phase: 'Running' },
    },
    internPool,
    idState,
  );
  const revisionBody = toBinary(InternedStructSchema, revisionBodyStruct);

  const mockRevisions: RevisionDTO[] = [
    {
      id: 1,
      logId: 1,
      changedTime: timestamp - 65_000_000_000n,
      principalStringId,
      verbTypeId: 1,
      stateTypeId: 1, // Existing
      body: revisionBody,
    },
  ];

  const mockEvents: EventDTO[] = [
    {
      id: 1,
      logId: 1,
    },
  ];

  // Generate static and dynamic timelines
  const timelines: TimelineDTO[] = [
    {
      id: 1,
      timelineTypeId: 6,
      nameStringId: composerTimelineStringId,
      parentTimelineId: 0,
      revisionIds: [],
      eventIds: [],
    },
    {
      id: 2,
      timelineTypeId: 7,
      nameStringId: airflowWorkerTimelineStringId,
      parentTimelineId: 1,
      revisionIds: [],
      eventIds: [1],
    },
    {
      id: 3,
      timelineTypeId: 7,
      nameStringId: airflowWorkerTimelineStringId,
      parentTimelineId: 1,
      revisionIds: [],
      eventIds: [1],
    },
    {
      id: 4,
      timelineTypeId: 7,
      nameStringId: airflowWorkerTimelineStringId,
      parentTimelineId: 1,
      revisionIds: [],
      eventIds: [1],
    },
    {
      id: 5,
      timelineTypeId: 1, // APIVersion
      nameStringId: apiVersionStringId,
      parentTimelineId: 0,
      revisionIds: [],
      eventIds: [],
    },
    {
      id: 6,
      timelineTypeId: 2, // Kind
      nameStringId: kindStringId,
      parentTimelineId: 5,
      revisionIds: [],
      eventIds: [],
    },
    {
      id: 7,
      timelineTypeId: 3, // Namespace
      nameStringId: namespaceStringId,
      parentTimelineId: 6,
      revisionIds: [],
      eventIds: [],
    },
    {
      id: 8,
      timelineTypeId: 4, // Resource
      nameStringId: podNameStringId,
      parentTimelineId: 7,
      revisionIds: [1],
      eventIds: [1],
    },
    {
      id: 9,
      timelineTypeId: 5, // SubResource
      nameStringId: subresourceStringId,
      parentTimelineId: 8,
      revisionIds: [],
      eventIds: [],
    },
    {
      id: 10,
      timelineTypeId: 4, // Resource
      nameStringId: podNameMockPod2StringId,
      parentTimelineId: 7,
      revisionIds: [],
      eventIds: [],
    },
    {
      id: 11,
      timelineTypeId: 1, // APIVersion
      nameStringId: apiVersionAppsV1StringId,
      parentTimelineId: 0,
      revisionIds: [],
      eventIds: [],
    },
    {
      id: 12,
      timelineTypeId: 2, // Kind
      nameStringId: kindDeploymentStringId,
      parentTimelineId: 11,
      revisionIds: [],
      eventIds: [],
    },
    {
      id: 13,
      timelineTypeId: 3, // Namespace
      nameStringId: namespaceKubeSystemStringId,
      parentTimelineId: 12,
      revisionIds: [],
      eventIds: [],
    },
    {
      id: 14,
      timelineTypeId: 4, // Resource
      nameStringId: deploymentNameCorednsStringId,
      parentTimelineId: 13,
      revisionIds: [],
      eventIds: [],
    },
    {
      id: 15,
      timelineTypeId: 5, // SubResource
      nameStringId: subresourceStatusStringId,
      parentTimelineId: 14,
      revisionIds: [],
      eventIds: [],
    },
  ];

  let nextTimelineId = 16;
  let mockItemIndex = 2;

  const resourceTimelineIds: number[] = [];
  const subresourceTimelineIds: number[] = [];

  const revisionIdsMap = new Map<number, number[]>();
  const eventIdsMap = new Map<number, number[]>();

  for (let a = 0; a < 10; a++) {
    const apiId = nextTimelineId++;
    timelines.push({
      id: apiId,
      timelineTypeId: 1, // APIVersion
      nameStringId: apiStrings[a],
      parentTimelineId: 0,
      revisionIds: [],
      eventIds: [],
    });

    for (let k = 0; k < 10; k++) {
      const kindId = nextTimelineId++;
      timelines.push({
        id: kindId,
        timelineTypeId: 2, // Kind
        nameStringId: kindStrings[k],
        parentTimelineId: apiId,
        revisionIds: [],
        eventIds: [],
      });

      for (let n = 0; n < 10; n++) {
        const nsId = nextTimelineId++;
        timelines.push({
          id: nsId,
          timelineTypeId: 3, // Namespace
          nameStringId: nsStrings[n],
          parentTimelineId: kindId,
          revisionIds: [],
          eventIds: [],
        });

        for (let r = 0; r < 10; r++) {
          const resId = nextTimelineId++;
          resourceTimelineIds.push(resId);

          const revisionIds: number[] = [];
          const eventIds: number[] = [];
          revisionIdsMap.set(resId, revisionIds);
          eventIdsMap.set(resId, eventIds);

          timelines.push({
            id: resId,
            timelineTypeId: 4, // Resource
            nameStringId: resStrings[r],
            parentTimelineId: nsId,
            revisionIds,
            eventIds,
          });

          for (let s = 0; s < 10; s++) {
            const subId = nextTimelineId++;
            subresourceTimelineIds.push(subId);

            const subRevisionIds: number[] = [];
            const subEventIds: number[] = [];
            revisionIdsMap.set(subId, subRevisionIds);
            eventIdsMap.set(subId, subEventIds);

            timelines.push({
              id: subId,
              timelineTypeId: 5, // Subresource
              nameStringId: subStrings[s],
              parentTimelineId: resId,
              revisionIds: subRevisionIds,
              eventIds: subEventIds,
            });
          }
        }
      }
    }
  }

  // Timeline span: 60 seconds
  const startTs = timestamp - 60_000_000_000n;

  // PHASE 0: 1st Revision/Event for each Resource timeline (spread across timestamp - 50s to -35s)
  for (let i = 0; i < resourceTimelineIds.length; i++) {
    const resId = resourceTimelineIds[i];
    const currentRevisionId = mockItemIndex;
    const currentEventId = mockItemIndex;
    const currentLogId = mockItemIndex;
    mockItemIndex++;

    const currentLogTs = startTs + 10_000_000_000n + BigInt(i) * 1_500_000n;

    mockLogs.push({
      id: currentLogId,
      ts: currentLogTs,
      logTypeId: 2, // K8sEvent
      severityTypeId: (mockItemIndex % 3) + 1,
      summaryStringId: logSummaryStringId,
      body: logBody,
    });

    mockRevisions.push({
      id: currentRevisionId,
      logId: currentLogId,
      changedTime: currentLogTs - 50_000n,
      principalStringId,
      verbTypeId: (mockItemIndex % 3) + 1,
      stateTypeId: (mockItemIndex % 2) + 1,
      body: revisionBody,
    });

    mockEvents.push({
      id: currentEventId,
      logId: currentLogId,
    });

    revisionIdsMap.get(resId)!.push(currentRevisionId);
    eventIdsMap.get(resId)!.push(currentEventId);
  }

  // PHASE 1: 2nd Revision/Event for each Resource timeline (spread across timestamp - 30s to -15s)
  for (let i = 0; i < resourceTimelineIds.length; i++) {
    const resId = resourceTimelineIds[i];
    const currentRevisionId = mockItemIndex;
    const currentEventId = mockItemIndex;
    const currentLogId = mockItemIndex;
    mockItemIndex++;

    const currentLogTs = startTs + 30_000_000_000n + BigInt(i) * 1_500_000n;

    mockLogs.push({
      id: currentLogId,
      ts: currentLogTs,
      logTypeId: 2, // K8sEvent
      severityTypeId: (mockItemIndex % 3) + 1,
      summaryStringId: logSummaryStringId,
      body: logBody,
    });

    mockRevisions.push({
      id: currentRevisionId,
      logId: currentLogId,
      changedTime: currentLogTs - 50_000n,
      principalStringId,
      verbTypeId: (mockItemIndex % 3) + 1,
      stateTypeId: (mockItemIndex % 2) + 1,
      body: revisionBody,
    });

    mockEvents.push({
      id: currentEventId,
      logId: currentLogId,
    });

    revisionIdsMap.get(resId)!.push(currentRevisionId);
    eventIdsMap.get(resId)!.push(currentEventId);
  }

  // PHASE 2: 3rd Revision/Event for Resources + 1st Revision/Event for Subresources (spread across timestamp - 10s to -1s)
  let phase2Index = 0;

  // 3rd Revision/Event for Resource
  for (let i = 0; i < resourceTimelineIds.length; i++) {
    const resId = resourceTimelineIds[i];
    const currentRevisionId = mockItemIndex;
    const currentEventId = mockItemIndex;
    const currentLogId = mockItemIndex;
    mockItemIndex++;

    const currentLogTs =
      startTs + 50_000_000_000n + BigInt(phase2Index) * 80_000n;
    phase2Index++;

    mockLogs.push({
      id: currentLogId,
      ts: currentLogTs,
      logTypeId: 2,
      severityTypeId: (mockItemIndex % 3) + 1,
      summaryStringId: logSummaryStringId,
      body: logBody,
    });

    mockRevisions.push({
      id: currentRevisionId,
      logId: currentLogId,
      changedTime: currentLogTs - 50_000n,
      principalStringId,
      verbTypeId: (mockItemIndex % 3) + 1,
      stateTypeId: (mockItemIndex % 2) + 1,
      body: revisionBody,
    });

    mockEvents.push({
      id: currentEventId,
      logId: currentLogId,
    });

    revisionIdsMap.get(resId)!.push(currentRevisionId);
    eventIdsMap.get(resId)!.push(currentEventId);
  }

  // 1st Revision/Event for Subresource
  for (let i = 0; i < subresourceTimelineIds.length; i++) {
    const subId = subresourceTimelineIds[i];
    const currentRevisionId = mockItemIndex;
    const currentEventId = mockItemIndex;
    const currentLogId = mockItemIndex;
    mockItemIndex++;

    const currentLogTs =
      startTs + 50_000_000_000n + BigInt(phase2Index) * 80_000n;
    phase2Index++;

    mockLogs.push({
      id: currentLogId,
      ts: currentLogTs,
      logTypeId: 2,
      severityTypeId: (mockItemIndex % 3) + 1,
      summaryStringId: logSummaryStringId,
      body: logBody,
    });

    mockRevisions.push({
      id: currentRevisionId,
      logId: currentLogId,
      changedTime: currentLogTs - 50_000n,
      principalStringId,
      verbTypeId: (mockItemIndex % 3) + 1,
      stateTypeId: (mockItemIndex % 2) + 1,
      body: revisionBody,
    });

    mockEvents.push({
      id: currentEventId,
      logId: currentLogId,
    });

    revisionIdsMap.get(subId)!.push(currentRevisionId);
    eventIdsMap.get(subId)!.push(currentEventId);
  }

  logStore.initialize(mockLogs, mockLogs.length);

  timelineStore.initialize(
    timelines,
    timelines.length,
    mockRevisions,
    mockRevisions.length,
    mockEvents,
    mockEvents.length,
  );

  // --- 4. Define Header Metadata ---
  const metadata: MetadataStore = {
    header: {
      inspectionType: 'Mock',
      inspectionName: 'Debug Mock Inspection',
      inspectTimeUnixSeconds: unixSeconds,
      startTimeUnixSeconds: unixSeconds - 60,
      endTimeUnixSeconds: unixSeconds,
      suggestedFilename: 'mock-debug.khi',
      fileSize: 2048,
    },
    queries: [],
  };

  return {
    internPool,
    styleStore,
    logStore,
    timelineStore,
    metadata,
  };
}
