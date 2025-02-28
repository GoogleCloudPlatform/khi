/**
 * Copyright 2024 Google LLC
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

/**
 * metadata-types.ts
 * Defines the schema of metadata generated from KHI inspection task.
 * `Metadata` in kHI inspection task is a non main data(KHIFile) generated from a task run/dryrun.
 * Each inspection task generates the set of metadata and it is used in frontend.
 * Metadata includes the form fields needed to be filled on new inspection dialogs, task progress,..etc
 */

export type InspectionMetadataPlan = {
  taskGraph: string;
};

export type InspectionMetadataQuery = {
  id: string;
  name: string;
  query: string;
};

export type InspectionMetadataErrorSet = {
  errorMessages: InspectionMetadataError[];
};

export type InspectionMetadataError = {
  errorId: string;
  message: string;
  link: string;
};

export type InspectionMetadataHeader = {
  inspectionType: string;
  inspectionTypeIconPath: string;
  inspectTimeUnixSeconds: number;
  startTimeUnixSeconds: number;
  endTimeUnixSeconds: number;
  suggestedFilename: string;
  fileSize?: number;
};

export type InspectionMetadataProgressPhase =
  | 'RUNNING'
  | 'ERROR'
  | 'CANCELLED'
  | 'DONE';

export type InspectionMetadataProgress = {
  phase: InspectionMetadataProgressPhase;
  progresses: InspectionMetadataProgressElement[];
  totalProgress: InspectionMetadataProgressElement;
};

export type InspectionMetadataProgressElement = {
  id: string;
  label: string;
  message: string;
  percentage: number;
  indeterminate: boolean;
};

export type InspectionMetadataLog = {
  id: string;
  name: string;
  log: string;
};
