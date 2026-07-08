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

import { Meta, StoryObj } from '@storybook/angular';
import { YamlViewerComponent } from './yaml-viewer.component';
import {
  YamlAnnotationProvider,
  YamlFieldAnnotation,
} from 'src/app/shared/components/yaml-viewer/yaml-annotation';
import { ManagedFieldTooltipComponent } from 'src/app/shared/components/yaml-viewer/components/managed-field-tooltip.component';

class MockAnnotationProvider implements YamlAnnotationProvider {
  constructor(private readonly tooltipMap: Record<string, string>) {}
  getAnnotations(): YamlFieldAnnotation[] {
    const annotations: YamlFieldAnnotation[] = [];
    for (const [pathStr] of Object.entries(this.tooltipMap)) {
      const path = pathStr.split('.');
      annotations.push({
        path: path,
        component: ManagedFieldTooltipComponent,
        inputs: {
          manager: 'mock-manager',
          operation: 'Update',
          time: '2026-06-29T11:00:00Z',
        },
      });
    }
    return annotations;
  }
}

const leftYamlMock = `apiVersion: v1
kind: Pod
metadata:
  name: my-pod
  namespace: default
  annotations:
    description: |
      This is a pod description.
      It has multiple lines.
      Old content here.
spec:
  replicas: 1
  containers:
    - name: nginx
      image: nginx:1.14.2
status:
  conditions:
    - type: Ready
      status: "True"
      lastTransitionTime: "2026-06-29T11:00:00Z"
    - type: PodScheduled
      status: "True"
      lastTransitionTime: "2026-06-29T10:59:00Z"`;

const rightYamlMock = `apiVersion: v1
kind: Pod
metadata:
  name: new-pod
  namespace: default
  annotations:
    description: |
      This is a pod description.
      It has multiple lines.
      New content here.
spec:
  replicas: 3
  containers:
    - name: nginx
      image: nginx:1.16.0
    - name: sidecar
      image: busybox:latest
      commands:
        - /bin/sh
        - -c
        - whoami
      args: []
  securityContext: {}
status:
  conditions:
    - type: PodScheduled
      status: "True"
      lastTransitionTime: "2026-06-29T10:59:00Z"
    - type: Ready
      status: "True"
      lastTransitionTime: "2026-06-29T11:15:00Z"`;

const meta: Meta<YamlViewerComponent> = {
  title: 'Shared/YamlViewer',
  component: YamlViewerComponent,
  tags: ['autodocs'],
  argTypes: {
    leftYaml: {
      control: 'text',
      description: 'The original YAML content for diffing.',
    },
    rightYaml: {
      control: 'text',
      description: 'The updated YAML content.',
    },
    searchQuery: {
      control: 'text',
      description: 'Search query to highlight.',
    },
  },
  args: {
    leftYaml: null,
    rightYaml: rightYamlMock,
    annotationProviders: [],
    searchQuery: '',
  },
};

export default meta;
type Story = StoryObj<YamlViewerComponent>;

/**
 * Preview Mode: Displays the updated YAML in a single, highlighted preview.
 */
export const PreviewMode: Story = {
  args: {
    leftYaml: null,
    rightYaml: rightYamlMock,
  },
};

/**
 * Diff Mode: Shows unified semantic diffs (added/deleted lines) between two YAML documents.
 */
export const DiffMode: Story = {
  args: {
    leftYaml:
      'apiVersion: v1\nkind: Pod\nmetadata:\n  name: my-pod\n  namespace: default\n  annotations:\n    description: |\n      This is a pod description.\n      It has multiple lines.\n      Old content here.\nspec:\n  replicas: 1\n  containers:\n    - name: nginx\n      image: nginx:1.14.2\nstatus:\n  conditions:\n    - type: Ready\n      status: "True"\n      lastTransitionTime: "2026-06-29T11:00:00Z"\n    - type: PodScheduled\n      status: "True"\n      lastTransitionTime: "2026-06-29T10:11:11Z"',
    rightYaml: rightYamlMock,
  },
};

/**
 * Tooltip Demo: Demonstrates binding tooltips to specific JSON paths.
 * Hover over "metadata.name" or "spec.replicas" to check.
 */
export const TooltipsDemo: Story = {
  args: {
    leftYaml:
      'apiVersion: v1\nkind: Pod\nmetadata:\n  name: my-pod\n  namespace: default\n  annotations:\n    description: |\n      This is a pod description.\n      It has multiple lines.\n      Old content here.\nspec:\n  replicas: 1\n  containers:\n    - name: nginx\n      image: nginx:1.14.22',
    rightYaml: rightYamlMock,
    annotationProviders: [
      new MockAnnotationProvider({
        'metadata.name': 'This is the unique name of the Kubernetes resource.',
        'spec.replicas': 'Defines the number of desired pod replicas.',
      }),
    ],
  },
};

/**
 * Search Demo: Highlight matched segments inside the document.
 * "nginx" is set as the active search query.
 */
export const SearchDemo: Story = {
  args: {
    leftYaml: leftYamlMock,
    rightYaml: rightYamlMock,
    searchQuery: '',
  },
};

/**
 * Array Move Demo: Shows how array elements that are moved are highlighted in blue.
 * Hover over the moved element to see the original index.
 */
export const ArrayMoveDemo: Story = {
  args: {
    leftYaml: `items:
  - foo
  - bar
  - baz`,
    rightYaml: `items:
  - bar
  - foo
  - baz`,
  },
};

/**
 * Array Multiple Moves Demo: Shows a complex scenario where multiple object elements
 * within a single array are reordered.
 */
export const ArrayMultipleMovesDemo: Story = {
  args: {
    leftYaml: `items:
  - id: 1
    name: first
  - id: 2
    name: second
  - id: 3
    name: third
  - id: 4
    name: fourth
  - id: 5
    name: fifth
  - id: 6
    name: sixth`,
    rightYaml: `items:
  - id: 5
    name: fifth
  - id: 3
    name: third
  - id: 6
    name: sixth
  - id: 1
    name: first
  - id: 4
    name: fourth
  - id: 2
    name: second`,
  },
};

/**
 * Nested Object Add Demo: Shows how nested objects that are added are highlighted.
 */
export const NestedObjectAddDemo: Story = {
  args: {
    leftYaml: `metadata:
  name: my-pod`,
    rightYaml: `metadata:
  name: my-pod
  labels:
    app: my-app
    env: prod
  annotations:
    description: "New resource"`,
  },
};

/**
 * Nested Object Delete Demo: Shows how nested objects that are deleted are highlighted.
 */
export const NestedObjectDeleteDemo: Story = {
  args: {
    leftYaml: `metadata:
  name: my-pod
  labels:
    app: my-app
    env: prod
  annotations:
    description: "Old resource"`,
    rightYaml: `metadata:
  name: my-pod`,
  },
};

/**
 * Empty Object Demo: Shows how empty objects are rendered.
 */
export const EmptyObjectDemo: Story = {
  args: {
    leftYaml: `metadata:
  name: my-pod
  deletedEmpty: {}
  retainedEmpty: {}`,
    rightYaml: `metadata:
  name: my-pod
  addedEmpty: {}
  retainedEmpty: {}`,
  },
};

/**
 * All Added Demo: Shows a newly created resource where leftYaml is empty and rightYaml is a Pod manifest.
 * The entire document should be highlighted as added (green).
 */
export const AllAddedDemo: Story = {
  args: {
    leftYaml: ' ',
    rightYaml: rightYamlMock,
  },
};
