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

/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
export function isKHIWorkerPacket(packet: any): packet is KHIWorkerPacket {
  return 'isKHIWorkerPacket' in packet && packet['isKHIWorkerPacket'];
}

export interface KHIWorkerPacket {
  isKHIWorkerPacket?: boolean;
  taskId?: string;
}

export interface FilterQuery extends KHIWorkerPacket {
  regexInStr: string;
  logs: FilterWorkerLog[];
}

/**
 * A log entry passed to the worker.
 */
export interface FilterWorkerLog {
  index: number;
  logBody: string;
  logSummary: string;
}

export interface FilterResult extends KHIWorkerPacket {
  taskId: string;
  notMatch: number[];
}
