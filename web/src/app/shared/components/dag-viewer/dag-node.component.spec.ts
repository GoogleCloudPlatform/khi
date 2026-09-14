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

import { Component } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { DagNodeComponent } from 'src/app/shared/components/dag-viewer/dag-node.component';
import { ProvidedTagInfo } from 'src/app/generated/api/v1/inspection_task_graph_pb';
import {
  DagNodeRunPhase,
  DagPositionedNode,
  TASK_DESCRIPTION_LABEL_KEY,
} from 'src/app/shared/components/dag-viewer/dag-viewer.model';

@Component({
  standalone: true,
  imports: [DagNodeComponent],
  template: `
    <svg>
      <g
        khi-dag-node
        [node]="node"
        [isSelected]="isSelected"
        [isHighlighted]="isHighlighted"
        [isDimmed]="isDimmed"
        (selectNode)="onSelectNode($event)"
      ></g>
    </svg>
  `,
})
class TestHostComponent {
  node: DagPositionedNode = mockNode;
  isSelected = false;
  isHighlighted = false;
  isDimmed = false;
  emittedNode?: DagPositionedNode;

  onSelectNode(node: DagPositionedNode): void {
    this.emittedNode = node;
  }
}

const mockNode: DagPositionedNode = {
  id: 'khi.google.com/test-task#1a2b3c',
  referenceId: 'khi.google.com/test-task',
  isFeature: false,
  isFormTask: false,
  topologicalOrder: 2,
  priority: 100,
  labels: {},
  outputType: 'string',
  providedTags: [],
  runPhase: DagNodeRunPhase.NONE,
  runDurationMs: 0,
  x: 120,
  y: 80,
  width: 420,
  height: 88,
  layer: 1,
};

