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

import { InspectionDataBuilder } from 'src/app/parser/core/builder';
import { LogDTO } from 'src/app/store/domain/log-store';
import {
  TimelineDTO,
  RevisionDTO,
  EventDTO,
} from 'src/app/store/domain/timeline-store';
import {
  Severity,
  LogType,
  Verb,
  RevisionState,
  TimelineType,
  RevisionStateStyle,
} from 'src/app/store/domain/style';

describe('InspectionDataBuilder (Core)', () => {
  let builder: InspectionDataBuilder;

  const mockColor = { r: 0.1, g: 0.2, b: 0.3, a: 1.0 };

  beforeEach(() => {
    builder = new InspectionDataBuilder();
  });

  it('should accumulate stores and construct domain instances correctly', async () => {
    const rawLogs: LogDTO[] = [
      {
        id: 100,
        ts: 1234n,
        logTypeId: 1,
        severityTypeId: 1,
        summaryStringId: 1,
      },
    ];

    const rawTimelines: TimelineDTO[] = [
      {
        id: 10,
        timelineTypeId: 1,
        nameStringId: 1,
        parentTimelineId: 0,
        revisionIds: [100],
        eventIds: [200],
      },
    ];

    const rawRevisions: RevisionDTO[] = [
      {
        id: 100,
        logId: 100,
        changedTime: 1234n,
        principalStringId: 1,
        verbTypeId: 1,
        stateTypeId: 1,
      },
    ];

    const rawEvents: EventDTO[] = [
      {
        id: 200,
        logId: 100,
      },
    ];

    const rawSeverity: Severity = {
      id: 1,
      label: 'INFO',
      shortLabel: 'I',
      backgroundColor: mockColor,
      foregroundColor: mockColor,
      order: 0,
    };

    const rawLogType: LogType = {
      id: 1,
      label: 'type-label',
      description: '',
      backgroundColor: mockColor,
      foregroundColor: mockColor,
    };

    const rawVerb: Verb = {
      id: 1,
      label: 'VERB',
      backgroundColor: mockColor,
      foregroundColor: mockColor,
      visible: true,
    };

    const rawRevisionState: RevisionState = {
      id: 1,
      label: 'normal',
      icon: '',
      description: '',
      backgroundColor: mockColor,
      style: RevisionStateStyle.NORMAL,
    };

    const rawTimelineType: TimelineType = {
      id: 1,
      label: 'tl-type',
      description: '',
      icon: '',
      backgroundColor: mockColor,
      foregroundColor: mockColor,
      typeChipBackgroundColor: mockColor,
      typeChipForegroundColor: mockColor,
      visible: true,
      sortPriority: 0,
      height: 1,
    };

    const result = await builder
      .addStrings([{ id: 1, value: 'summary_value' }])
      .addFieldPathSets([{ id: 10, fieldPathStringIds: [1] }])
      .addLogs(rawLogs)
      .addTimelines(rawTimelines)
      .addRevisions(rawRevisions)
      .addEvents(rawEvents)
      .addSeverities([rawSeverity])
      .addLogTypes([rawLogType])
      .addVerbs([rawVerb])
      .addRevisionStates([rawRevisionState])
      .addTimelineTypes([rawTimelineType])
      .build();

    expect(result.internPool.getString(1)).toBe('summary_value');
    expect(result.styleStore.getLogType(1).label).toBe('type-label');

    const l = result.logStore.getLog(100);
    expect(l.id).toBe(100);
    expect(l.timestamp).toBe(1234n);
  });
});
