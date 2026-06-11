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

type LogType = number;
type RevisionState = number;
type RevisionVerb = number;
type Severity = number;
import { InternPoolStore } from 'src/app/store/domain/intern-pool-store';
import { StyleStore } from 'src/app/store/domain/style-store';
import { LogStore, LogDTO } from 'src/app/store/domain/log-store';
import {
  TimelineStore,
  TimelineDTO,
  RevisionDTO,
  EventDTO,
} from 'src/app/store/domain/timeline-store';
import { Log } from 'src/app/store/domain/log';
import { TimelineChartViewModel } from '../timeline-chart.viewmodel';
import {
  RulerViewModelBuilder,
  TimelineRulerViewModel,
} from '../timeline-ruler.viewmodel';
import { HistogramCache } from './histogram-cache';
import { getMinTimeSpanForHistogram } from '../calculator/human-friendly-tick';

/**
 * DemoViewModelBuilder is a utility class for constructing `TimelineChartViewModel` and `TimelineRulerViewModel`
 * specifically for testing and Storybook demonstrations using the V2 domain stores.
 */
export class DemoViewModelBuilder {
  private readonly internPool = InternPoolStore.create();
  private readonly styleStore = new StyleStore();
  private readonly logStore = LogStore.create(this.internPool, this.styleStore);
  private readonly timelineStore = TimelineStore.create(
    this.internPool,
    this.styleStore,
    this.logStore,
  );

  private logIndex = 0;
  private nextStringId = 1;
  private nextTimelineId = 1;
  private nextRevisionId = 1;
  private nextEventId = 1;

  private readonly logDTOs: LogDTO[] = [];
  private readonly timelineDTOs: TimelineDTO[] = [];
  private readonly revisionDTOs: RevisionDTO[] = [];
  private readonly eventDTOs: EventDTO[] = [];

  private readonly stringsMap = new Map<string, number>();
  private readonly timelineTypeIds = new Map<string, number>();

  private isInitialized = false;

  /**
   * Initializes a new instance of DemoViewModelBuilder.
   * Registers standard mock severity, log, and timeline style configurations.
   *
   * @param startTime The start timestamp of the timeline in milliseconds.
   * @param endTime The end timestamp of the timeline in milliseconds.
   */
  constructor(
    private readonly startTime: number,
    private readonly endTime: number,
  ) {
    // Setup standard severities
    this.styleStore.addSeverities([
      {
        id: 0, // SeverityUnknown
        label: 'Unknown',
        shortLabel: 'U',
        backgroundColor: { r: 0.502, g: 0.502, b: 0.502, a: 1 },
        foregroundColor: { r: 1, g: 1, b: 1, a: 1 },
        order: 0,
      },
      {
        id: 1, // SeverityInfo
        label: 'Info',
        shortLabel: 'I',
        backgroundColor: { r: 0, g: 0, b: 1, a: 1 },
        foregroundColor: { r: 1, g: 1, b: 1, a: 1 },
        order: 1,
      },
      {
        id: 2, // SeverityWarning
        label: 'Warning',
        shortLabel: 'W',
        backgroundColor: { r: 1, g: 0.667, b: 0.267, a: 1 },
        foregroundColor: { r: 1, g: 1, b: 1, a: 1 },
        order: 2,
      },
      {
        id: 3, // SeverityError
        label: 'Error',
        shortLabel: 'E',
        backgroundColor: { r: 1, g: 0.224, b: 0.208, a: 1 },
        foregroundColor: { r: 1, g: 1, b: 1, a: 1 },
        order: 3,
      },
      {
        id: 4, // SeverityFatal
        label: 'Fatal',
        shortLabel: 'F',
        backgroundColor: { r: 0.667, g: 0.4, b: 0.667, a: 1 },
        foregroundColor: { r: 1, g: 1, b: 1, a: 1 },
        order: 4,
      },
    ]);

    // Setup standard log types
    this.styleStore.addLogTypes([
      {
        id: 0, // LogTypeUnknown
        label: 'Unknown',
        description: 'Unknown log type',
        backgroundColor: { r: 0.502, g: 0.502, b: 0.502, a: 1 },
        foregroundColor: { r: 1, g: 1, b: 1, a: 1 },
      },
      {
        id: 1, // LogTypeAudit
        label: 'Audit',
        description: 'Audit log entry',
        backgroundColor: { r: 0, g: 0.502, b: 0, a: 1 },
        foregroundColor: { r: 1, g: 1, b: 1, a: 1 },
      },
      {
        id: 2, // LogTypeEvent
        label: 'Event',
        description: 'Kubernetes Event',
        backgroundColor: { r: 1, g: 0.647, b: 0, a: 1 },
        foregroundColor: { r: 1, g: 1, b: 1, a: 1 },
      },
    ]);
  }

  private getStringId(val: string): number {
    let id = this.stringsMap.get(val);
    if (id === undefined) {
      id = this.nextStringId++;
      this.stringsMap.set(val, id);
      this.internPool.addStrings([{ id, value: val }]);
    }
    return id;
  }

  private getTimelineTypeId(resourcePath: string): number {
    const lastPart = resourcePath.split('#').pop() || resourcePath;
    let id = this.timelineTypeIds.get(lastPart);
    if (id === undefined) {
      id = this.timelineTypeIds.size + 1;
      this.timelineTypeIds.set(lastPart, id);
      this.styleStore.addTimelineTypes([
        {
          id,
          label: lastPart,
          description: `Timeline type for ${lastPart}`,
          icon: 'list',
          backgroundColor: { r: 33, g: 150, b: 243, a: 255 },
          foregroundColor: { r: 255, g: 255, b: 255, a: 255 },
          typeChipBackgroundColor: { r: 33, g: 150, b: 243, a: 255 },
          typeChipForegroundColor: { r: 255, g: 255, b: 255, a: 255 },
          visible: true,
          sortPriority: id,
          height: 20,
        },
      ]);
    }
    return id;
  }

