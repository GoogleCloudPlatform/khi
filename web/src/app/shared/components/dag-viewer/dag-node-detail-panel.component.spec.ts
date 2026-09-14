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

import { create } from '@bufbuild/protobuf';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ProvidedTagInfoSchema } from 'src/app/generated/api/v1/inspection_task_graph_pb';
import { DagNodeDetailPanelComponent } from 'src/app/shared/components/dag-viewer/dag-node-detail-panel.component';
import {
  DagViewerNode,
  TASK_DESCRIPTION_LABEL_KEY,
} from 'src/app/shared/components/dag-viewer/dag-viewer.model';

describe('DagNodeDetailPanelComponent', () => {
  let component: DagNodeDetailPanelComponent;
  let fixture: ComponentFixture<DagNodeDetailPanelComponent>;

  const mockNode: DagViewerNode = {
    id: 'khi.k8s.pod-parser#01',
    referenceId: 'khi.k8s.pod-parser',
    isFeature: false,
    isFormTask: false,
    topologicalOrder: 1,
    priority: 120,
    labels: {
      'khi.google.com/task-type': 'parser',
    },
    outputType: 'k8s.PodList',
    providedTags: [
      create(ProvidedTagInfoSchema, {
        tag: 'parsed-pods',
        priority: 0,
        outputType: 'k8s.PodList',
      }),
    ],
  };

  const mockUpstream: DagViewerNode[] = [
    {
      id: 'khi.source.reader#00',
      referenceId: 'khi.source.reader',
      isFeature: false,
      isFormTask: true,
      topologicalOrder: 0,
      priority: 100,
      labels: {},
      outputType: 'io.Reader',
      providedTags: [],
    },
  ];

  const mockDownstream: DagViewerNode[] = [
    {
      id: 'khi.timeline.builder#02',
      referenceId: 'khi.timeline.builder',
      isFeature: false,
      isFormTask: false,
      topologicalOrder: 2,
      priority: 100,
      labels: {},
      outputType: 'timeline.Timeline',
      providedTags: [],
    },
  ];

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [DagNodeDetailPanelComponent],
    }).compileComponents();

    fixture = TestBed.createComponent(DagNodeDetailPanelComponent);
    component = fixture.componentInstance;
  });

  it('renders node details when node is provided', () => {
    fixture.componentRef.setInput('node', mockNode);
    fixture.componentRef.setInput('upstreamNodes', mockUpstream);
    fixture.componentRef.setInput('downstreamNodes', mockDownstream);
    fixture.detectChanges();

    const title = fixture.nativeElement.querySelector('.panel-title');
    expect(title.textContent.trim()).toBe('khi.k8s.pod-parser');

    const subtitle = fixture.nativeElement.querySelector('.panel-subtitle');
    expect(subtitle.textContent.trim()).toBe('khi.k8s.pod-parser#01');

    const labelKey = fixture.nativeElement.querySelector('.label-key');
    expect(labelKey.textContent.trim()).toBe('khi.google.com/task-type');

    const labelVal = fixture.nativeElement.querySelector('.label-val');
    expect(labelVal.textContent.trim()).toBe('parser');

    const propLabels = Array.from(
      fixture.nativeElement.querySelectorAll('.prop-label'),
    ) as HTMLElement[];
    const propValues = Array.from(
      fixture.nativeElement.querySelectorAll('.prop-value'),
    ) as HTMLElement[];
    const outputTypeIndex = propLabels.findIndex(
      (el) => el.textContent?.trim() === 'Output Type',
    );
    expect(outputTypeIndex).toBeGreaterThanOrEqual(0);
    expect(propValues[outputTypeIndex].textContent?.trim()).toBe('k8s.PodList');

    const tagChips = Array.from(
      fixture.nativeElement.querySelectorAll('.tag-chip'),
    ) as HTMLElement[];
    expect(tagChips.length).toBe(1);
    expect(tagChips[0].querySelector('.tag-name')?.textContent?.trim()).toBe(
      '#parsed-pods',
    );
    expect(
      tagChips[0].querySelector('.tag-output-type')?.textContent?.trim(),
    ).toBe('k8s.PodList');

    const depTypes = (
      Array.from(
        fixture.nativeElement.querySelectorAll('.dep-output-type'),
      ) as HTMLElement[]
    ).map((el) => el.textContent?.trim());
    expect(depTypes).toContain('io.Reader');
    expect(depTypes).toContain('timeline.Timeline');
  });

  it('emits closePanel event when close button is clicked', () => {
    fixture.componentRef.setInput('node', mockNode);
    fixture.detectChanges();

    let closed = false;
    component.closePanel.subscribe(() => {
      closed = true;
    });

    const closeBtn = fixture.nativeElement.querySelector('.close-btn');
    closeBtn.dispatchEvent(new MouseEvent('click'));

    expect(closed).toBeTrue();
  });

  it('emits selectNode when a connected dependency is clicked', () => {
    fixture.componentRef.setInput('node', mockNode);
    fixture.componentRef.setInput('upstreamNodes', mockUpstream);
    fixture.detectChanges();

    let selectedId = '';
    component.selectNode.subscribe((id) => {
      selectedId = id;
    });

    const depItem = fixture.nativeElement.querySelector('.dependency-item');
    depItem.dispatchEvent(new MouseEvent('click'));

    expect(selectedId).toBe('khi.source.reader#00');
  });

  it('renders description section when task has description label', () => {
    const nodeWithDesc: DagViewerNode = {
      ...mockNode,
      labels: {
        ...mockNode.labels,
        [TASK_DESCRIPTION_LABEL_KEY]: 'Parses k8s pod events into graph.',
      },
    };
    fixture.componentRef.setInput('node', nodeWithDesc);
    fixture.detectChanges();

    const descEl = fixture.nativeElement.querySelector(
      '.description-section .description-text',
    );
    expect(descEl).toBeTruthy();
    expect(descEl.textContent.trim()).toBe('Parses k8s pod events into graph.');
  });

  it('does not render description section when task lacks description label', () => {
    fixture.componentRef.setInput('node', mockNode);
    fixture.detectChanges();

    const descEl = fixture.nativeElement.querySelector('.description-section');
    expect(descEl).toBeNull();
  });
});