describe('DagNodeComponent', () => {
  let hostComponent: TestHostComponent;
  let fixture: ComponentFixture<TestHostComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [TestHostComponent],
    }).compileComponents();

    fixture = TestBed.createComponent(TestHostComponent);
    hostComponent = fixture.componentInstance;
  });

  it('renders node texts and positions correctly', () => {
    hostComponent.node = mockNode;
    fixture.detectChanges();

    const group = fixture.nativeElement.querySelector('.dag-node-group');
    expect(group).toBeTruthy();
    expect(group.getAttribute('transform')).toBe('translate(120, 80)');

    const titleEl = fixture.nativeElement.querySelector('title');
    expect(titleEl.textContent.trim()).toBe(
      'khi.google.com/test-task\nOutput: string',
    );

    const domainText = fixture.nativeElement.querySelector('.node-domain-text');
    expect(domainText.textContent.trim()).toBe('khi.google.com/');

    const taskText = fixture.nativeElement.querySelector('.node-task-text');
    expect(taskText.textContent.trim()).toBe('test-task');
    expect(taskText.getAttribute('y')).toBe('35');

    const implText = fixture.nativeElement.querySelector('.node-impl-text');
    expect(implText.textContent.trim()).toBe('#1a2b3c');
    expect(implText.getAttribute('x')).toBe('406');
    expect(implText.getAttribute('y')).toBe('18');

    const outputText = fixture.nativeElement.querySelector(
      '.node-output-type-text',
    );
    expect(outputText.textContent.trim()).toBe('string');
    expect(outputText.getAttribute('x')).toBe('14');
    expect(outputText.getAttribute('y')).toBe('50');
  });

  it('renders task without domain correctly', () => {
    hostComponent.node = {
      ...mockNode,
      id: 'simple-task#abc',
      referenceId: 'simple-task',
    };
    fixture.detectChanges();

    const domainText = fixture.nativeElement.querySelector('.node-domain-text');
    expect(domainText).toBeNull();

    const taskText = fixture.nativeElement.querySelector('.node-task-text');
    expect(taskText.textContent.trim()).toBe('simple-task');
    expect(taskText.getAttribute('y')).toBe('26');
  });

  it('renders feature and input form task badges when enabled', () => {
    hostComponent.node = {
      ...mockNode,
      isFeature: true,
      isFormTask: true,
    };
    fixture.detectChanges();

    const group = fixture.nativeElement.querySelector('.dag-node-group');
    expect(group.classList.contains('feature')).toBeTrue();
    expect(group.classList.contains('form-task')).toBeTrue();

    const featureBadge = fixture.nativeElement.querySelector('.feature-badge');
    expect(featureBadge).toBeTruthy();

    const inputBadge = fixture.nativeElement.querySelector('.input-badge');
    expect(inputBadge).toBeTruthy();
    expect(inputBadge.getAttribute('transform')).toBe('translate(62, 0)');
  });

  it('positions input badge correctly when node is form task but not a feature', () => {
    hostComponent.node = {
      ...mockNode,
      isFeature: false,
      isFormTask: true,
    };
    fixture.detectChanges();

    const featureBadge = fixture.nativeElement.querySelector('.feature-badge');
    expect(featureBadge).toBeNull();

    const inputBadge = fixture.nativeElement.querySelector('.input-badge');
    expect(inputBadge).toBeTruthy();
    expect(inputBadge.getAttribute('transform')).toBe('translate(0, 0)');
  });

  it('truncates long task names with ellipsis based on node width', () => {
    hostComponent.node = {
      ...mockNode,
      referenceId:
        'khi.google.com/a-very-long-task-name-exceeding-card-width-limit-that-keeps-going-on-and-on',
    };
    fixture.detectChanges();

    const taskText = fixture.nativeElement.querySelector('.node-task-text');
    expect(taskText.textContent.trim()).toBe(
      'a-very-long-task-name-exceeding-card-width-limit-...',
    );
  });

  it('emits selectNode when clicked', () => {
    hostComponent.node = mockNode;
    fixture.detectChanges();

    const group = fixture.nativeElement.querySelector(
      '.dag-node-group',
    ) as SVGGraphicsElement;
    group.dispatchEvent(new MouseEvent('click', { bubbles: true }));

    expect(hostComponent.emittedNode).toEqual(mockNode);
  });

  it('applies selected, highlighted, and dimmed CSS classes based on inputs', () => {
    hostComponent.node = mockNode;
    hostComponent.isSelected = true;
    hostComponent.isHighlighted = true;
    hostComponent.isDimmed = true;
    fixture.detectChanges();

    const group = fixture.nativeElement.querySelector('.dag-node-group');
    expect(group.classList.contains('selected')).toBeTrue();
    expect(group.classList.contains('highlighted')).toBeTrue();
    expect(group.classList.contains('dimmed')).toBeTrue();
  });

  it('renders provided tag badges in alphabetical order with output types', () => {
    hostComponent.node = {
      ...mockNode,
      providedTags: [
        {
          $typeName: 'api.v1.ProvidedTagInfo',
          tag: 'zebra',
          outputType: 'int',
          priority: 0,
        },
        {
          $typeName: 'api.v1.ProvidedTagInfo',
          tag: 'apple',
          outputType: 'string',
          priority: 0,
        },
        {
          $typeName: 'api.v1.ProvidedTagInfo',
          tag: 'banana',
          outputType: '',
          priority: 0,
        },
      ] as unknown as readonly ProvidedTagInfo[],
    };
    fixture.detectChanges();

    const root = fixture.nativeElement as HTMLElement;
    const tagBadges = Array.from(root.querySelectorAll('.tag-badge'));
    expect(tagBadges.length).toBe(3);

    const texts = tagBadges.map((b) =>
      b.querySelector('.badge-text')?.textContent?.trim(),
    );
    expect(texts).toEqual(['#apple', '#banana', '#zebra']);

    const tooltips = tagBadges.map((b) =>
      b.querySelector('title')?.textContent?.trim(),
    );
    expect(tooltips).toEqual(['apple (string)', 'banana', 'zebra (int)']);
  });

  it('renders overflow counter badge when tag badges exceed row width', () => {
    hostComponent.node = {
      ...mockNode,
      providedTags: [
        {
          $typeName: 'api.v1.ProvidedTagInfo',
          tag: 'tag-alpha-1-with-long-name-to-overflow',
          outputType: 'string',
          priority: 0,
        },
        {
          $typeName: 'api.v1.ProvidedTagInfo',
          tag: 'tag-beta-2-with-long-name-to-overflow',
          outputType: 'string',
          priority: 0,
        },
        {
          $typeName: 'api.v1.ProvidedTagInfo',
          tag: 'tag-gamma-3-with-long-name-to-overflow',
          outputType: 'string',
          priority: 0,
        },
        {
          $typeName: 'api.v1.ProvidedTagInfo',
          tag: 'tag-delta-4-with-long-name-to-overflow',
          outputType: 'string',
          priority: 0,
        },
        {
          $typeName: 'api.v1.ProvidedTagInfo',
          tag: 'tag-epsilon-5-with-long-name-to-overflow',
          outputType: 'string',
          priority: 0,
        },
      ] as unknown as readonly ProvidedTagInfo[],
    };
    fixture.detectChanges();

    const root = fixture.nativeElement as HTMLElement;
    const tagBadges = root.querySelectorAll('.tag-badge');
    expect(tagBadges.length).toBeGreaterThan(0);

    const lastBadge = tagBadges[tagBadges.length - 1];
    const lastText = lastBadge
      .querySelector('.badge-text')
      ?.textContent?.trim();
    expect(lastText).toMatch(/^\+\d+$/);
    const lastTooltip = lastBadge.querySelector('title')?.textContent?.trim();
    expect(lastTooltip).toMatch(/\d+ more tag\(s\)/);
  });

  it('renders svg title with description when description label is present', () => {
    hostComponent.node = {
      ...mockNode,
      labels: {
        [TASK_DESCRIPTION_LABEL_KEY]: 'Parses logs into timeline events.',
      },
    };
    fixture.detectChanges();

    const titleEl = fixture.nativeElement.querySelector('title');
    expect(titleEl.textContent.trim()).toBe(
      'khi.google.com/test-task - Parses logs into timeline events.\nOutput: string',
    );
  });

  const runPhaseCases: readonly {
    readonly name: string;
    readonly runPhase: DagNodeRunPhase;
    readonly expectedAttribute: string | null;
  }[] = [
    {
      name: 'omits the attribute when the node is not tied to a run',
      runPhase: DagNodeRunPhase.NONE,
      expectedAttribute: null,
    },
    {
      name: 'marks waiting tasks',
      runPhase: DagNodeRunPhase.WAITING,
      expectedAttribute: 'WAITING',
    },
    {
      name: 'marks running tasks',
      runPhase: DagNodeRunPhase.RUNNING,
      expectedAttribute: 'RUNNING',
    },
    {
      name: 'marks finished tasks',
      runPhase: DagNodeRunPhase.DONE,
      expectedAttribute: 'DONE',
    },
    {
      name: 'marks failed tasks',
      runPhase: DagNodeRunPhase.ERROR,
      expectedAttribute: 'ERROR',
    },
  ];

  runPhaseCases.forEach((testCase) => {
    it(`exposes the run phase as a host attribute: ${testCase.name}`, () => {
      hostComponent.node = { ...mockNode, runPhase: testCase.runPhase };
      fixture.detectChanges();

      const group = fixture.nativeElement.querySelector('.dag-node-group');
      expect(group.getAttribute('data-run-phase')).toBe(
        testCase.expectedAttribute,
      );
    });
  });

  it('renders the formatted run duration when the duration is known', () => {
    hostComponent.node = {
      ...mockNode,
      runPhase: DagNodeRunPhase.DONE,
      runDurationMs: 4321,
    };
    fixture.detectChanges();

    const durationText = fixture.nativeElement.querySelector(
      '.node-run-duration-text',
    );
    expect(durationText.textContent.trim()).toBe('4.3s');
    expect(durationText.getAttribute('x')).toBe('406');
  });

  it('omits the run duration when the duration is unknown', () => {
    hostComponent.node = {
      ...mockNode,
      runPhase: DagNodeRunPhase.WAITING,
      runDurationMs: 0,
    };
    fixture.detectChanges();

    expect(
      fixture.nativeElement.querySelector('.node-run-duration-text'),
    ).toBeNull();
  });

  it('truncates output type earlier when run duration is displayed to avoid overlap', () => {
    const fiftyCharOutputType =
      'pkg/model/history/resourceinfo/TimelineResourceMap';
    hostComponent.node = {
      ...mockNode,
      outputType: fiftyCharOutputType,
      runPhase: DagNodeRunPhase.NONE,
      runDurationMs: 0,
    };
    fixture.detectChanges();

    const outputTextWithoutDuration = fixture.nativeElement
      .querySelector('.node-output-type-text')
      .textContent.trim();
    expect(outputTextWithoutDuration).toBe(fiftyCharOutputType);

    hostComponent.node = {
      ...mockNode,
      outputType: fiftyCharOutputType,
      runPhase: DagNodeRunPhase.DONE,
      runDurationMs: 12345,
    };
    fixture.detectChanges();

    const outputTextWithDuration = fixture.nativeElement
      .querySelector('.node-output-type-text')
      .textContent.trim();
    expect(outputTextWithDuration.endsWith('...')).toBeTrue();
    expect(outputTextWithDuration.length).toBeLessThan(
      fiftyCharOutputType.length,
    );
  });
});
