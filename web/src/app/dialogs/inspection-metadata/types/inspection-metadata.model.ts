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

import { InspectionMetadataOfRunResult } from 'src/app/common/schema/api-types';
import { formatBytes } from 'src/app/utils/byte-format-util';
import { formatIsoTimestampSeconds } from 'src/app/utils/time-format-util';

/**
 * View model representing the overview section of an inspection.
 */
export interface MetadataOverviewViewModel {
  /** The type identifier of the inspection (e.g., 'gcp-gke'). */
  readonly inspectionType: string;
  /** The display name of the inspection task. */
  readonly inspectionName: string;
  /** Optional icon path representing the inspection type. */
  readonly inspectionTypeIconPath: string;
  /** Formatted start time string. */
  readonly formattedStartTime: string;
  /** Formatted end time string. */
  readonly formattedEndTime: string;
  /** Formatted duration string representing the time window. */
  readonly durationText: string;
  /** The suggested filename for exporting the inspection data. */
  readonly suggestedFilename: string;
  /** Formatted human-readable file size string. */
  readonly fileSizeText: string;
}

/**
 * View model representing a query executed during the inspection.
 */
export interface MetadataQueryViewModel {
  /** Unique identifier of the query. */
  readonly id: string;
  /** Human readable name of the query. */
  readonly name: string;
  /** The query string. */
  readonly query: string;
}

/**
 * View model representing a single task log.
 */
export interface MetadataLogViewModel {
  /** Unique identifier of the log / task. */
  readonly id: string;
  /** Name of the task that generated the log. */
  readonly name: string;
  /** Content of the log. */
  readonly log: string;
}

/**
 * View model representing a single error encountered during the inspection.
 */
export interface MetadataErrorViewModel {
  /** Unique error identifier or code. */
  readonly errorId: string;
  /** Error message details. */
  readonly message: string;
  /** Optional reference link for troubleshooting or documentation. */
  readonly link: string;
}

/**
 * View model representing the inspection task execution plan.
 */
export interface MetadataPlanViewModel {
  /** Task graph representation (e.g., Graphviz or text graph). */
  readonly taskGraph: string;
}

/**
 * Aggregated view model for the entire inspection metadata dialog.
 */
export interface InspectionMetadataViewModel {
  /** Overview information. */
  readonly overview: MetadataOverviewViewModel;
  /** List of executed queries. */
  readonly queries: readonly MetadataQueryViewModel[];
  /** List of task logs. */
  readonly logs: readonly MetadataLogViewModel[];
  /** Task plan graph. */
  readonly plan: MetadataPlanViewModel;
  /** List of errors. */
  readonly errors: readonly MetadataErrorViewModel[];
  /** CLI command representation for headless job execution. */
  readonly jobCommand?: string;
}

/**
 * Formats a duration in seconds into a human readable string.
 * @param durationSeconds The duration in seconds.
 * @returns A formatted string such as '1h 20m 30s' or '45s'.
 */
function formatDuration(durationSeconds: number): string {
  if (durationSeconds <= 0 || !Number.isFinite(durationSeconds)) {
    return '0s';
  }
  const totalSec = Math.floor(durationSeconds);
  const hours = Math.floor(totalSec / 3600);
  const minutes = Math.floor((totalSec % 3600) / 60);
  const seconds = totalSec % 60;

  const parts: string[] = [];
  if (hours > 0) {
    parts.push(`${hours}h`);
  }
  if (minutes > 0) {
    parts.push(`${minutes}m`);
  }
  if (seconds > 0 || parts.length === 0) {
    parts.push(`${seconds}s`);
  }
  return parts.join(' ');
}

/**
 * Converts raw InspectionMetadataOfRunResult into an InspectionMetadataViewModel.
 * @param metadata The raw metadata response from the backend.
 * @param timezoneShiftHours The timezone offset from UTC in hours.
 * @returns The converted view model ready for UI consumption.
 */
export function convertToInspectionMetadataViewModel(
  metadata: InspectionMetadataOfRunResult,
  timezoneShiftHours: number,
): InspectionMetadataViewModel {
  const header = metadata.header;
  const startSec = header.startTimeUnixSeconds;
  const endSec = header.endTimeUnixSeconds;
  const durationSec = startSec > 0 && endSec > startSec ? endSec - startSec : 0;

  const overview: MetadataOverviewViewModel = {
    inspectionType: header.inspectionType || 'Unknown',
    inspectionName: header.inspectionName || 'Untitled Inspection',
    inspectionTypeIconPath: header.inspectionTypeIconPath,
    formattedStartTime: formatIsoTimestampSeconds(startSec, timezoneShiftHours),
    formattedEndTime: formatIsoTimestampSeconds(endSec, timezoneShiftHours),
    durationText: formatDuration(durationSec),
    suggestedFilename: header.suggestedFilename || 'inspection.khi',
    fileSizeText: formatBytes(header.fileSize ?? 0),
  };

  const queries: MetadataQueryViewModel[] = metadata.query.map((q) => ({
    id: q.id,
    name: q.name,
    query: q.query,
  }));

  const logs: MetadataLogViewModel[] = metadata.log.map((l) => ({
    id: l.id,
    name: l.name,
    log: l.log,
  }));

  const plan: MetadataPlanViewModel = {
    taskGraph: metadata.plan.taskGraph,
  };

  const errors: MetadataErrorViewModel[] = metadata.error.errorMessages.map(
    (e) => ({
      errorId: e.errorId,
      message: e.message,
      link: e.link,
    }),
  );

  return {
    overview,
    queries,
    logs,
    plan,
    errors,
    jobCommand: metadata.jobCommand?.command,
  };
}
