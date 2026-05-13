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
} from 'src/app/store/domain/timeline-store';
import { MetadataStore } from 'src/app/store/domain/metadata-store';
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
  const internPool = new InternPoolStore();
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
      backgroundColor: parseHexColor('#0078D7'),
      foregroundColor: parseHexColor('#FFFFFF'),
      visible: true,
      sortPriority: 1,
    },
    {
      id: 2,
      label: 'Kind',
      description: 'Kubernetes Resource Kind',
      icon: 'workspaces',
      backgroundColor: parseHexColor('#3f51b5'),
      foregroundColor: parseHexColor('#FFFFFF'),
      visible: true,
      sortPriority: 2,
    },
    {
      id: 3,
      label: 'Namespace',
      description: 'Kubernetes Namespace',
      icon: 'folder',
      backgroundColor: parseHexColor('#646464'),
      foregroundColor: parseHexColor('#FFFFFF'),
      visible: true,
      sortPriority: 3,
    },
    {
      id: 4,
      label: 'Resource',
      description: 'Kubernetes Resource Instance',
      backgroundColor: parseHexColor('#FD7E14'),
      foregroundColor: parseHexColor('#FFFFFF'),
      visible: true,
      sortPriority: 4,
    },
    {
      id: 5,
      label: 'Subresource',
      description: 'Kubernetes Subresource',
      backgroundColor: parseHexColor('#20C997'),
      foregroundColor: parseHexColor('#FFFFFF'),
      visible: true,
      sortPriority: 5,
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

  const logBody = objectToInternedStruct(
    {
      message: 'Pod created successfully',
      reason: 'Created',
      source: { component: 'kubelet', host: 'node-1' },
    },
    internPool,
    idState,
  );

  logStore.initialize(
    [
      {
        id: 1,
        ts: timestamp - 1000000000n,
        logTypeId: 2,
        severityTypeId: 1,
        summaryStringId: logSummaryStringId,
        body: logBody,
      },
    ],
    1,
  );

  const revisionBody = objectToInternedStruct(
    {
      spec: { containers: [{ name: 'app', image: 'nginx:latest' }] },
      status: { phase: 'Running' },
    },
    internPool,
    idState,
  );

  // Generate static and dynamic timelines
  const timelines: TimelineDTO[] = [
    {
      id: 1,
      timelineTypeId: 1, // APIVersion
      nameStringId: apiVersionStringId,
      parentTimelineId: 0,
      revisionIds: [],
      eventIds: [],
    },
    {
      id: 2,
      timelineTypeId: 2, // Kind
      nameStringId: kindStringId,
      parentTimelineId: 1,
      revisionIds: [],
      eventIds: [],
    },
    {
      id: 3,
      timelineTypeId: 3, // Namespace
      nameStringId: namespaceStringId,
      parentTimelineId: 2,
      revisionIds: [],
      eventIds: [],
    },
    {
      id: 4,
      timelineTypeId: 4, // Resource
      nameStringId: podNameStringId,
      parentTimelineId: 3,
      revisionIds: [1],
      eventIds: [1],
    },
    {
      id: 5,
      timelineTypeId: 5, // SubResource
      nameStringId: subresourceStringId,
      parentTimelineId: 4,
      revisionIds: [],
      eventIds: [],
    },
    {
      id: 6,
      timelineTypeId: 4, // Resource
      nameStringId: podNameMockPod2StringId,
      parentTimelineId: 3,
      revisionIds: [],
      eventIds: [],
    },
    {
      id: 7,
      timelineTypeId: 1, // APIVersion
      nameStringId: apiVersionAppsV1StringId,
      parentTimelineId: 0,
      revisionIds: [],
      eventIds: [],
    },
    {
      id: 8,
      timelineTypeId: 2, // Kind
      nameStringId: kindDeploymentStringId,
      parentTimelineId: 7,
      revisionIds: [],
      eventIds: [],
    },
    {
      id: 9,
      timelineTypeId: 3, // Namespace
      nameStringId: namespaceKubeSystemStringId,
      parentTimelineId: 8,
      revisionIds: [],
      eventIds: [],
    },
    {
      id: 10,
      timelineTypeId: 4, // Resource
      nameStringId: deploymentNameCorednsStringId,
      parentTimelineId: 9,
      revisionIds: [],
      eventIds: [],
    },
    {
      id: 11,
      timelineTypeId: 5, // SubResource
      nameStringId: subresourceStatusStringId,
      parentTimelineId: 10,
      revisionIds: [],
      eventIds: [],
    },
  ];

  let nextTimelineId = 12;

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
          timelines.push({
            id: resId,
            timelineTypeId: 4, // Resource
            nameStringId: resStrings[r],
            parentTimelineId: nsId,
            revisionIds: [],
            eventIds: [],
          });

          for (let s = 0; s < 10; s++) {
            const subId = nextTimelineId++;
            timelines.push({
              id: subId,
              timelineTypeId: 5, // Subresource
              nameStringId: subStrings[s],
              parentTimelineId: resId,
              revisionIds: [],
              eventIds: [],
            });
          }
        }
      }
    }
  }

  timelineStore.initialize(
    timelines,
    timelines.length,
    [
      {
        id: 1,
        logId: 1,
        changedTime: timestamp - 5n * 1000_000_000n,
        principalStringId,
        verbTypeId: 1,
        stateTypeId: 1, // Existing
        body: revisionBody,
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