  private ensureInitialized() {
    if (this.isInitialized) {
      return;
    }
    this.logDTOs.sort((a, b) => Number(a.ts - b.ts));
    this.logStore.initialize(this.logDTOs, this.logDTOs.length);

    this.timelineStore.initialize(
      this.timelineDTOs,
      this.timelineDTOs.length,
      this.revisionDTOs,
      this.revisionDTOs.length,
      this.eventDTOs,
      this.eventDTOs.length,
    );
    this.isInitialized = true;
  }

  /**
   * Creates a mock revision, registers a corresponding log, and returns the generated revision reference.
   */
  createRevision(
    startTime: number,
    endTime: number,
    revisionState: RevisionState,
    verb: RevisionVerb,
    logTime: number = NaN,
  ) {
    if (Number.isNaN(logTime)) {
      logTime = startTime;
    }
    const logId = this.logIndex++;

    this.logDTOs.push({
      id: logId,
      ts: BigInt(Math.floor(logTime)) * 1000000n,
      logTypeId: 1, // LogTypeAudit
      severityTypeId: 1, // SeverityInfo
      summaryStringId: this.getStringId(''),
      body: undefined,
    });

    const revisionId = this.nextRevisionId++;
    const revisionDto: RevisionDTO = {
      id: revisionId,
      logId,
      changedTime: BigInt(Math.floor(startTime)) * 1000000n,
      principalStringId: this.getStringId(''),
      verbTypeId: verb,
      stateTypeId: revisionState,
      body: undefined,
    };
    this.revisionDTOs.push(revisionDto);
    return { type: 'revision' as const, id: revisionId };
  }

  /**
   * Creates a mock timeline and associates it with any specified child revisions or events.
   */
  createTimeline(
    resourcePath: string,
    ...items: (
      | { type: 'revision'; id: number }
      | { type: 'event'; id: number }
    )[]
  ) {
    const revisionIds: number[] = [];
    const eventIds: number[] = [];
    for (const item of items) {
      if (item.type === 'revision') {
        revisionIds.push(item.id);
      } else {
        eventIds.push(item.id);
      }
    }

    const parts = resourcePath.split('#');
    const pathWithoutUniqueId = parts[0];
    const parentPathParts = pathWithoutUniqueId.split('/');
    parentPathParts.pop();
    const parentPath = parentPathParts.join('/');

    let parentTimelineId = 0;
    if (parentPath) {
      const parent = this.timelineDTOs.find((t) => {
        const tName = this.internPool.getString(t.nameStringId);
        return tName === parentPath || tName.startsWith(parentPath + '#');
      });
      if (parent) {
        parentTimelineId = parent.id;
      }
    }

    const timelineId = this.nextTimelineId++;
    const timelineDto: TimelineDTO = {
      id: timelineId,
      timelineTypeId: this.getTimelineTypeId(resourcePath),
      nameStringId: this.getStringId(resourcePath),
      parentTimelineId,
      revisionIds,
      eventIds,
    };
    this.timelineDTOs.push(timelineDto);
  }

  /**
   * Creates a mock event, registers a corresponding log, and returns the event reference.
   */
  createEvent(startTime: number, logType: LogType, logSeverity: Severity) {
    const logId = this.logIndex++;
    this.logDTOs.push({
      id: logId,
      ts: BigInt(Math.floor(startTime)) * 1000000n,
      logTypeId: logType,
      severityTypeId: logSeverity,
      summaryStringId: this.getStringId(''),
      body: undefined,
    });

    const eventId = this.nextEventId++;
    const eventDto: EventDTO = {
      id: eventId,
      logId,
    };
    this.eventDTOs.push(eventDto);
    return { type: 'event' as const, id: eventId };
  }

  /**
   * Generates a `TimelineChartViewModel` based on the accumulated timelines.
   */
  getChartViewModel(): TimelineChartViewModel {
    this.ensureInitialized();
    return {
      timelinesInDrawArea: Array.from(this.timelineStore.timelines),
      logBeginTime: this.startTime,
      logEndTime: this.endTime,
      inspectionDataUniqueID: 'demo',
      styleStore: this.styleStore,
    };
  }

  /**
   * Generates a `TimelineRulerViewModel` based on the accumulated logs and viewport width.
   */
  getRulerViewModel(viewportWidth: number): TimelineRulerViewModel {
    this.ensureInitialized();
    const rulerViewModelBuilder = new RulerViewModelBuilder();
    const logsList = Array.from(this.logStore.logs()) as Log[];

    const allLogsCache = new HistogramCache(
      this.styleStore.severities,
      logsList,
      getMinTimeSpanForHistogram(10000, this.startTime, this.endTime),
      this.startTime,
      this.endTime,
    );
    const filteredLogsCache = new HistogramCache(
      this.styleStore.severities,
      logsList,
      getMinTimeSpanForHistogram(10000, this.startTime, this.endTime),
      this.startTime,
      this.endTime,
    );
    return rulerViewModelBuilder.generateRulerViewModel(
      this.startTime,
      viewportWidth / (this.endTime - this.startTime),
      viewportWidth,
      0,
      allLogsCache,
      filteredLogsCache,
    );
  }

  /**
   * Returns a Set of all log indices generated by this builder.
   */
  getAllActiveLogIndices(): Set<number> {
    this.ensureInitialized();
    const result = new Set<number>();
    for (const log of this.logStore.logs()) {
      result.add(log.logIndex);
    }
    return result;
  }
}
