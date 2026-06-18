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

/**
 * Represents a high dynamic range color.
 */
export interface HDRColor4 {
  readonly r: number;
  readonly g: number;
  readonly b: number;
  readonly a: number;
}

/**
 * Severity level of a log entry.
 */
export interface Severity {
  readonly id: number;
  readonly label: string;
  readonly shortLabel: string;
  readonly backgroundColor: HDRColor4;
  readonly foregroundColor: HDRColor4;
  readonly order: number;
}

/**
 * Action taken in an audit log or an event.
 */
export interface Verb {
  readonly id: number;
  readonly label: string;
  readonly backgroundColor: HDRColor4;
  readonly foregroundColor: HDRColor4;
  readonly visible: boolean;
}

/**
 * Category or source of a log entry.
 */
export interface LogType {
  readonly id: number;
  readonly label: string;
  readonly description: string;
  readonly backgroundColor: HDRColor4;
  readonly foregroundColor: HDRColor4;
}

/**
 * Internal visual presentation modes for revision state.
 */
export enum RevisionStateStyle {
  NORMAL = 0,
  DELETED = 1,
  PARTIAL_INFO = 2,
}

/**
 * Specific status of a resource at a given point in time.
 */
export interface RevisionState {
  readonly id: number;
  readonly label: string;
  readonly icon: string;
  readonly description: string;
  readonly backgroundColor: HDRColor4;
  readonly style: RevisionStateStyle;
}

/**
 * Presentation style for a specific type of timeline line.
 */
export interface TimelineType {
  readonly id: number;
  readonly label: string;
  readonly description: string;
  readonly icon: string;
  readonly backgroundColor: HDRColor4;
  readonly foregroundColor: HDRColor4;
  readonly typeChipBackgroundColor: HDRColor4;
  readonly typeChipForegroundColor: HDRColor4;
  readonly visible: boolean;
  readonly sortPriority: number;
  readonly height: number;
}

/**
 * Represents common properties in the BMFont configuration.
 */
export interface BMFontCommon {
  readonly lineHeight: number;
  readonly base: number;
  readonly scaleW: number;
  readonly scaleH: number;
  readonly pages: number;
  readonly packed: number;
  readonly alphaChnl: number;
  readonly redChnl: number;
  readonly greenChnl: number;
  readonly blueChnl: number;
}

/**
 * BMFontConfig is the JSON representation of a BMFont config.
 * See https://github.com/Experience-Monks/load-bmfont/blob/master/json-spec.md
 */
export interface BMFontConfig {
  readonly pages: readonly string[];
  readonly chars: readonly BMFontChar[];
  readonly common: BMFontCommon;
}

/**
 * Represents a single character in the BMFont configuration.
 */
export interface BMFontChar {
  readonly id: number;
  readonly index: number;
  readonly char: string;
  readonly width: number;
  readonly height: number;
  readonly xoffset: number;
  readonly yoffset: number;
  readonly xadvance: number;
  readonly chnl: number;
  readonly x: number;
  readonly y: number;
  readonly page: number;
}

/**
 * MSDF icon atlas.
 */
export interface IconAtlas {
  readonly msdfIconImage: TexImageSource[];
  readonly bmfontJson: BMFontConfig;
  readonly nameToCodepoints: Map<string, string>;
}

/**
 * Interface representing a provider of style configurations (severities, log types, etc.).
 * Structurally matches StyleStore and StyleOverrideService, but has no DOM/Angular dependencies.
 */
export interface StyleProvider {
  /** The collection of severity level configurations. */
  readonly severities: ReadonlyDomainElement<Severity[]>;
  /** The collection of log category/source configurations. */
  readonly logTypes: ReadonlyDomainElement<LogType[]>;
  /** The collection of action verb configurations. */
  readonly verbs: ReadonlyDomainElement<Verb[]>;
  /** The collection of revision status configurations. */
  readonly revisionStates: ReadonlyDomainElement<RevisionState[]>;
  /** The collection of timeline presentation styles. */
  readonly timelineTypes: ReadonlyDomainElement<TimelineType[]>;

  /**
   * Retrieves a severity configuration by its ID.
   * @param id The severity level ID.
   */
  getSeverity(id: number): ReadonlyDomainElement<Severity>;

  /**
   * Retrieves a log type configuration by its ID.
   * @param id The log category ID.
   */
  getLogType(id: number): ReadonlyDomainElement<LogType>;

  /**
   * Retrieves an action verb configuration by its ID.
   * @param id The action verb ID.
   */
  getVerb(id: number): ReadonlyDomainElement<Verb>;

  /**
   * Retrieves a revision state configuration by its ID.
   * @param id The revision status ID.
   */
  getRevisionState(id: number): ReadonlyDomainElement<RevisionState>;

  /**
   * Retrieves a timeline presentation style by its ID.
   * @param id The timeline type ID.
   */
  getTimelineType(id: number): ReadonlyDomainElement<TimelineType>;
}

/**
 * Interface representing shared StyleStore data.
 */
export interface StyleStoreSharedData {
  /** List of all severity level configurations. */
  readonly severities: readonly Severity[];
  /** List of all log category/source configurations. */
  readonly logTypes: readonly LogType[];
  /** List of all action verb configurations. */
  readonly verbs: readonly Verb[];
  /** List of all revision status configurations. */
  readonly revisionStates: readonly RevisionState[];
  /** List of all timeline presentation styles. */
  readonly timelineTypes: readonly TimelineType[];
}
