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

import { IconAtlasDTO, StyleStore } from 'src/app/store/domain/style-store';
import { RevisionStateStyle } from 'src/app/store/domain/style';

describe('StyleStore', () => {
  let store: StyleStore;
  const mockColor = { r: 1, g: 0.5, b: 0.2, a: 1 };

  beforeEach(() => {
    store = new StyleStore();
  });

  it('should add and retrieve severities correctly', () => {
    const item = {
      id: 10,
      label: 'CRITICAL',
      shortLabel: 'C',
      backgroundColor: mockColor,
      foregroundColor: mockColor,
      order: 1,
    };

    store.addSeverities([item]);
    const retrieved = store.getSeverity(10);

    expect(retrieved).toEqual(item);
    expect(() => store.getSeverity(99)).toThrowError(
      'Severity ID 99 not found',
    );
  });

  it('should add and retrieve log types correctly', () => {
    const item = {
      id: 5,
      label: 'Kubelet',
      description: 'Kubelet core services',
      backgroundColor: mockColor,
      foregroundColor: mockColor,
    };

    store.addLogTypes([item]);
    const retrieved = store.getLogType(5);

    expect(retrieved).toEqual(item);
    expect(() => store.getLogType(99)).toThrowError('LogType ID 99 not found');
  });

  it('should add and retrieve verbs correctly', () => {
    const item = {
      id: 2,
      label: 'Create',
      backgroundColor: mockColor,
      foregroundColor: mockColor,
      visible: true,
    };

    store.addVerbs([item]);
    const retrieved = store.getVerb(2);

    expect(retrieved).toEqual(item);
    expect(() => store.getVerb(99)).toThrowError('Verb ID 99 not found');
  });

  it('should add and retrieve revision states correctly', () => {
    const item = {
      id: 3,
      label: 'Terminated',
      icon: 'cancel',
      description: 'Resource instance terminated',
      backgroundColor: mockColor,
      style: RevisionStateStyle.NORMAL,
    };

    store.addRevisionStates([item]);
    const retrieved = store.getRevisionState(3);

    expect(retrieved).toEqual(item);
    expect(() => store.getRevisionState(99)).toThrowError(
      'RevisionState ID 99 not found',
    );
  });

  it('should add and retrieve timeline types correctly', () => {
    const item = {
      id: 8,
      label: 'Deployment',
      description: 'Replicaset deployment status',
      icon: 'workspaces',
      backgroundColor: mockColor,
      foregroundColor: mockColor,
      typeChipBackgroundColor: mockColor,
      visible: true,
      sortPriority: 100,
      height: 1,
    };

    store.addTimelineTypes([item]);
    const retrieved = store.getTimelineType(8);

    expect(retrieved).toEqual(item);
    expect(() => store.getTimelineType(99)).toThrowError(
      'TimelineType ID 99 not found',
    );
  });

  describe('Array-returning getters', () => {
    it('should return all added elements, filtering out undefined slots', () => {
      const severity1 = {
        id: 2,
        label: 'WARNING',
        shortLabel: 'W',
        backgroundColor: mockColor,
        foregroundColor: mockColor,
        order: 2,
      };
      const severity2 = {
        id: 5,
        label: 'ERROR',
        shortLabel: 'E',
        backgroundColor: mockColor,
        foregroundColor: mockColor,
        order: 3,
      };
      store.addSeverities([severity1, severity2]);
      expect(store.severities).toEqual([severity1, severity2]);

      const logType1 = {
        id: 1,
        label: 'API Server',
        description: 'Kubernetes API Server logs',
        backgroundColor: mockColor,
        foregroundColor: mockColor,
      };
      const logType2 = {
        id: 3,
        label: 'Controller Manager',
        description: 'Kubernetes Controller Manager logs',
        backgroundColor: mockColor,
        foregroundColor: mockColor,
      };
      store.addLogTypes([logType1, logType2]);
      expect(store.logTypes).toEqual([logType1, logType2]);

      const verb1 = {
        id: 0,
        label: 'Get',
        backgroundColor: mockColor,
        foregroundColor: mockColor,
        visible: true,
      };
      const verb2 = {
        id: 2,
        label: 'Create',
        backgroundColor: mockColor,
        foregroundColor: mockColor,
        visible: true,
      };
      store.addVerbs([verb1, verb2]);
      expect(store.verbs).toEqual([verb1, verb2]);

      const revisionState1 = {
        id: 1,
        label: 'Running',
        icon: 'play_arrow',
        description: 'Resource is running',
        backgroundColor: mockColor,
        style: RevisionStateStyle.NORMAL,
      };
      const revisionState2 = {
        id: 4,
        label: 'Failed',
        icon: 'error',
        description: 'Resource failed',
        backgroundColor: mockColor,
        style: RevisionStateStyle.NORMAL,
      };
      store.addRevisionStates([revisionState1, revisionState2]);
      expect(store.revisionStates).toEqual([revisionState1, revisionState2]);

      const timelineType1 = {
        id: 2,
        label: 'Pod',
        description: 'Pod lifecycle timeline',
        icon: 'pod',
        backgroundColor: mockColor,
        foregroundColor: mockColor,
        typeChipBackgroundColor: mockColor,
        visible: true,
        sortPriority: 10,
        height: 1,
      };
      const timelineType2 = {
        id: 5,
        label: 'Node',
        description: 'Node lifecycle timeline',
        icon: 'node',
        backgroundColor: mockColor,
        foregroundColor: mockColor,
        typeChipBackgroundColor: mockColor,
        visible: true,
        sortPriority: 20,
        height: 1,
      };
      store.addTimelineTypes([timelineType1, timelineType2]);
      expect(store.timelineTypes).toEqual([timelineType1, timelineType2]);
    });
  });

  describe('IconAtlas', () => {
    it('should initialize and retrieve IconAtlas correctly', async () => {
      const mockPngBuffer = new Uint8Array([
        137, 80, 78, 71, 13, 10, 26, 10, 0, 0, 0, 13, 73, 72, 68, 82, 0, 0, 0,
        1, 0, 0, 0, 1, 8, 6, 0, 0, 0, 31, 21, 196, 137, 0, 0, 0, 13, 73, 68, 65,
        84, 120, 1, 99, 96, 96, 0, 0, 0, 2, 0, 1, 73, 175, 167, 2, 0, 0, 0, 0,
        73, 69, 78, 68, 174, 66, 96, 130,
      ]).buffer;

      const mockBMFontJson = JSON.stringify({
        pages: ['page1'],
        chars: [],
        common: {
          lineHeight: 24,
          base: 18,
          scaleW: 256,
          scaleH: 256,
          pages: 1,
          packed: 0,
          alphaChnl: 0,
          redChnl: 0,
          greenChnl: 0,
          blueChnl: 0,
        },
      });
      const mockBMFontBuffer = new TextEncoder().encode(mockBMFontJson).buffer;

      const mockCodepoints = new Map<string, string>([['test-icon', 'e000']]);

      const dto: IconAtlasDTO = {
        msdfIconImage: [mockPngBuffer],
        bmfontJson: mockBMFontBuffer,
        nameToCodepoints: mockCodepoints,
      };

      await store.setIconAtlas(dto);

      const retrieved = store.getIconAtlas();
      expect(retrieved).toBeDefined();
      expect(retrieved!.msdfIconImage.length).toBe(1);
      expect(
        retrieved!.msdfIconImage[0] instanceof HTMLImageElement,
      ).toBeTrue();
      expect((retrieved!.msdfIconImage[0] as HTMLImageElement).width).toBe(1);
      expect(retrieved!.bmfontJson.pages).toEqual(['page1']);
      expect(retrieved!.nameToCodepoints).toEqual(mockCodepoints);
    });
  });
});
