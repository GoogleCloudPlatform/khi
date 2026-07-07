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
