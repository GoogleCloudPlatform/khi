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

import { RevisionState, RevisionVerb } from '../generated';
import { ResourceRevision } from './revision';

describe('ResourceRevision', () => {
  describe('get duration', () => {
    it('returns the calculated duration from startTime and endTime', () => {
      const revision = new ResourceRevision(
        1,
        10,
        RevisionState.RevisionStateExisting,
        RevisionVerb.RevisionVerbUpdate,
        '',
        '',
        false,
        false,
        0,
      );

      const duration = revision.duration;

      expect(duration).toBe(9);
    });
  });

  describe('get parsedManifest', () => {
    it('returns parsed YAML object from the resource content', () => {
      const revision = new ResourceRevision(
        0,
        1,
        RevisionState.RevisionStateExisting,
        RevisionVerb.RevisionVerbUpdate,
        `kind: foo`,
        '',
        false,
        false,
        0,
      );

      const manifest = revision.parsedManifest;

      expect(manifest?.kind).toBe('foo');
    });

    it('returns the cached YAML object', () => {
      const revision = new ResourceRevision(
        0,
        1,
        RevisionState.RevisionStateExisting,
        RevisionVerb.RevisionVerbUpdate,
        `kind: foo`,
        '',
        false,
        false,
        0,
      );

      const firstCall = revision.parsedManifest;
      const secoundCall = revision.parsedManifest;

      // The result must be an object with the same reference.
      expect(firstCall).toBe(secoundCall);
    });
  });
});
