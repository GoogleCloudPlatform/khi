import {
  YamlAnnotationProvider,
  YamlFieldAnnotation,
} from 'src/app/shared/components/yaml-viewer/yaml-annotation';
import { ManagedFieldTooltipComponent } from 'src/app/shared/components/yaml-viewer/components/managed-field-tooltip.component';

/**
 * Represents the structure of an entry in Kubernetes metadata.managedFields.
 */
interface ManagedFieldEntry {
  readonly manager: string;
  readonly time: bigint;
  readonly operation: string;
  readonly fieldsV1?: Record<string, unknown>;
}

/**
 * Provides tooltips indicating when and by whom each field was last edited.
 *
 * It parses the Kubernetes metadata.managedFields structure to map fields to their managers.
 */
export class ManagedFieldsAnnotationProvider implements YamlAnnotationProvider {
  /**
   * @param timezoneShift The timezone shift in hours from UTC to pass to tooltips.
   * @param overrideManagedFields Optional explicit managedFields to use when parsedYaml does not contain them.
   */
  constructor(
    private readonly timezoneShift: number,
    private readonly overrideManagedFields?: ManagedFieldEntry[],
  ) {}

  /**
   * Generates annotations by extracting paths from the fieldsV1 structure.
   */
  getAnnotations(parsedYaml: unknown): YamlFieldAnnotation[] {
    const annotations: YamlFieldAnnotation[] = [];

    if (typeof parsedYaml !== 'object' || parsedYaml === null) {
      return annotations;
    }

    const yamlRecord = parsedYaml as Record<string, unknown>;
    const metadata = yamlRecord['metadata'] as
      | Record<string, unknown>
      | undefined;

    const managedFields =
      this.overrideManagedFields ??
      (metadata?.['managedFields'] as ManagedFieldEntry[] | undefined);

    if (!Array.isArray(managedFields)) {
      return annotations;
    }

    for (const entry of managedFields) {
      if (entry.fieldsV1) {
        // Convert the time field from string/Date to bigint (nanoseconds since epoch).
        let timeNs = 0n;
        const timeVal = entry.time as unknown;
        if (timeVal instanceof Date) {
          timeNs = BigInt(timeVal.getTime()) * 1000000n;
        } else if (typeof timeVal === 'string' || typeof timeVal === 'number') {
          timeNs = BigInt(new Date(timeVal).getTime()) * 1000000n;
        } else if (typeof timeVal === 'bigint') {
          timeNs = timeVal;
        }

        this.extractPaths(
          entry.fieldsV1 as Record<string, unknown>,
          parsedYaml,
          [],
          entry.manager,
          timeNs,
          annotations,
        );
      }
    }

    return annotations;
  }

  /**
   * Traverses the fieldsV1 structure recursively to build JSON paths.
   *
   * Kubernetes uses prefixes like 'f:' to denote fields. This method strips those
   * prefixes to generate standard paths that match the actual YAML document.
   * For 'k:' (map list keys) and 'v:' (set values), it resolves the matching index
   * against the actual YAML document.
   */
  private extractPaths(
    current: Record<string, unknown>,
    currentYamlData: unknown,
    currentPath: readonly (string | number)[],
    manager: string,
    time: bigint,
    annotations: YamlFieldAnnotation[],
  ): void {
    if (typeof current !== 'object' || current === null) {
      return;
    }

    for (const key of Object.keys(current)) {
      if (key === '.') {
        // The '.' key indicates that the current object itself is managed.
        // We push an annotation for the currentPath.
        annotations.push({
          path: currentPath,
          component: ManagedFieldTooltipComponent,
          inputs: {
            manager: manager,
            time: time,
            timezoneShift: this.timezoneShift,
          },
        });
        continue;
      }

      let fieldName: string | number = key;
      let nextYamlData: unknown = undefined;

      if (key.startsWith('f:')) {
        fieldName = key.substring(2);
        nextYamlData = (currentYamlData as Record<string, unknown>)?.[
          fieldName
        ];
      } else if (key.startsWith('k:')) {
        try {
          const matchKeys = JSON.parse(key.substring(2)) as Record<
            string,
            unknown
          >;
          if (Array.isArray(currentYamlData)) {
            const index = currentYamlData.findIndex((item) => {
              if (typeof item !== 'object' || item === null) return false;
              for (const [mKey, mVal] of Object.entries(matchKeys)) {
                if ((item as Record<string, unknown>)[mKey] !== mVal) {
                  return false;
                }
              }
              return true;
            });
            if (index !== -1) {
              fieldName = index;
              nextYamlData = currentYamlData[index];
            } else {
              fieldName = key;
            }
          } else {
            fieldName = key;
          }
        } catch (e) {
          fieldName = key;
        }
      } else if (key.startsWith('v:')) {
        try {
          const matchVal = JSON.parse(key.substring(2));
          if (Array.isArray(currentYamlData)) {
            const index = currentYamlData.findIndex(
              (item) => item === matchVal,
            );
            if (index !== -1) {
              fieldName = index;
              nextYamlData = currentYamlData[index];
            } else {
              fieldName = key;
            }
          } else {
            fieldName = key;
          }
        } catch (e) {
          fieldName = key;
        }
      }

      const newPath = [...currentPath, fieldName];
      const childObject = current[key] as Record<string, unknown>;

      const hasChildren =
        childObject &&
        typeof childObject === 'object' &&
        Object.keys(childObject).some((k) => k !== '.');

      if (!hasChildren) {
        annotations.push({
          path: newPath,
          component: ManagedFieldTooltipComponent,
          inputs: {
            manager: manager,
            time: time,
            timezoneShift: this.timezoneShift,
          },
        });
      } else {
        this.extractPaths(
          childObject,
          nextYamlData,
          newPath,
          manager,
          time,
          annotations,
        );
      }
    }
  }
}
