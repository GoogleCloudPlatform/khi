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
  FEATURE_GATE_TASK_REF_LABEL_KEY,
  FORM_TASK_LABEL_KEY,
  getFeatureGateTaskRef,
  getTaskDescription,
  isFormTask,
  TASK_DESCRIPTION_LABEL_KEY,
} from 'src/app/shared/components/dag-viewer/dag-viewer.model';

describe('dag-viewer.model', () => {
  describe('isFormTask', () => {
    it('returns true when FORM_TASK_LABEL_KEY is set to "true"', () => {
      const labels = {
        [FORM_TASK_LABEL_KEY]: 'true',
      };
      expect(isFormTask(labels)).toBeTrue();
    });

    it('returns true when form-field-label is present', () => {
      const labels = {
        'khi.google.com/inspection/form-field-label': 'Project ID',
      };
      expect(isFormTask(labels)).toBeTrue();
    });

    it('returns false when FORM_TASK_LABEL_KEY is not "true"', () => {
      const labels = {
        [FORM_TASK_LABEL_KEY]: 'false',
      };
      expect(isFormTask(labels)).toBeFalse();
    });

    it('returns false for empty labels', () => {
      expect(isFormTask({})).toBeFalse();
    });

    it('returns false for unrelated labels', () => {
      const labels = {
        'khi.google.com/inspection/is-feature': 'true',
        'khi.google.com/test-unrelated-label': 'true',
      };
      expect(isFormTask(labels)).toBeFalse();
    });
  });

  describe('getTaskDescription', () => {
    it('returns description when TASK_DESCRIPTION_LABEL_KEY is present', () => {
      const labels = {
        [TASK_DESCRIPTION_LABEL_KEY]: 'Parses audit logs into timeline events.',
      };
      expect(getTaskDescription(labels)).toBe(
        'Parses audit logs into timeline events.',
      );
    });

    it('returns empty string when labels map is empty', () => {
      expect(getTaskDescription({})).toBe('');
    });

    it('returns empty string when labels is null or undefined', () => {
      expect(getTaskDescription(null)).toBe('');
      expect(getTaskDescription(undefined)).toBe('');
    });

    it('returns empty string without fallback when other description labels exist', () => {
      const labels = {
        'khi.google.com/inspection/form-field-description': 'Form description',
        'khi.google.com/inspection/feature/description': 'Feature description',
      };
      expect(getTaskDescription(labels)).toBe('');
    });
  });

  describe('getFeatureGateTaskRef', () => {
    it('returns the reference string when FEATURE_GATE_TASK_REF_LABEL_KEY is present in labels', () => {
      const labels = {
        [FEATURE_GATE_TASK_REF_LABEL_KEY]: 'k8scontainer/tail-task',
      };
      expect(getFeatureGateTaskRef(labels)).toBe('k8scontainer/tail-task');
    });

    it('returns empty string when labels does not contain the key', () => {
      const labels = {
        'khi.google.com/inspection/is-feature': 'true',
      };
      expect(getFeatureGateTaskRef(labels)).toBe('');
      expect(getFeatureGateTaskRef({})).toBe('');
    });

    it('returns empty string when labels is null or undefined', () => {
      expect(getFeatureGateTaskRef(null)).toBe('');
      expect(getFeatureGateTaskRef(undefined)).toBe('');
    });
  });
});
