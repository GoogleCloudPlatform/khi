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

import { TestBed } from '@angular/core/testing';
import { Injector } from '@angular/core';
import {
  Reporter,
  ConsoleReporter,
  HierarchicalReporter,
  provideReporterContext,
} from './reporter';

describe('Reporter', () => {
  describe('provideReporterContext', () => {
    it('should create HierarchicalReporter and delegate to parent with labels directly', () => {
      const mockParent = jasmine.createSpyObj('Reporter', ['send']);

      const parentInjector = Injector.create({
        providers: [{ provide: Reporter, useValue: mockParent }],
      });

      const childInjector = Injector.create({
        providers: [provideReporterContext({ feature: 'B' })],
        parent: parentInjector,
      });

      const reporter = childInjector.get(Reporter);
      expect(reporter).toBeTruthy();

      reporter.send({ event: 'test2' });

      expect(mockParent.send).toHaveBeenCalledWith({
        feature: 'B',
        event: 'test2',
      });
    });
  });

  describe('Reporter resolution', () => {
    beforeEach(() => {
      TestBed.configureTestingModule({
        providers: [{ provide: Reporter, useClass: ConsoleReporter }],
      });
    });

    it('should resolve Reporter from DI context', () => {
      const reporter = TestBed.inject(Reporter);
      expect(reporter).toBeTruthy();
      expect(reporter instanceof ConsoleReporter).toBe(true);
    });
  });

  describe('HierarchicalReporter (Manual instantiation)', () => {
    it('should delegate to parent with merged labels', () => {
      const mockParent = jasmine.createSpyObj('Reporter', ['send']);
      const reporter = new HierarchicalReporter({ feature: 'A' }, mockParent);

      reporter.send({ event: 'test' });

      expect(mockParent.send).toHaveBeenCalledWith({
        feature: 'A',
        event: 'test',
      });
    });

    it('should throw error if no parent', () => {
      const reporter = new HierarchicalReporter({ feature: 'A' }, null);

      expect(() => reporter.send({ event: 'test' })).toThrowError(
        'No parent Reporter found',
      );
    });

    it('should accumulate labels via addLabels', () => {
      const mockParent = jasmine.createSpyObj('Reporter', ['send']);
      const reporter = new HierarchicalReporter({ feature: 'A' }, mockParent);

      reporter.addLabels({ service: 'auth' });
      reporter.send({ event: 'test' });

      expect(mockParent.send).toHaveBeenCalledWith({
        feature: 'A',
        service: 'auth',
        event: 'test',
      });
    });
  });
});
