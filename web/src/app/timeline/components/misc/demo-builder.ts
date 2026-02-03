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

import { ToTextReferenceFromKHIFileBinary } from 'src/app/common/loader/reference-type';
import {
  LogType,
  ParentRelationship,
  RevisionState,
  RevisionVerb,
  Severity,
} from 'src/app/zzz-generated';
import { ResourceEvent } from 'src/app/store/event';
import { LogEntry } from 'src/app/store/log';
import { ResourceRevision } from 'src/app/store/revision';
import { ResourceTimeline } from 'src/app/store/timeline';
import { TimelineChartViewModel } from '../timeline-chart.viewmodel';
import {
  RulerViewModelBuilder,
  TimelineRulerViewModel,
} from '../timeline-ruler.viewmodel';
import { HistogramCache } from './histogram-cache';
import { getMinTimeSpanForHistogram } from '../calculator/human-friendly-tick';

/**
 * DemoViewModelBuilder is a utility class for constructing `TimelineChartViewModel` and `TimelineRulerViewModel`
 * specifically for testing and Storybook demonstrations.
 *
 * It simplifies the creation of complex timeline data structures (Timelines, Revisions, Events, Logs)
 * and allows generating consistent view models for both the chart and the ruler.
 */
export class DemoViewModelBuilder {
  private logIndex = 0;

  timelines: ResourceTimeline[] = [];

  logs: LogEntry[] = [];

  /**
   * Initializes a new instance of DemoViewModelBuilder.
   *
   * @param startTime The start timestamp of the timeline in milliseconds.
   * @param endTime The end timestamp of the timeline in milliseconds.
   */
  constructor(
    private readonly startTime: number,
    private readonly endTime: number,
  ) {}

  /**
   * Creates a `ResourceRevision` with the specified properties and registers a corresponding start log.
   *
   * @param startTime The start timestamp of the revision (ms).
   * @param endTime The end timestamp of the revision (ms).
   * @param revisionState The state of the revision (e.g., Active, Deleted).
   * @param verb The verb describing the revision change (e.g., Created, Updated).
   * @returns A new `ResourceRevision` instance.
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
    const logIndex = this.logIndex++;
    this.logs.push(
      new LogEntry(
        logIndex,
        '',
        LogType.LogTypeAudit,
        Severity.SeverityInfo,
        logTime,
        '',
        ToTextReferenceFromKHIFileBinary(),
        [],
      ),
    );
    return new ResourceRevision(
      startTime,
      endTime,
      revisionState,
      verb,
      '',
      '',
      false,
      false,
      logIndex,
    );
  }

  /**
   * Creates a `ResourceTimeline` containing the provided revisions and events.
   *
   * @param resourcePath The name or path of the resource for this timeline.
   * @param relationship The relationship of this timeline to its parent (default is Child).
   * @param items A variable number of `ResourceRevision` or `ResourceEvent` items to include in the timeline.
   */
  createTimeline(
    resourcePath: string,
    relationship: ParentRelationship = ParentRelationship.RelationshipChild,
    ...items: (ResourceRevision | ResourceEvent)[]
  ) {
    const revisions: ResourceRevision[] = [];
    const events: ResourceEvent[] = [];
    for (const item of items) {
      if (item instanceof ResourceRevision) {
        revisions.push(item);
      } else {
        events.push(item);
      }
    }
    this.timelines.push(
      new ResourceTimeline(
        `${resourcePath}#${this.timelines.length}`,
        resourcePath,
        revisions,
        events,
        relationship,
      ),
    );
  }

  /**
   * Creates a `ResourceEvent` with the specified properties and registers a corresponding log.
   *
   * @param startTime The timestamp of the event (ms).
   * @param logType The type of the log (e.g., Audit, K8sEvent).
   * @param logSeverity The severity of the log (e.g., Info, Error).
   * @returns A new `ResourceEvent` instance.
   */
  createEvent(startTime: number, logType: LogType, logSeverity: Severity) {
    const logIndex = this.logIndex++;
    this.logs.push(
      new LogEntry(
        logIndex,
        '',
        logType,
        logSeverity,
        startTime,
        '',
        ToTextReferenceFromKHIFileBinary(),
        [],
      ),
    );
    return new ResourceEvent(logIndex, startTime, logType, logSeverity);
  }

  /**
   * Generates a `TimelineChartViewModel` based on the accumulated timelines.
   *
   * @returns The view model for the timeline chart.
   */
  getChartViewModel(): TimelineChartViewModel {
    return {
      timelinesInDrawArea: this.timelines,
      logBeginTime: this.startTime,
      logEndTime: this.endTime,
    };
  }

  /**
   * Generates a `TimelineRulerViewModel` based on the accumulated logs and the provided viewport width.
   *
   * It calculates the histogram and ruler marks appropriate for the current zoom level (implied by viewport width).
   *
   * @param viewportWidth The width of the viewport in pixels.
   * @returns The view model for the timeline ruler.
   */
  getRulerViewModel(viewportWidth: number): TimelineRulerViewModel {
    const rulerViewModelBuilder = new RulerViewModelBuilder();
    const allLogsCache = new HistogramCache(
      this.logs,
      getMinTimeSpanForHistogram(10000, this.startTime, this.endTime),
      this.startTime,
      this.endTime,
    );
    const filteredLogsCache = new HistogramCache(
      this.logs,
      getMinTimeSpanForHistogram(10000, this.startTime, this.endTime),
      this.startTime,
      this.endTime,
    );
    return rulerViewModelBuilder.generateRulerViewModel(
      this.startTime,
      viewportWidth / (this.endTime - this.startTime), // pixelsPerMs
      viewportWidth, // viewportWidth
      0, // timezoneShiftHours
      allLogsCache,
      filteredLogsCache,
    );
  }

  /**
   * Returns a Set of all log indices that have been generated by this builder.
   *
   * @returns A Set containing all unique log indices.
   */
  getAllActiveLogIndices(): Set<number> {
    const result = new Set<number>();
    for (const log of this.logs) {
      result.add(log.logIndex);
    }
    return result;
  }
}
