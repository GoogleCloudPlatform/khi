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

import { Type } from '@angular/core';

/**
 * Represents a visual annotation attached to a specific YAML path.
 *
 * This architecture guarantees that a dynamic component is always rendered.
 */
export interface YamlFieldAnnotation {
  /** Defines the exact location in the YAML structure (e.g., ['metadata', 'labels', 'app']). */
  readonly path: readonly (string | number)[];

  /** Specifies the component class to render dynamically in the tooltip. */
  readonly component: Type<unknown>;

  /** Provides the data to bind to the dynamically instantiated component's inputs. */
  readonly inputs?: Record<string, unknown>;
}

/**
 * Defines the contract for providing field-level annotations to the YAML viewer.
 */
export interface YamlAnnotationProvider {
  /**
   * Generates a list of annotations based on the parsed YAML structure.
   */
  getAnnotations(parsedYaml: unknown): YamlFieldAnnotation[];
}
