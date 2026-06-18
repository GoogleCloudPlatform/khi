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

import { ReadonlyDomainElement } from 'src/app/store/domain/types';
import {
  Severity,
  LogType,
  Verb,
  RevisionState,
  TimelineType,
  StyleProvider,
  StyleStoreSharedData,
} from 'src/app/store/domain/style';

/**
 * Lightweight StyleStore implementation for WebWorkers.
 * Implements StyleProvider using maps populated from StyleStoreSharedData.
 */
export class WorkerStyleStore implements StyleProvider {
  /** The collection of severity level configurations. */
  public readonly severities: ReadonlyDomainElement<Severity[]>;
  /** The collection of log category/source configurations. */
  public readonly logTypes: ReadonlyDomainElement<LogType[]>;
  /** The collection of action verb configurations. */
  public readonly verbs: ReadonlyDomainElement<Verb[]>;
  /** The collection of revision status configurations. */
  public readonly revisionStates: ReadonlyDomainElement<RevisionState[]>;
  /** The collection of timeline presentation styles. */
  public readonly timelineTypes: ReadonlyDomainElement<TimelineType[]>;

  private readonly severityMap = new Map<number, Severity>();
  private readonly logTypeMap = new Map<number, LogType>();
  private readonly verbMap = new Map<number, Verb>();
  private readonly revisionStateMap = new Map<number, RevisionState>();
  private readonly timelineTypeMap = new Map<number, TimelineType>();

  /**
   * Constructs a WorkerStyleStore from serialized shared data.
   * @param sharedData The serialized style store data.
   */
  constructor(sharedData: StyleStoreSharedData) {
    this.severities = sharedData.severities;
    this.logTypes = sharedData.logTypes;
    this.verbs = sharedData.verbs;
    this.revisionStates = sharedData.revisionStates;
    this.timelineTypes = sharedData.timelineTypes;
    for (const s of sharedData.severities) {
      this.severityMap.set(s.id, s);
    }
    for (const l of sharedData.logTypes) {
      this.logTypeMap.set(l.id, l);
    }
    for (const v of sharedData.verbs) {
      this.verbMap.set(v.id, v);
    }
    for (const r of sharedData.revisionStates) {
      this.revisionStateMap.set(r.id, r);
    }
    for (const t of sharedData.timelineTypes) {
      this.timelineTypeMap.set(t.id, t);
    }
  }

  /**
   * Retrieves a severity configuration by its ID.
   * @param id The severity level ID.
   */
  public getSeverity(id: number): ReadonlyDomainElement<Severity> {
    const s = this.severityMap.get(id);
    if (!s) {
      throw new Error(`Severity not found: ${id}`);
    }
    return s as ReadonlyDomainElement<Severity>;
  }

  /**
   * Retrieves a log type configuration by its ID.
   * @param id The log category ID.
   */
  public getLogType(id: number): ReadonlyDomainElement<LogType> {
    const l = this.logTypeMap.get(id);
    if (!l) {
      throw new Error(`LogType not found: ${id}`);
    }
    return l as ReadonlyDomainElement<LogType>;
  }

  /**
   * Retrieves an action verb configuration by its ID.
   * @param id The action verb ID.
   */
  public getVerb(id: number): ReadonlyDomainElement<Verb> {
    const v = this.verbMap.get(id);
    if (!v) {
      throw new Error(`Verb not found: ${id}`);
    }
    return v as ReadonlyDomainElement<Verb>;
  }

  /**
   * Retrieves a revision state configuration by its ID.
   * @param id The revision status ID.
   */
  public getRevisionState(id: number): ReadonlyDomainElement<RevisionState> {
    const r = this.revisionStateMap.get(id);
    if (!r) {
      throw new Error(`RevisionState not found: ${id}`);
    }
    return r as ReadonlyDomainElement<RevisionState>;
  }

  /**
   * Retrieves a timeline presentation style by its ID.
   * @param id The timeline type ID.
   */
  public getTimelineType(id: number): ReadonlyDomainElement<TimelineType> {
    const t = this.timelineTypeMap.get(id);
    if (!t) {
      throw new Error(`TimelineType not found: ${id}`);
    }
    return t as ReadonlyDomainElement<TimelineType>;
  }
}
