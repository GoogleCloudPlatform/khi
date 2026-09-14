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

import { parseJobModeCommand } from './job-command-parser';

describe('parseJobModeCommand', () => {
  it('parses a standard multi-line job mode command', () => {
    const cmd = `./khi \\
  --job-mode \\
  --job-inspection-type="google-cloud-cluster" \\
  --job-inspection-features="cai,audit-log" \\
  --job-inspection-values='{
    "project-id": "my-project",
    "cluster-name": "my-cluster"
  }' \\
  --job-export-destination="output.khi"`;

    const result = parseJobModeCommand(cmd);

    expect(result.inspectionType).toBe('google-cloud-cluster');
    expect(result.features).toEqual(['cai', 'audit-log']);
    expect(result.parameters).toEqual({
      'project-id': 'my-project',
      'cluster-name': 'my-cluster',
    });
  });

  it('parses single-line command with double-quoted values', () => {
    const cmd =
      './khi --job-mode --job-inspection-type="gcp-gke" --job-inspection-features="audit-log" --job-inspection-values="{\\"cluster\\": \\"c1\\"}"';

    const result = parseJobModeCommand(cmd);

    expect(result.inspectionType).toBe('gcp-gke');
    expect(result.features).toEqual(['audit-log']);
    expect(result.parameters).toEqual({ cluster: 'c1' });
  });

  it('handles escaped single quotes inside single-quoted JSON values', () => {
    const cmd =
      "./khi --job-mode --job-inspection-type='gcp-gke' --job-inspection-values='{\"message\": \"it'\\''s fine\"}'";

    const result = parseJobModeCommand(cmd);

    expect(result.inspectionType).toBe('gcp-gke');
    expect(result.parameters).toEqual({ message: "it's fine" });
  });

  it('handles empty or missing features gracefully', () => {
    const cmd =
      './khi --job-mode --job-inspection-type="custom" --job-inspection-features=""';

    const result = parseJobModeCommand(cmd);

    expect(result.inspectionType).toBe('custom');
    expect(result.features).toEqual([]);
    expect(result.parameters).toEqual({});
  });

  it('handles missing --job-inspection-values by returning empty parameters', () => {
    const cmd =
      './khi --job-mode --job-inspection-type="custom" --job-inspection-features="feat1"';

    const result = parseJobModeCommand(cmd);

    expect(result.inspectionType).toBe('custom');
    expect(result.features).toEqual(['feat1']);
    expect(result.parameters).toEqual({});
  });

  it('throws error when command is empty or whitespace only', () => {
    expect(() => parseJobModeCommand('')).toThrowError('Command is empty.');
    expect(() => parseJobModeCommand('   ')).toThrowError('Command is empty.');
  });

  it('throws error when --job-inspection-type is missing', () => {
    const cmd = './khi --job-mode --job-inspection-values=\'{"project": "p"}\'';
    expect(() => parseJobModeCommand(cmd)).toThrowError(
      'Missing required flag: --job-inspection-type',
    );
  });

  it('throws error when --job-inspection-values is invalid JSON', () => {
    const cmd =
      './khi --job-mode --job-inspection-type="gcp" --job-inspection-values=\'{invalid-json}\'';
    expect(() => parseJobModeCommand(cmd)).toThrowError(
      /Failed to parse --job-inspection-values JSON/,
    );
  });

  it('throws error when --job-inspection-values is not a JSON object', () => {
    const cmd =
      './khi --job-mode --job-inspection-type="gcp" --job-inspection-values=\'["array"]\'';
    expect(() => parseJobModeCommand(cmd)).toThrowError(/Expected JSON object/);
  });
});
