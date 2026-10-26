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

import { signal, WritableSignal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { InspectionDataStore } from 'src/app/services/inspection-data-store.service';
import {
  StringMaskService,
  compileStringMaskTransform,
  formatDefaultMaskReplacement,
} from 'src/app/services/mask/string-mask.service';
import { InspectionData } from 'src/app/store/domain/inspection-data';
import { InternPoolStore } from 'src/app/store/domain/intern-pool-store';
import { LogStore } from 'src/app/store/domain/log-store';
import { StyleStore } from 'src/app/store/domain/style-store';
import { TimelineStore } from 'src/app/store/domain/timeline-store';

describe('StringMaskService', () => {
  let service: StringMaskService;
  let inspectionDataSignal: WritableSignal<InspectionData | null>;
  let internPool: InternPoolStore;
  let timelineStore: TimelineStore;

  const mockColor = { r: 0, g: 0, b: 0, a: 1 };

  beforeEach(() => {
    internPool = InternPoolStore.create();
    const styleStore = new StyleStore();
    styleStore.addTimelineTypes([
      {
        id: 1,
        label: 'pod',
        description: '',
        icon: '',
        backgroundColor: mockColor,
        foregroundColor: mockColor,
        typeChipBackgroundColor: mockColor,
        typeChipForegroundColor: mockColor,
        visible: true,
        sortPriority: 0,
        height: 1,
      },
    ]);
    const logStore = LogStore.create(internPool, styleStore, 0);
    timelineStore = TimelineStore.create(internPool, styleStore, logStore);

    internPool.addStrings([
      { id: 1, value: 'core/v1' },
      { id: 2, value: 'pod' },
      { id: 3, value: 'prod-ns' },
      { id: 4, value: 'web-pod-1' },
      { id: 5, value: 'web-pod-2' },
      { id: 6, value: 'node' },
      { id: 7, value: 'cluster-scope' },
      { id: 8, value: 'gke-node-1' },
      { id: 9, value: 'prod-ns/web-pod-1[uid-123]' },
    ]);

    timelineStore.addTimelines([
      {
        id: 1,
        timelineTypeId: 1,
        nameStringId: 1,
        parentTimelineId: 0,
        revisionIds: [],
        eventIds: [],
      },
      {
        id: 2,
        timelineTypeId: 1,
        nameStringId: 2,
        parentTimelineId: 1,
        revisionIds: [],
        eventIds: [],
      },
      {
        id: 3,
        timelineTypeId: 1,
        nameStringId: 3,
        parentTimelineId: 2,
        revisionIds: [],
        eventIds: [],
      },
      {
        id: 4,
        timelineTypeId: 1,
        nameStringId: 4,
        parentTimelineId: 3,
        revisionIds: [],
        eventIds: [],
      },
      {
        id: 5,
        timelineTypeId: 1,
        nameStringId: 5,
        parentTimelineId: 3,
        revisionIds: [],
        eventIds: [],
      },
      {
        id: 6,
        timelineTypeId: 1,
        nameStringId: 6,
        parentTimelineId: 1,
        revisionIds: [],
        eventIds: [],
      },
      {
        id: 7,
        timelineTypeId: 1,
        nameStringId: 7,
        parentTimelineId: 6,
        revisionIds: [],
        eventIds: [],
      },
      {
        id: 8,
        timelineTypeId: 1,
        nameStringId: 8,
        parentTimelineId: 7,
        revisionIds: [],
        eventIds: [],
      },
      {
        id: 9,
        timelineTypeId: 1,
        nameStringId: 9,
        parentTimelineId: 8,
        revisionIds: [],
        eventIds: [],
      },
    ]);
    timelineStore.shrinkToFit();

    inspectionDataSignal = signal<InspectionData | null>({
      internPool,
      timelineStore,
    } as unknown as InspectionData);

    TestBed.configureTestingModule({
      providers: [
        StringMaskService,
        {
          provide: InspectionDataStore,
          useValue: {
            inspectionData: inspectionDataSignal,
          },
        },
      ],
    });

    service = TestBed.inject(StringMaskService);
  });

  it('should format sequential numbered default mask replacements', () => {
    expect(formatDefaultMaskReplacement(1)).toBe('<masked_1>');
    expect(formatDefaultMaskReplacement(12)).toBe('<masked_12>');
    expect(formatDefaultMaskReplacement(0)).toBe('<masked_1>');
  });

  it('should apply longest-match-first single-pass masking without cascading into <masked_N> tokens', () => {
    const transform = compileStringMaskTransform([
      { target: 'prod', replacement: '<masked_1>', enabled: true },
      { target: 'prod-ns', replacement: '', enabled: true },
      { target: 'masked', replacement: '<masked_2>', enabled: true },
      { target: '1', replacement: '<masked_3>', enabled: true },
    ]);
    expect(transform).not.toBeNull();
    expect(transform!('ns=prod-ns env=prod id=1')).toBe(
      'ns= env=<masked_1> id=<masked_3>',
    );
  });

  it('should allocate sequential <masked_N> placeholders, preserve numbers across removals, and reset when emptied', () => {
    service.toggleTarget('web-pod-1', true);
    service.toggleTarget('web-pod-2', true);
    service.addCustomRule('gke-node-1');

    expect(
      service
        .rules()
        .map((r) => [r.target, r.replacement, r.defaultReplacement]),
    ).toEqual([
      ['web-pod-1', '<masked_1>', '<masked_1>'],
      ['web-pod-2', '<masked_2>', '<masked_2>'],
      ['gke-node-1', '<masked_3>', '<masked_3>'],
    ]);

    // Removing a middle rule or disabling/re-enabling an existing rule keeps assigned numbers stable.
    service.removeRule('web-pod-2');
    service.toggleTarget('web-pod-1', false);
    service.toggleTarget('web-pod-1', true);
    service.toggleTarget('prod-ns', true);

    expect(service.rules().map((r) => [r.target, r.replacement])).toEqual([
      ['web-pod-1', '<masked_1>'],
      ['gke-node-1', '<masked_3>'],
      ['prod-ns', '<masked_4>'],
    ]);

    // Resetting restores the rule's own defaultReplacement, or allocates a fresh one if claimed by another rule.
    service.updateReplacement('web-pod-1', 'CUSTOM');
    service.resetReplacement('web-pod-1');
    expect(service.rules()[0].replacement).toBe('<masked_1>');

    service.updateReplacement('web-pod-1', 'CUSTOM');
    service.updateReplacement('gke-node-1', '<masked_1>');
    service.resetReplacement('web-pod-1');
    expect(service.rules()[0].replacement).toBe('<masked_5>');
    expect(service.rules()[0].defaultReplacement).toBe('<masked_5>');

    // Removing all remaining rules resets the counter back to <masked_1>.
    service.removeRule('web-pod-1');
    service.removeRule('gke-node-1');
    service.removeRule('prod-ns');
    expect(service.rules()).toEqual([]);

    service.addTargets(['web-pod-1', 'web-pod-2']);
    expect(service.rules().map((r) => [r.target, r.replacement])).toEqual([
      ['web-pod-1', '<masked_1>'],
      ['web-pod-2', '<masked_2>'],
    ]);

    // clearAllRules also resets sequential numbering back to <masked_1>.
    service.clearAllRules();
    service.toggleTarget('prod-ns', true);
    expect(service.rules()[0].replacement).toBe('<masked_1>');
  });

  it('should synchronize live mask transform with InternPoolStore and allow empty replacement deletion', () => {
    service.toggleTarget('web-pod-1', true);
    TestBed.flushEffects();

    expect(internPool.getString(4)).toBe('<masked_1>');
    expect(service.applyMask('name: web-pod-1')).toBe('name: <masked_1>');

    // Empty string replacement deletes the substring.
    service.updateReplacement('web-pod-1', '');
    TestBed.flushEffects();

    expect(internPool.getString(4)).toBe('');
    expect(timelineStore.getTimeline(4).name).toBe('');
    expect(service.applyMask('pod=web-pod-1!')).toBe('pod=!');

    // Disabling live preview restores unmasked getString while retaining rules.
    service.setLivePreviewEnabled(false);
    TestBed.flushEffects();
    expect(internPool.getString(4)).toBe('web-pod-1');
    expect(service.enabledRules().length).toBe(1);
  });
});
