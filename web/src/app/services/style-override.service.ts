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

import { inject, Injectable, computed, signal } from '@angular/core';
import { InspectionDataStoreV2 } from 'src/app/services/inspection-data-store-v2.service';
import {
  IconAtlas,
  LogType,
  RevisionState,
  Severity,
  TimelineType,
  Verb,
} from 'src/app/store/domain/style';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';
import { StyleStoreLike } from 'src/app/store/domain/style-store';

/**
 * Service to override timeline element style configurations dynamically.
 * Injects InspectionDataStoreV2 to fetch baseline StyleStore settings.
 */
@Injectable({
  providedIn: 'root',
})
export class StyleOverrideService implements StyleStoreLike {
  private readonly inspectionDataStore = inject(InspectionDataStoreV2);

  /** Map of overridden revision states. Key is the state ID. */
  private readonly _revisionStateOverrides = signal<Map<number, RevisionState>>(
    new Map(),
  );

  /** Map of overridden timeline types. Key is the timeline type ID. */
  private readonly _timelineTypeOverrides = signal<Map<number, TimelineType>>(
    new Map(),
  );

  /** Map of overridden log types. Key is the log type ID. */
  private readonly _logTypeOverrides = signal<Map<number, LogType>>(new Map());

  /** Signal triggered/incremented whenever styles are updated. */
  public readonly stylesUpdated = signal(0);

  /** Base style store from current active inspection data. */
  protected readonly baseStyleStore = computed(() => {
    return this.inspectionDataStore.inspectionData()?.styleStore ?? null;
  });

  /**
   * Overrides the style configuration of a revision state.
   * @param s The new revision state config DTO to override with.
   */
  public overrideRevisionState(s: RevisionState): void {
    this._revisionStateOverrides.update((map) => {
      const newMap = new Map(map);
      newMap.set(s.id, s);
      return newMap;
    });
    this.stylesUpdated.update((v) => v + 1);
  }

  /**
   * Resets the override for a specific revision state.
   * @param id The ID of the revision state.
   */
  public resetRevisionState(id: number): void {
    this._revisionStateOverrides.update((map) => {
      if (map.has(id)) {
        const newMap = new Map(map);
        newMap.delete(id);
        return newMap;
      }
      return map;
    });
    this.stylesUpdated.update((v) => v + 1);
  }

  /**
   * Checks if a revision state's color has been overridden.
   * @param id The ID of the revision state.
   * @returns True if overridden, false otherwise.
   */
  public isRevisionStateOverridden(id: number): boolean {
    return this._revisionStateOverrides().has(id);
  }

  /**
   * Overrides the style configuration of a timeline type.
   * @param t The new timeline type config DTO to override with.
   */
  public overrideTimelineType(t: TimelineType): void {
    this._timelineTypeOverrides.update((map) => {
      const newMap = new Map(map);
      newMap.set(t.id, t);
      return newMap;
    });
    this.stylesUpdated.update((v) => v + 1);
  }

  /**
   * Resets the override for a specific timeline type.
   * @param id The ID of the timeline type.
   */
  public resetTimelineType(id: number): void {
    this._timelineTypeOverrides.update((map) => {
      if (map.has(id)) {
        const newMap = new Map(map);
        newMap.delete(id);
        return newMap;
      }
      return map;
    });
    this.stylesUpdated.update((v) => v + 1);
  }

  /**
   * Checks if a timeline type's color has been overridden.
   * @param id The ID of the timeline type.
   * @returns True if overridden, false otherwise.
   */
  public isTimelineTypeOverridden(id: number): boolean {
    return this._timelineTypeOverrides().has(id);
  }

  /**
   * Overrides the style configuration of a log type.
   * @param l The new log type config DTO to override with.
   */
  public overrideLogType(l: LogType): void {
    this._logTypeOverrides.update((map) => {
      const newMap = new Map(map);
      newMap.set(l.id, l);
      return newMap;
    });
    this.stylesUpdated.update((v) => v + 1);
  }

  /**
   * Resets the override for a specific log type.
   * @param id The ID of the log type.
   */
  public resetLogType(id: number): void {
    this._logTypeOverrides.update((map) => {
      if (map.has(id)) {
        const newMap = new Map(map);
        newMap.delete(id);
        return newMap;
      }
      return map;
    });
    this.stylesUpdated.update((v) => v + 1);
  }

