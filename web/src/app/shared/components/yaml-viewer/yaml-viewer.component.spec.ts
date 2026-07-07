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
  ComponentFixture,
  TestBed,
  fakeAsync,
  tick,
} from '@angular/core/testing';
import { ElementRef } from '@angular/core';
import { YamlViewerComponent } from 'src/app/shared/components/yaml-viewer/yaml-viewer.component';
import { YamlLine } from 'src/app/shared/components/yaml-viewer/diff-renderer';
import { ValueType } from 'src/app/shared/components/yaml-viewer/diff-util';
import { DiffStatus } from 'src/app/shared/components/yaml-viewer/lcs';
import * as yaml from 'js-yaml';

describe('YamlViewerComponent', () => {
  let component: YamlViewerComponent;
  let fixture: ComponentFixture<YamlViewerComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [YamlViewerComponent],
    }).compileComponents();

    fixture = TestBed.createComponent(YamlViewerComponent);
    component = fixture.componentInstance;
  });

  it('should create the component', () => {
    expect(component).toBeTruthy();
  });

  it('should generate preview lines when leftYaml is null', () => {
    fixture.componentRef.setInput('rightYaml', 'metadata:\n  name: my-pod');
    fixture.detectChanges();

    const lines = component.lines();
    expect(lines.length).toBe(2);

    expect(lines[0].key).toBe('metadata');
    expect(lines[0].diffStatus).toBe(DiffStatus.Unchanged);
    expect(lines[0].lineNumber).toBe(1);

    expect(lines[1].key).toBe('name');
    expect(lines[1].valueText).toBe('my-pod');
    expect(lines[1].diffStatus).toBe(DiffStatus.Unchanged);
    expect(lines[1].lineNumber).toBe(2);
  });

  it('should generate diff status for additions, deletions, and modifications', () => {
    fixture.componentRef.setInput(
      'leftYaml',
      'metadata:\n  name: old-pod\n  uid: 123',
    );
    fixture.componentRef.setInput(
      'rightYaml',
      'metadata:\n  name: new-pod\n  namespace: default',
    );
    fixture.detectChanges();

    const lines = component.lines();

    // metadata is unchanged
    const metaLine = lines.find((l) => l.key === 'metadata');
    expect(metaLine).toBeTruthy();
    expect(metaLine?.diffStatus).toBe(DiffStatus.Unchanged);

    // name: modified (split into deleted + added)
    const nameDeletedLine = lines.find(
      (l) => l.key === 'name' && l.diffStatus === DiffStatus.Deleted,
    );
    const nameAddedLine = lines.find(
      (l) => l.key === 'name' && l.diffStatus === DiffStatus.Added,
    );
    expect(nameDeletedLine).toBeTruthy();
    expect(nameDeletedLine?.valueText).toBe('old-pod');
    expect(nameDeletedLine?.lineNumber).toBeUndefined();
    expect(nameDeletedLine?.valueSegments).toEqual([
      { text: 'old', diffStatus: DiffStatus.Deleted },
      { text: '-pod', diffStatus: DiffStatus.Unchanged },
    ]);

    expect(nameAddedLine).toBeTruthy();
    expect(nameAddedLine?.valueText).toBe('new-pod');
    expect(nameAddedLine?.lineNumber).toBe(2); // line number 1: metadata, 2: name (added)
    expect(nameAddedLine?.valueSegments).toEqual([
      { text: 'new', diffStatus: DiffStatus.Added },
      { text: '-pod', diffStatus: DiffStatus.Unchanged },
    ]);

    // uid: deleted
    const uidLine = lines.find((l) => l.key === 'uid');
    expect(uidLine).toBeTruthy();
    expect(uidLine?.diffStatus).toBe(DiffStatus.Deleted);
    expect(uidLine?.valueText).toBe('123');
    expect(uidLine?.lineNumber).toBeUndefined();

    // namespace: added
    const nsLine = lines.find((l) => l.key === 'namespace');
    expect(nsLine).toBeTruthy();
    expect(nsLine?.diffStatus).toBe(DiffStatus.Added);
    expect(nsLine?.valueText).toBe('default');
    expect(nsLine?.lineNumber).toBe(3); // Line number 3: namespace
  });

  it('should bind annotations to specified JSON paths', () => {
    fixture.componentRef.setInput('rightYaml', 'metadata:\n  name: my-pod');
    
    class FakeProvider {
      getAnnotations() {
        return [
          {
            path: ['metadata', 'name'],
            component: {} as any,
            inputs: { testData: 'Specifies resource name' },
          }
        ];
      }
    }
    
    fixture.componentRef.setInput('annotationProviders', [new FakeProvider()]);
    fixture.detectChanges();

    const lines = component.lines();
    const nameLine = lines.find((l) => l.key === 'name');
    expect(nameLine).toBeTruthy();
    expect(nameLine?.annotation).toBeTruthy();
    expect(nameLine?.annotation?.inputs?.['testData']).toBe('Specifies resource name');
  });

  it('should split text into highlighted segments matching the query across key and value', () => {
    fixture.componentRef.setInput('rightYaml', '  image: nginx:1.14.2');
    fixture.detectChanges();

    const line: YamlLine = {
      text: '  image: nginx:1.14.2',
      indent: 2,
      key: 'image',
      valueText: 'nginx:1.14.2',
      valueType: ValueType.String,
      valueSegments: [
        { text: 'nginx:1.14.2', diffStatus: DiffStatus.Unchanged },
      ],
      diffStatus: DiffStatus.Unchanged,
      path: 'image',
    };
    // Search for 'image: nginx' which spans across the key, colon, and value
    const segments = component['getRenderSegments'](line, 'image: nginx', 0);

    expect(segments).toEqual([
      {
        text: 'image',
        diffStatus: DiffStatus.Unchanged,
        isMatch: true,
        isActiveMatch: false,
        isKey: true,
      },
      {
        text: ': ',
        diffStatus: DiffStatus.Unchanged,
        isMatch: true,
        isActiveMatch: false,
        isColon: true,
      },
      {
        text: 'nginx',
        diffStatus: DiffStatus.Unchanged,
        isMatch: true,
        isActiveMatch: false,
        isValue: true,
      },
      {
        text: ':1.14.2',
        diffStatus: DiffStatus.Unchanged,
        isMatch: false,
        isActiveMatch: false,
        isValue: true,
      },
    ]);
  });

  it('should set isArrayElementStart correctly for arrays of scalars and objects', () => {
    fixture.componentRef.setInput(
      'rightYaml',
      `items:
  - foo
  - name: bar
    value: baz`,
    );
    fixture.detectChanges();

    const lines = component.lines();
    // Expected lines:
    // 0: items: (isArrayElementStart: false)
    // 1: - foo (isArrayElementStart: true)
    // 2: - name: bar (isArrayElementStart: true)
    // 3:   value: baz (isArrayElementStart: false)
    expect(lines.length).toBe(4);

    expect(lines[0].key).toBe('items');
    expect(lines[0].isArrayElementStart).toBeFalsy();

    expect(lines[1].valueText).toBe('foo');
    expect(lines[1].isArrayElementStart).toBe(true);

    expect(lines[2].key).toBe('name');
    expect(lines[2].valueText).toBe('bar');
    expect(lines[2].isArrayElementStart).toBe(true);

    expect(lines[3].key).toBe('value');
    expect(lines[3].valueText).toBe('baz');
    expect(lines[3].isArrayElementStart).toBeFalsy();
  });

  it('should render multiline strings as block scalars', () => {
    fixture.componentRef.setInput(
      'rightYaml',
      `description: |
  line 1
  line 2`,
    );
    fixture.detectChanges();

    const lines = component.lines();
    // Expected lines:
    // 0: description: |
    // 1:   line 1
    // 2:   line 2
    expect(lines.length).toBe(3);

    expect(lines[0].key).toBe('description');
    expect(lines[0].valueText).toBe('|');
    expect(lines[0].valueType).toBe('none');
    expect(lines[0].valueSegments).toEqual([
      { text: '|', diffStatus: DiffStatus.Unchanged },
    ]);

    expect(lines[1].key).toBeUndefined();
    expect(lines[1].valueText).toBe('line 1');
    expect(lines[1].valueType).toBe('string');
    expect(lines[1].indent).toBe(2);

    expect(lines[2].key).toBeUndefined();
    expect(lines[2].valueText).toBe('line 2');
    expect(lines[2].valueType).toBe('string');
    expect(lines[2].indent).toBe(2);
  });

  it('should detect and render moved array elements', () => {
    fixture.componentRef.setInput(
      'leftYaml',
      `items:
  - foo
  - bar
  - baz`,
    );
    fixture.componentRef.setInput(
      'rightYaml',
      `items:
  - bar
  - foo
  - baz`,
    );
    fixture.detectChanges();

    const lines = component.lines();
    // Expected lines:
    // 0: items:
    // 1: - bar ⇠ moved from [1] (moved-in, now at [0])
    // 2: - foo (unchanged, shifted, now at [1])
    // 3: - bar ➔ moved to [0] (moved-out, was at [1])
    // 4: - baz (unchanged)
    expect(lines.length).toBe(5);

    expect(lines[0].key).toBe('items');

    // - bar (moved-in at 0)
    expect(lines[1].valueText).toBe('bar');
    expect(lines[1].diffStatus).toBe(DiffStatus.MovedIn);
    expect(lines[1].movedFrom).toBe('[1]');
    expect(lines[1].text).not.toContain('⇠ moved from [1]');

    // - foo (now at 1, unchanged/shifted)
    expect(lines[2].valueText).toBe('foo');
    expect(lines[2].diffStatus).toBe(DiffStatus.Unchanged);

    // - bar (moved-out from 1)
    expect(lines[3].valueText).toBe('bar');
    expect(lines[3].diffStatus).toBe(DiffStatus.MovedOut);
    expect(lines[3].movedTo).toBe('[0]');
    expect(lines[3].text).not.toContain('➔ moved to [0]');
    expect(lines[3].lineNumber).toBeUndefined();

    // - baz
    expect(lines[4].valueText).toBe('baz');
    expect(lines[4].diffStatus).toBe(DiffStatus.Unchanged);
  });

  it('should detect and render moved object array elements', () => {
    const leftVal = {
      conditions: [
        { type: 'Ready', status: 'True' },
        { type: 'PodScheduled', status: 'True' },
      ],
    };
    const rightVal = {
      conditions: [
        { type: 'PodScheduled', status: 'True' },
        { type: 'Ready', status: 'True' },
      ],
    };

    fixture.componentRef.setInput('leftYaml', yaml.dump(leftVal));
    fixture.componentRef.setInput('rightYaml', yaml.dump(rightVal));
    fixture.detectChanges();

    const lines = component.lines();
    expect(lines.length).toBe(7);

    expect(lines[0].key).toBe('conditions');

    // - status: True (moved-in, first property of PodScheduled which moved from [1])
    expect(lines[1].text).toContain('- status: True');
    expect(lines[1].diffStatus).toBe(DiffStatus.MovedIn);
    expect(lines[1].movedFrom).toBe('[1]');

    // type: PodScheduled (moved-in, second property of PodScheduled)
    expect(lines[2].text).toContain('type: PodScheduled');
    expect(lines[2].diffStatus).toBe(DiffStatus.MovedIn);
    expect(lines[2].movedFrom).toBe('[1]');

    // - status: True (unchanged, first property of Ready)
    expect(lines[3].text).toContain('- status: True');
    expect(lines[3].diffStatus).toBe(DiffStatus.Unchanged);

    // type: Ready (unchanged, second property of Ready)
    expect(lines[4].text).toContain('type: Ready');
    expect(lines[4].diffStatus).toBe(DiffStatus.Unchanged);

    // - status: True (moved-out, first property of PodScheduled which moved to [0])
    expect(lines[5].text).toContain('- status: True');
    expect(lines[5].diffStatus).toBe(DiffStatus.MovedOut);
    expect(lines[5].movedTo).toBe('[0]');

    // type: PodScheduled (moved-out, second property of PodScheduled)
    expect(lines[6].text).toContain('type: PodScheduled');
    expect(lines[6].diffStatus).toBe(DiffStatus.MovedOut);
    expect(lines[6].movedTo).toBe('[0]');
  });

  it('should not deeply highlight child elements when the parent object is added or deleted as a whole', () => {
    fixture.componentRef.setInput(
      'leftYaml',
      `metadata:
  name: old-pod`,
    );
    fixture.componentRef.setInput(
      'rightYaml',
      `metadata:
  name: old-pod
newObject:
  key1: value1
  key2: value2`,
    );
    fixture.detectChanges();

    const lines = component.lines();

    // newObject is added, so lines for newObject, key1, and key2 should have diffStatus: Added.
    // However, the value segments for key1 and key2 should have diffStatus: Unchanged (not Added).
    const key1Line = lines.find((l) => l.key === 'key1');
    expect(key1Line).toBeTruthy();
    expect(key1Line?.diffStatus).toBe(DiffStatus.Added);
    expect(key1Line?.valueSegments).toEqual([
      { text: 'value1', diffStatus: DiffStatus.Unchanged },
    ]);

    const key2Line = lines.find((l) => l.key === 'key2');
    expect(key2Line).toBeTruthy();
    expect(key2Line?.diffStatus).toBe(DiffStatus.Added);
    expect(key2Line?.valueSegments).toEqual([
      { text: 'value2', diffStatus: DiffStatus.Unchanged },
    ]);
  });

  describe('Collapsing', () => {
    it('should toggle collapsed paths and filter lines accordingly', () => {
      fixture.componentRef.setInput(
        'rightYaml',
        `metadata:
  name: my-pod
  labels:
    app: khi`,
      );
      fixture.detectChanges();

      const initialLines = component.lines();
      expect(initialLines.length).toBe(4);

      const metadataLine = initialLines[0];
      expect(metadataLine.key).toBe('metadata');
      expect(metadataLine.isCollapsible).toBeTrue();

      // Collapse 'metadata'
      const event = new MouseEvent('click');
      spyOn(event, 'stopPropagation');
      component['toggleCollapsed'](metadataLine, event);
      fixture.detectChanges();

      expect(event.stopPropagation).toHaveBeenCalled();
      expect(component.collapsedPaths().has('metadata')).toBeTrue();

      // Check rendered lines after collapsing.
      // Expected:
      // 0: metadata: ... (valueText becomes '...', children are skipped)
      const collapsedLines = component.lines();
      expect(collapsedLines.length).toBe(1);
      expect(collapsedLines[0].key).toBe('metadata');
      expect(collapsedLines[0].valueText).toBe('...');

      // Expand 'metadata' again
      component['toggleCollapsed'](metadataLine, event);
      fixture.detectChanges();
      expect(component.collapsedPaths().has('metadata')).toBeFalse();
      expect(component.lines().length).toBe(4);
    });
  });

  describe('Auto Scrolling', () => {
    let mockElements: HTMLElement[];

    beforeEach(() => {
      // Mock the DOM elements querySelectorAll return value.
      // We need at least 6 elements to cover the diff blocks in the tests.
      mockElements = [
        document.createElement('div'),
        document.createElement('div'),
        document.createElement('div'),
        document.createElement('div'),
        document.createElement('div'),
        document.createElement('div'),
      ];
      mockElements.forEach((el) => {
        el.classList.add('yaml-line-wrapper');
        spyOn(el, 'scrollIntoView');
      });

      const elRef = fixture.debugElement.injector.get(ElementRef) as ElementRef;
      spyOn(elRef.nativeElement, 'querySelectorAll').and.callFake(
        (selector: string) => {
          if (selector === '.yaml-line-wrapper') {
            return mockElements as unknown as NodeListOf<Element>;
          }
          return null as unknown as NodeListOf<Element>;
        },
      );
    });

    it('should scroll to the active search match when activeMatchIndex changes', fakeAsync(() => {
      // Use valid YAML object format so each property is on its own line
      fixture.componentRef.setInput(
        'rightYaml',
        'key1: line1\nkey2: line2\nkey3: line3',
      );
      fixture.componentRef.setInput('searchQuery', 'line');
      fixture.detectChanges();

      expect(component.matches().length).toBe(3);

      // Trigger activeMatchIndex change (1-based index 2 -> mockElements[1] which is line2)
      fixture.componentRef.setInput('activeMatchIndex', 2);
      fixture.detectChanges();

      // Resolve the setTimeout in the effect
      tick();

      expect(mockElements[1].scrollIntoView).toHaveBeenCalledWith({
        behavior: 'smooth',
        block: 'center',
      });
      expect(mockElements[0].scrollIntoView).not.toHaveBeenCalled();
      expect(mockElements[2].scrollIntoView).not.toHaveBeenCalled();
    }));

    it('should scroll to the active diff block when activeDiffIndex changes', fakeAsync(() => {
      // Setup a YAML with 2 distinct diff blocks separated by unchanged key2 and key3
      fixture.componentRef.setInput(
        'leftYaml',
        'key1: line1\nkey2: line2\nkey3: line3\nkey4: line4',
      );
      fixture.componentRef.setInput(
        'rightYaml',
        'key1: line1_added\nkey2: line2\nkey3: line3\nkey4: line4_added',
      );
      fixture.detectChanges();

      // Diff blocks should be:
      // Block 1: key1 (index 0)
      // Block 2: key4 (index 4 - because key1 has delete/add pair: indices 0 and 1, key2 is 2, key3 is 3, key4 is 4)
      expect(component.diffLineIndices().length).toBe(2);
      expect(component.diffLineIndices()).toEqual([0, 4]);

      // Trigger activeDiffIndex change (1-based index 2 -> second diff block, which starts at index 4)
      fixture.componentRef.setInput('activeDiffIndex', 2);
      fixture.detectChanges();

      tick();

      expect(mockElements[4].scrollIntoView).toHaveBeenCalledWith({
        behavior: 'smooth',
        block: 'center',
      });
      expect(mockElements[0].scrollIntoView).not.toHaveBeenCalled();
    }));
  });
});
