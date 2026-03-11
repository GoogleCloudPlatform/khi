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

import {
  formatTimeLabel,
  parsePrincipal,
  PrincipalType,
} from './diff-list.component';

describe('DiffListComponent utils', () => {
  describe('formatTimeLabel', () => {
    it('formats time with 0 timezone shift', () => {
      // 2024-01-01T12:34:56Z
      const timeInMs = Date.UTC(2024, 0, 1, 12, 34, 56);
      expect(formatTimeLabel(timeInMs, 0)).toBe('12:34:56');
    });

    it('formats time with positive timezone shift', () => {
      // +9 hours (JST)
      const timeInMs = Date.UTC(2024, 0, 1, 12, 34, 56);
      expect(formatTimeLabel(timeInMs, 9)).toBe('21:34:56');
    });

    it('formats time with negative timezone shift', () => {
      // -5 hours (EST)
      const timeInMs = Date.UTC(2024, 0, 1, 12, 34, 56);
      expect(formatTimeLabel(timeInMs, -5)).toBe('07:34:56');
    });

    it('handles padding correctly for single digit times', () => {
      // 09:05:01
      const timeInMs = Date.UTC(2024, 0, 1, 9, 5, 1);
      expect(formatTimeLabel(timeInMs, 0)).toBe('09:05:01');
    });
  });

  describe('parsePrincipal', () => {
    it('returns empty/NotAvailable when value is empty', () => {
      const principal = parsePrincipal('');
      expect(principal.type).toBe(PrincipalType.NotAvailable);
      expect(principal.short).toBe('');
      expect(principal.full).toBe('');
    });

    it('handles system:serviceaccount: principals', () => {
      const principal = parsePrincipal(
        'system:serviceaccount:kube-system:default',
      );
      expect(principal.type).toBe(PrincipalType.ServiceAccount);
      expect(principal.short).toBe('kube-system:default');
      expect(principal.full).toBe('system:serviceaccount:kube-system:default');
    });

    it('handles system:node: principals', () => {
      const principal = parsePrincipal('system:node:gke-cluster-pool-123');
      expect(principal.type).toBe(PrincipalType.Node);
      expect(principal.short).toBe('gke-cluster-pool-123');
      expect(principal.full).toBe('system:node:gke-cluster-pool-123');
    });

    it('handles system: principals', () => {
      const principal = parsePrincipal('system:kube-controller-manager');
      expect(principal.type).toBe(PrincipalType.System);
      expect(principal.short).toBe('kube-controller-manager');
      expect(principal.full).toBe('system:kube-controller-manager');
    });

    it('defaults to User type for other values', () => {
      const principal = parsePrincipal('user@example.com');
      expect(principal.type).toBe(PrincipalType.User);
      expect(principal.short).toBe('user@example.com');
      expect(principal.full).toBe('user@example.com');
    });
  });
});