  /**
   * Checks if a log type's style has been overridden.
   * @param id The ID of the log type.
   * @returns True if overridden, false otherwise.
   */
  public isLogTypeOverridden(id: number): boolean {
    return this._logTypeOverrides().has(id);
  }

  /**
   * Returns all severities defined in the base store.
   */
  public get severities(): ReadonlyDomainElement<Severity[]> {
    return this.baseStyleStore()?.severities ?? [];
  }

  /**
   * Returns all log types, applying overrides if they exist.
   */
  public get logTypes(): ReadonlyDomainElement<LogType[]> {
    const originalTypes = this.baseStyleStore()?.logTypes ?? [];
    const overrides = this._logTypeOverrides();
    if (overrides.size === 0) {
      return originalTypes;
    }
    return originalTypes.map((type) => overrides.get(type.id) ?? type);
  }

  /**
   * Returns all verbs defined in the base store.
   */
  public get verbs(): ReadonlyDomainElement<Verb[]> {
    return this.baseStyleStore()?.verbs ?? [];
  }

  /**
   * Returns all revision states, applying overrides if they exist.
   */
  public get revisionStates(): ReadonlyDomainElement<RevisionState[]> {
    const originalStates = this.baseStyleStore()?.revisionStates ?? [];
    const overrides = this._revisionStateOverrides();
    if (overrides.size === 0) {
      return originalStates;
    }
    return originalStates.map((state) => overrides.get(state.id) ?? state);
  }

  /**
   * Returns all timeline types, applying overrides if they exist.
   */
  public get timelineTypes(): ReadonlyDomainElement<TimelineType[]> {
    const originalTypes = this.baseStyleStore()?.timelineTypes ?? [];
    const overrides = this._timelineTypeOverrides();
    if (overrides.size === 0) {
      return originalTypes;
    }
    return originalTypes.map((type) => overrides.get(type.id) ?? type);
  }

  /**
   * Resolves severity by ID from the base store.
   * @param id The ID of the severity.
   */
  public getSeverity(id: number): ReadonlyDomainElement<Severity> {
    const store = this.baseStyleStore();
    if (!store) {
      throw new Error('StyleStore is not loaded');
    }
    return store.getSeverity(id);
  }

  /**
   * Resolves log type by ID, applying override if set.
   * @param id The ID of the log type.
   */
  public getLogType(id: number): ReadonlyDomainElement<LogType> {
    const override = this._logTypeOverrides().get(id);
    if (override) {
      return override;
    }
    const store = this.baseStyleStore();
    if (!store) {
      throw new Error('StyleStore is not loaded');
    }
    return store.getLogType(id);
  }

  /**
   * Resolves verb by ID from the base store.
   * @param id The ID of the verb.
   */
  public getVerb(id: number): ReadonlyDomainElement<Verb> {
    const store = this.baseStyleStore();
    if (!store) {
      throw new Error('StyleStore is not loaded');
    }
    return store.getVerb(id);
  }

  /**
   * Resolves revision state by ID, applying override if set.
   * @param id The ID of the revision state.
   */
  public getRevisionState(id: number): ReadonlyDomainElement<RevisionState> {
    const override = this._revisionStateOverrides().get(id);
    if (override) {
      return override;
    }
    const store = this.baseStyleStore();
    if (!store) {
      throw new Error('StyleStore is not loaded');
    }
    return store.getRevisionState(id);
  }

  /**
   * Resolves timeline type by ID, applying override if set.
   * @param id The ID of the timeline type.
   */
  public getTimelineType(id: number): ReadonlyDomainElement<TimelineType> {
    const override = this._timelineTypeOverrides().get(id);
    if (override) {
      return override;
    }
    const store = this.baseStyleStore();
    if (!store) {
      throw new Error('StyleStore is not loaded');
    }
    return store.getTimelineType(id);
  }

  /**
   * Resolves loaded icon atlas.
   */
  public getIconAtlas(): IconAtlas | undefined {
    return this.baseStyleStore()?.getIconAtlas();
  }
}
