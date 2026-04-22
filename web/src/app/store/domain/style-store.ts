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

import {
  BMFontConfig,
  IconAtlas,
  LogType,
  RevisionState,
  Severity,
  TimelineType,
  Verb,
} from 'src/app/store/domain/style';
import { ReadonlyDomainElement } from './types';

// Style store's input and output share the same type; defining this alias for clarity.

/**
 * Severity level DTO for the style store.
 */
export type SeverityDTO = Severity;

/**
 * Log category or source DTO for the style store.
 */
export type LogTypeDTO = LogType;

/**
 * Action verb DTO for the style store.
 */
export type VerbDTO = Verb;

/**
 * Revision status DTO for the style store.
 */
export type RevisionStateDTO = RevisionState;

/**
 * Timeline presentation style DTO for the style store.
 */
export type TimelineTypeDTO = TimelineType;

/**
 * DTO representation of the MSDF icon atlas.
 * Contains raw array buffers that need to be processed in the browser.
 */
export interface IconAtlasDTO {
  /** Raw binary buffers for the MSDF atlas images. */
  readonly msdfIconImage: ArrayBufferLike[];
  /** Raw binary buffer for the BMFont configuration JSON. */
  readonly bmfontJson: ArrayBufferLike;
  /** Mapping of icon names to their Unicode codepoints. */
  readonly nameToCodepoints: Map<string, string>;
}

/**
 * Manages the style-related definitions for the UI.
 */
export class StyleStore {
  private readonly severities: Severity[] = [];
  private readonly logTypes: LogType[] = [];
  private readonly verbs: Verb[] = [];
  private readonly revisionStates: RevisionState[] = [];
  private readonly timelineTypes: TimelineType[] = [];

  private iconAtlas: IconAtlas | undefined;

  /**
   * Adds severities to the store.
   * @param items Iterable of severities.
   */
  public addSeverities(items: Iterable<SeverityDTO>): void {
    for (const item of items) {
      this.severities[item.id] = item;
    }
  }

  /**
   * Adds log types to the store.
   * @param items Iterable of log types.
   */
  public addLogTypes(items: Iterable<LogTypeDTO>): void {
    for (const item of items) {
      this.logTypes[item.id] = item;
    }
  }

  /**
   * Adds verbs to the store.
   * @param items Iterable of verbs.
   */
  public addVerbs(items: Iterable<VerbDTO>): void {
    for (const item of items) {
      this.verbs[item.id] = item;
    }
  }

  /**
   * Adds revision states to the store.
   * @param items Iterable of revision states.
   */
  public addRevisionStates(items: Iterable<RevisionStateDTO>): void {
    for (const item of items) {
      this.revisionStates[item.id] = item;
    }
  }

  /**
   * Adds timeline types to the store.
   * @param items Iterable of timeline types.
   */
  public addTimelineTypes(items: Iterable<TimelineTypeDTO>): void {
    for (const item of items) {
      this.timelineTypes[item.id] = item;
    }
  }

  /**
   * Sets the icon atlas by initializing it from a DTO.
   * Decodes MSDF icon image buffers and parses BMFont JSON data in the browser environment.
   * @param dto The DTO containing the raw icon atlas data.
   */
  public async setIconAtlas(dto: IconAtlasDTO): Promise<void> {
    const msdfIconImagePromises = dto.msdfIconImage.map(async (buffer) => {
      const blob = new Blob([buffer as ArrayBuffer], { type: 'image/png' });
      const url = URL.createObjectURL(blob);
      const image = new Image();
      image.src = url;
      try {
        await image.decode();
        return image;
      } finally {
        URL.revokeObjectURL(url);
      }
    });

    const msdfIconImage = await Promise.all(msdfIconImagePromises);

    const decoder = new TextDecoder('UTF-8');
    const bmfontJsonText = decoder.decode(dto.bmfontJson);
    const bmfontJson = JSON.parse(bmfontJsonText) as BMFontConfig;

    this.iconAtlas = {
      msdfIconImage,
      bmfontJson,
      nameToCodepoints: dto.nameToCodepoints,
    };
  }

  /**
   * Gets a severity by ID.
   * @param id The ID of the severity.
   * @returns The resolved severity.
   */
  public getSeverity(id: number): ReadonlyDomainElement<SeverityDTO> {
    const item = this.severities[id];
    if (!item) {
      throw new Error(`Severity ID ${id} not found`);
    }
    return item;
  }

  /**
   * Gets a log type by ID.
   * @param id The ID of the log type.
   * @returns The resolved log type.
   */
  public getLogType(id: number): ReadonlyDomainElement<LogTypeDTO> {
    const item = this.logTypes[id];
    if (!item) {
      throw new Error(`LogType ID ${id} not found`);
    }
    return item;
  }

  /**
   * Gets a verb by ID.
   * @param id The ID of the verb.
   * @returns The resolved verb.
   */
  public getVerb(id: number): ReadonlyDomainElement<VerbDTO> {
    const item = this.verbs[id];
    if (!item) {
      throw new Error(`Verb ID ${id} not found`);
    }
    return item;
  }

  /**
   * Gets a revision state by ID.
   * @param id The ID of the revision state.
   * @returns The resolved revision state.
   */
  public getRevisionState(id: number): ReadonlyDomainElement<RevisionStateDTO> {
    const item = this.revisionStates[id];
    if (!item) {
      throw new Error(`RevisionState ID ${id} not found`);
    }
    return item;
  }

  /**
   * Gets a timeline type by ID.
   * @param id The ID of the timeline type.
   * @returns The resolved timeline type.
   */
  public getTimelineType(id: number): ReadonlyDomainElement<TimelineTypeDTO> {
    const item = this.timelineTypes[id];
    if (!item) {
      throw new Error(`TimelineType ID ${id} not found`);
    }
    return item;
  }

  /**
   * Gets the loaded icon atlas.
   * @returns The parsed and initialized icon atlas.
   */
  public getIconAtlas(): IconAtlas {
    if (!this.iconAtlas) {
      throw new Error('IconAtlas is not yet loaded');
    }
    return this.iconAtlas;
  }
}
