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
  renderNode,
  YamlLine,
  formatValue,
  getRenderSegments,
  postRender,
} from './diff-renderer';
import { MergeNode, ValueType } from './diff-util';
import { DiffStatus } from './lcs';

describe('diff-renderer', () => {
  function createTestNode(overrides: Partial<MergeNode>): MergeNode {
    return {
      key: '',
      path: 'test',
      status: DiffStatus.Unchanged,
      valueType: ValueType.String,
      ...overrides,
    };
  }

  describe('renderNode', () => {
    let result: YamlLine[];

    beforeEach(() => {
      result = [];
    });

    describe('Scalar Nodes', () => {
      it('should render an unchanged scalar', () => {
        const node = createTestNode({
          key: 'foo',
          newValue: 'bar',
          valueType: ValueType.String,
        });
        renderNode(node, 0, false, result);
        expect(result.length).toBe(1);
        expect(result[0]).toEqual(
          jasmine.objectContaining({
            text: 'foo: bar',
            key: 'foo',
            valueText: 'bar',
            valueType: ValueType.String,
            diffStatus: DiffStatus.Unchanged,
            path: 'test',
          }),
        );
      });

      it('should render an added scalar', () => {
        const node = createTestNode({
          key: 'foo',
          status: DiffStatus.Added,
          newValue: 'bar',
          valueType: ValueType.String,
        });
        renderNode(node, 0, false, result);
        expect(result.length).toBe(1);
        expect(result[0].diffStatus).toBe(DiffStatus.Added);
        expect(result[0].valueText).toBe('bar');
      });

      it('should render a deleted scalar', () => {
        const node = createTestNode({
          key: 'foo',
          status: DiffStatus.Deleted,
          oldValue: 'bar',
          valueType: ValueType.String,
        });
        renderNode(node, 0, false, result);
        expect(result.length).toBe(1);
        expect(result[0].diffStatus).toBe(DiffStatus.Deleted);
        expect(result[0].valueText).toBe('bar');
      });

      it('should render a modified scalar with character-level diffs', () => {
        const node = createTestNode({
          key: 'foo',
          status: DiffStatus.Modified,
          oldValue: 'cat',
          newValue: 'cut',
          valueType: ValueType.String,
        });
        renderNode(node, 0, false, result);
        expect(result.length).toBe(2);

        expect(result[0].diffStatus).toBe(DiffStatus.Deleted);
        expect(result[0].valueText).toBe('cat');
        expect(result[0].valueSegments).toEqual([
          { text: 'c', diffStatus: DiffStatus.Unchanged },
          { text: 'a', diffStatus: DiffStatus.Deleted },
          { text: 't', diffStatus: DiffStatus.Unchanged },
        ]);

        expect(result[1].diffStatus).toBe(DiffStatus.Added);
        expect(result[1].valueText).toBe('cut');
        expect(result[1].valueSegments).toEqual([
          { text: 'c', diffStatus: DiffStatus.Unchanged },
          { text: 'u', diffStatus: DiffStatus.Added },
          { text: 't', diffStatus: DiffStatus.Unchanged },
        ]);
      });

      it('should render a modified scalar with full highlight if they are not comparable', () => {
        const node = createTestNode({
          key: 'foo',
          status: DiffStatus.Modified,
          oldValue: 123,
          newValue: 'bar',
          valueType: ValueType.String, // The helper will detect actual type
        });
        renderNode(node, 0, false, result);
        expect(result.length).toBe(2);

        expect(result[0].diffStatus).toBe(DiffStatus.Deleted);
        expect(result[0].valueSegments).toEqual([
          { text: '123', diffStatus: DiffStatus.Deleted },
        ]);

        expect(result[1].diffStatus).toBe(DiffStatus.Added);
        expect(result[1].valueSegments).toEqual([
          { text: 'bar', diffStatus: DiffStatus.Added },
        ]);
      });
    });

    describe('Move Operations', () => {
      it('should render MovedIn node and compute moveId and movedFrom', () => {
        const node = createTestNode({
          path: 'metadata.name',
          status: DiffStatus.MovedIn,
          movedFrom: 'oldName',
          newValue: 'my-pod',
        });
        renderNode(node, 0, false, result);
        expect(result.length).toBe(1);
        expect(result[0].diffStatus).toBe(DiffStatus.MovedIn);
        expect(result[0].movedFrom).toBe('oldName');
        expect(result[0].moveId).toBe('metadata.oldName->metadata.name');
      });

      it('should render MovedOut node and compute moveId and movedTo', () => {
        const node = createTestNode({
          path: 'metadata.name',
          status: DiffStatus.MovedOut,
          movedTo: 'newName',
          oldValue: 'my-pod',
        });
        renderNode(node, 0, false, result);
        expect(result.length).toBe(1);
        expect(result[0].diffStatus).toBe(DiffStatus.MovedOut);
        expect(result[0].movedTo).toBe('newName');
        expect(result[0].moveId).toBe('metadata.name->metadata.newName');
      });

      it('should propagate moveId and status to children', () => {
        const child = createTestNode({
          path: 'metadata.labels.app',
          status: DiffStatus.Unchanged,
          newValue: 'khi',
        });
        const parent = createTestNode({
          key: 'labels',
          path: 'metadata.labels',
          status: DiffStatus.MovedIn,
          movedFrom: 'oldLabels',
          valueType: ValueType.Object,
          children: [child],
        });

        renderNode(parent, 0, false, result);
        // Renders:
        // 1. labels:
        // 2.   app: khi
        expect(result.length).toBe(2);
        expect(result[0].diffStatus).toBe(DiffStatus.MovedIn);
        expect(result[0].moveId).toBe('metadata.oldLabels->metadata.labels');

        expect(result[1].diffStatus).toBe(DiffStatus.MovedIn);
        expect(result[1].moveId).toBe('metadata.oldLabels->metadata.labels');
      });

      it('should render MovedOut node with Modified status using character-level diffs', () => {
        const node = createTestNode({
          path: 'metadata.name',
          status: DiffStatus.Modified,
          oldValue: 'cat',
          newValue: 'cut',
        });
        renderNode(node, 0, false, result, undefined, DiffStatus.MovedOut);
        expect(result.length).toBe(1);
        expect(result[0].diffStatus).toBe(DiffStatus.MovedOut);
        expect(result[0].valueSegments).toEqual([
          { text: 'c', diffStatus: DiffStatus.Unchanged },
          { text: 'a', diffStatus: DiffStatus.Deleted },
          { text: 't', diffStatus: DiffStatus.Unchanged },
        ]);
      });

      it('should render MovedIn node with Modified status using character-level diffs', () => {
        const node = createTestNode({
          path: 'metadata.name',
          status: DiffStatus.Modified,
          oldValue: 'cat',
          newValue: 'cut',
        });
        renderNode(node, 0, false, result, undefined, DiffStatus.MovedIn);
        expect(result.length).toBe(1);
        expect(result[0].diffStatus).toBe(DiffStatus.MovedIn);
        expect(result[0].valueSegments).toEqual([
          { text: 'c', diffStatus: DiffStatus.Unchanged },
          { text: 'u', diffStatus: DiffStatus.Added },
          { text: 't', diffStatus: DiffStatus.Unchanged },
        ]);
      });

      it('should set movedTo on composite line if status is MovedOut', () => {
        const child = createTestNode({
          key: 'name',
          newValue: 'pod-1',
          path: 'metadata.name',
        });
        const node = createTestNode({
          key: 'metadata',
          path: 'metadata',
          status: DiffStatus.MovedOut,
          movedTo: 'newMetadata',
          valueType: ValueType.Object,
          children: [child],
        });
        renderNode(node, 0, false, result);
        expect(result.length).toBe(2);
        expect(result[0].text).toBe('metadata:');
        expect(result[0].diffStatus).toBe(DiffStatus.MovedOut);
        expect(result[0].movedTo).toBe('newMetadata');
      });
    });

    describe('shouldHighlightEntireValue triggering', () => {
      it('should render a modified scalar with full highlight if shouldHighlightEntireValue returns true', () => {
        const node = createTestNode({
          key: 'foo',
          status: DiffStatus.Modified,
          oldValue: true,
          newValue: 'false',
          valueType: ValueType.String,
        });
        renderNode(node, 0, false, result);
        expect(result.length).toBe(2);

        expect(result[0].diffStatus).toBe(DiffStatus.Deleted);
        expect(result[0].valueSegments).toEqual([
          { text: 'true', diffStatus: DiffStatus.Deleted },
        ]);

        expect(result[1].diffStatus).toBe(DiffStatus.Added);
        expect(result[1].valueSegments).toEqual([
          { text: 'false', diffStatus: DiffStatus.Added },
        ]);
      });
    });

    describe('Empty Collections', () => {
      it('should render an empty object as {}', () => {
        const node = createTestNode({
          key: 'emptyObj',
          valueType: ValueType.Object,
          newValue: {},
          children: [],
        });
        renderNode(node, 0, false, result);
        expect(result.length).toBe(1);
        expect(result[0].valueText).toBe('{}');
        expect(result[0].text).toBe('emptyObj: {}');
      });

      it('should render an empty array as []', () => {
        const node = createTestNode({
          key: 'emptyArr',
          valueType: ValueType.Array,
          newValue: [],
          children: [],
        });
        renderNode(node, 0, false, result);
        expect(result.length).toBe(1);
        expect(result[0].valueText).toBe('[]');
        expect(result[0].text).toBe('emptyArr: []');
      });
    });

    describe('Composite Nodes', () => {
      it('should render nested objects with correct indentation', () => {
        const grandChild = createTestNode({
          key: 'foo',
          newValue: 'bar',
          path: 'a.b.foo',
        });
        const child = createTestNode({
          key: 'b',
          valueType: ValueType.Object,
          path: 'a.b',
          children: [grandChild],
        });
        const parent = createTestNode({
          key: 'a',
          valueType: ValueType.Object,
          path: 'a',
          children: [child],
        });

        renderNode(parent, 0, false, result);
        expect(result.length).toBe(3);
        expect(result[0].text).toBe('a:');
        expect(result[0].indent).toBe(0);
        expect(result[1].text).toBe('  b:');
        expect(result[1].indent).toBe(2);
        expect(result[2].text).toBe('    foo: bar');
        expect(result[2].indent).toBe(4);
      });

      it('should render arrays of scalars with - prefix', () => {
        const item1 = createTestNode({
          newValue: 'val1',
          path: 'list.[0]',
        });
        const item2 = createTestNode({
          newValue: 'val2',
          path: 'list.[1]',
        });
        const parent = createTestNode({
          key: 'list',
          valueType: ValueType.Array,
          path: 'list',
          children: [item1, item2],
        });

        renderNode(parent, 0, false, result);
        expect(result.length).toBe(3);
        expect(result[0].text).toBe('list:');
        expect(result[1].text).toBe('- val1');
        expect(result[1].indent).toBe(0);
        expect(result[1].isArrayElementStart).toBe(true);
        expect(result[2].text).toBe('- val2');
        expect(result[2].indent).toBe(0);
        expect(result[2].isArrayElementStart).toBe(true);
      });

      it('should render arrays of objects correctly', () => {
        const prop = createTestNode({
          key: 'name',
          newValue: 'pod-1',
          path: 'list.[0].name',
        });
        const obj = createTestNode({
          valueType: ValueType.Object,
          path: 'list.[0]',
          children: [prop],
        });
        const parent = createTestNode({
          key: 'list',
          valueType: ValueType.Array,
          path: 'list',
          children: [obj],
        });

        renderNode(parent, 0, false, result);
        // Should render:
        // list:
        // - name: pod-1
        expect(result.length).toBe(2);
        expect(result[0].text).toBe('list:');
        expect(result[1].text).toBe('- name: pod-1');
        expect(result[1].indent).toBe(0);
        expect(result[1].isArrayElementStart).toBe(true);
      });
    });

    describe('Multiline Strings', () => {
      it('should render multiline string with header | and body lines', () => {
        const node = createTestNode({
          key: 'logs',
          newValue: 'line1\nline2\nline3\n',
          valueType: ValueType.String,
        });

        renderNode(node, 0, false, result);
        expect(result.length).toBe(4);
        expect(result[0].text).toBe('logs: |');
        expect(result[0].valueText).toBe('|');

        expect(result[1].text).toBe('  line1');
        expect(result[1].indent).toBe(2);
        expect(result[2].text).toBe('  line2');
        expect(result[3].text).toBe('  line3');
      });

      it('should handle multiline strings without trailing newline', () => {
        const node = createTestNode({
          key: 'logs',
          newValue: 'line1\nline2',
          valueType: ValueType.String,
        });

        renderNode(node, 0, false, result);
        expect(result.length).toBe(3);
        expect(result[0].text).toBe('logs: |');
        expect(result[1].text).toBe('  line1');
        expect(result[2].text).toBe('  line2');
      });
    });

    describe('Type Modifications', () => {
      it('should render scalar to object modification', () => {
        // e.g. foo: "bar" -> foo: { x: 1 }
        const child = createTestNode({
          key: 'x',
          newValue: 1,
          status: DiffStatus.Added,
          valueType: ValueType.Number,
          path: 'foo.x',
        });
        const node = createTestNode({
          key: 'foo',
          status: DiffStatus.Modified,
          valueType: ValueType.Object,
          oldValueType: ValueType.String,
          oldValue: 'bar',
          children: [child],
          path: 'foo',
        });

        renderNode(node, 0, false, result);
        // Expecting:
        // - foo: bar
        // + foo:
        // +   x: 1
        expect(result.length).toBe(3);
        expect(result[0]).toEqual(
          jasmine.objectContaining({
            text: 'foo: bar',
            diffStatus: DiffStatus.Deleted,
          }),
        );
        expect(result[1]).toEqual(
          jasmine.objectContaining({
            text: 'foo:',
            diffStatus: DiffStatus.Added,
          }),
        );
        expect(result[2]).toEqual(
          jasmine.objectContaining({
            text: '  x: 1',
            diffStatus: DiffStatus.Added,
          }),
        );
      });

      it('should render object to scalar modification', () => {
        // e.g. foo: { x: 1 } -> foo: "bar"
        const child = createTestNode({
          key: 'x',
          oldValue: 1,
          status: DiffStatus.Deleted,
          valueType: ValueType.Number,
          path: 'foo.x',
        });
        const node = createTestNode({
          key: 'foo',
          status: DiffStatus.Modified,
          valueType: ValueType.String, // New type is String
          oldValueType: ValueType.Object,
          oldValue: undefined, // Children represent the old object
          newValue: 'bar',
          children: [child],
          path: 'foo',
        });

        renderNode(node, 0, false, result);
        // Expecting:
        // - foo:
        // -   x: 1
        // + foo: bar
        expect(result.length).toBe(3);
        expect(result[0]).toEqual(
          jasmine.objectContaining({
            text: 'foo:',
            diffStatus: DiffStatus.Deleted,
          }),
        );
        expect(result[1]).toEqual(
          jasmine.objectContaining({
            text: '  x: 1',
            diffStatus: DiffStatus.Deleted,
          }),
        );
        expect(result[2]).toEqual(
          jasmine.objectContaining({
            text: 'foo: bar',
            diffStatus: DiffStatus.Added,
          }),
        );
      });

      it('should render scalar to array modification', () => {
        // e.g. foo: "bar" -> foo: [1]
        const child = createTestNode({
          newValue: 1,
          status: DiffStatus.Added,
          valueType: ValueType.Number,
          path: 'foo.[0]',
        });
        const node = createTestNode({
          key: 'foo',
          status: DiffStatus.Modified,
          valueType: ValueType.Array,
          oldValueType: ValueType.String,
          oldValue: 'bar',
          children: [child],
          path: 'foo',
        });

        renderNode(node, 0, false, result);
        // Expecting:
        // - foo: bar
        // + foo:
        // + - 1
        expect(result.length).toBe(3);
        expect(result[0]).toEqual(
          jasmine.objectContaining({
            text: 'foo: bar',
            diffStatus: DiffStatus.Deleted,
          }),
        );
        expect(result[1]).toEqual(
          jasmine.objectContaining({
            text: 'foo:',
            diffStatus: DiffStatus.Added,
          }),
        );
        expect(result[2]).toEqual(
          jasmine.objectContaining({
            text: '- 1',
            diffStatus: DiffStatus.Added,
          }),
        );
      });

      it('should render array to scalar modification', () => {
        // e.g. foo: [1] -> foo: "bar"
        const child = createTestNode({
          oldValue: 1,
          status: DiffStatus.Deleted,
          valueType: ValueType.Number,
          path: 'foo.[0]',
        });
        const node = createTestNode({
          key: 'foo',
          status: DiffStatus.Modified,
          valueType: ValueType.String,
          oldValueType: ValueType.Array,
          newValue: 'bar',
          children: [child],
          path: 'foo',
        });

        renderNode(node, 0, false, result);
        // Expecting:
        // - foo:
        // - - 1
        // + foo: bar
        expect(result.length).toBe(3);
        expect(result[0]).toEqual(
          jasmine.objectContaining({
            text: 'foo:',
            diffStatus: DiffStatus.Deleted,
          }),
        );
        expect(result[1]).toEqual(
          jasmine.objectContaining({
            text: '- 1',
            diffStatus: DiffStatus.Deleted,
          }),
        );
        expect(result[2]).toEqual(
          jasmine.objectContaining({
            text: 'foo: bar',
            diffStatus: DiffStatus.Added,
          }),
        );
      });

      it('should render array to object modification', () => {
        // e.g. foo: [1] -> foo: { x: 2 }
        const oldChild = createTestNode({
          oldValue: 1,
          status: DiffStatus.Deleted,
          valueType: ValueType.Number,
          path: 'foo.[0]',
        });
        const newChild = createTestNode({
          key: 'x',
          newValue: 2,
          status: DiffStatus.Added,
          valueType: ValueType.Number,
          path: 'foo.x',
        });
        const node = createTestNode({
          key: 'foo',
          status: DiffStatus.Modified,
          valueType: ValueType.Object, // New type
          oldValueType: ValueType.Array,
          children: [oldChild, newChild],
          path: 'foo',
        });

        renderNode(node, 0, false, result);
        // Expecting:
        // - foo:
        // - - 1
        // + foo:
        // +   x: 2
        expect(result.length).toBe(4);
        expect(result[0]).toEqual(
          jasmine.objectContaining({
            text: 'foo:',
            diffStatus: DiffStatus.Deleted,
          }),
        );
        expect(result[1]).toEqual(
          jasmine.objectContaining({
            text: '- 1',
            diffStatus: DiffStatus.Deleted,
          }),
        );
        expect(result[2]).toEqual(
          jasmine.objectContaining({
            text: 'foo:',
            diffStatus: DiffStatus.Added,
          }),
        );
        expect(result[3]).toEqual(
          jasmine.objectContaining({
            text: '  x: 2',
            diffStatus: DiffStatus.Added,
          }),
        );
      });

      it('should render object to array modification', () => {
        // e.g. foo: { x: 2 } -> foo: [1]
        const oldChild = createTestNode({
          key: 'x',
          oldValue: 2,
          status: DiffStatus.Deleted,
          valueType: ValueType.Number,
          path: 'foo.x',
        });
        const newChild = createTestNode({
          newValue: 1,
          status: DiffStatus.Added,
          valueType: ValueType.Number,
          path: 'foo.[0]',
        });
        const node = createTestNode({
          key: 'foo',
          status: DiffStatus.Modified,
          valueType: ValueType.Array, // New type
          oldValueType: ValueType.Object,
          children: [oldChild, newChild],
          path: 'foo',
        });

        renderNode(node, 0, false, result);
        // Expecting:
        // - foo:
        // -   x: 2
        // + foo:
        // + - 1
        expect(result.length).toBe(4);
        expect(result[0]).toEqual(
          jasmine.objectContaining({
            text: 'foo:',
            diffStatus: DiffStatus.Deleted,
          }),
        );
        expect(result[1]).toEqual(
          jasmine.objectContaining({
            text: '  x: 2',
            diffStatus: DiffStatus.Deleted,
          }),
        );
        expect(result[2]).toEqual(
          jasmine.objectContaining({
            text: 'foo:',
            diffStatus: DiffStatus.Added,
          }),
        );
        expect(result[3]).toEqual(
          jasmine.objectContaining({
            text: '- 1',
            diffStatus: DiffStatus.Added,
          }),
        );
      });
    });
  });

  describe('formatValue', () => {
    it('should format null to "null"', () => {
      expect(formatValue(null)).toBe('null');
    });

    it('should format boolean and number to string', () => {
      expect(formatValue(true)).toBe('true');
      expect(formatValue(123.45)).toBe('123.45');
    });

    it('should wrap string in quotes if it contains spaces or colons', () => {
      expect(formatValue('hello world')).toBe('"hello world"');
      expect(formatValue('foo:bar')).toBe('"foo:bar"');
      expect(formatValue('clean')).toBe('clean');
    });

    it('should format empty array and object', () => {
      expect(formatValue([])).toBe('[]');
      expect(formatValue({})).toBe('{}');
    });
  });

  describe('getRenderSegments', () => {
    it('should return simple segments when query is empty', () => {
      const line: YamlLine = {
        text: 'foo: bar',
        indent: 0,
        key: 'foo',
        valueText: 'bar',
        valueType: ValueType.String,
        valueSegments: [{ text: 'bar', diffStatus: DiffStatus.Unchanged }],
        diffStatus: DiffStatus.Unchanged,
        path: 'foo',
      };

      const segments = getRenderSegments(line, '', 0, null);
      expect(segments.length).toBe(3); // 'foo', ': ', 'bar'
      expect(segments[0]).toEqual(
        jasmine.objectContaining({
          text: 'foo',
          diffStatus: DiffStatus.Unchanged,
          isMatch: false,
          isKey: true,
        }),
      );
      expect(segments[1]).toEqual(
        jasmine.objectContaining({
          text: ': ',
          diffStatus: DiffStatus.Unchanged,
          isMatch: false,
          isColon: true,
        }),
      );
      expect(segments[2]).toEqual(
        jasmine.objectContaining({
          text: 'bar',
          diffStatus: DiffStatus.Unchanged,
          isMatch: false,
          isValue: true,
        }),
      );
    });

    it('should split segments when query matches', () => {
      const line: YamlLine = {
        text: 'foo: bar',
        indent: 0,
        key: 'foo',
        valueText: 'bar',
        valueType: ValueType.String,
        valueSegments: [{ text: 'bar', diffStatus: DiffStatus.Unchanged }],
        diffStatus: DiffStatus.Unchanged,
        path: 'foo',
      };

      const segments = getRenderSegments(line, 'ar', 0, null);
      // 'bar' should be split into 'b' and 'ar'
      expect(segments.length).toBe(4);
      expect(segments[2]).toEqual(
        jasmine.objectContaining({
          text: 'b',
          diffStatus: DiffStatus.Unchanged,
          isMatch: false,
          isValue: true,
        }),
      );
      expect(segments[3]).toEqual(
        jasmine.objectContaining({
          text: 'ar',
          diffStatus: DiffStatus.Unchanged,
          isMatch: true,
          isValue: true,
        }),
      );
    });
  });

  describe('postRender', () => {
    it('should identify moved block boundaries and set moveBlockStart/moveBlockEnd', () => {
      const lines: YamlLine[] = [
        {
          text: 'unchanged: 1',
          indent: 0,
          valueText: '1',
          valueType: ValueType.Number,
          diffStatus: DiffStatus.Unchanged,
          path: 'unchanged',
        },
        {
          text: '  moved1: a',
          indent: 2,
          valueText: 'a',
          valueType: ValueType.String,
          diffStatus: DiffStatus.MovedIn,
          movedFrom: 'old.block1[0]',
          path: 'new.block1[0]',
        },
        {
          text: '  moved2: b',
          indent: 2,
          valueText: 'b',
          valueType: ValueType.String,
          diffStatus: DiffStatus.MovedIn,
          movedFrom: 'old.block1[0]',
          path: 'new.block1[0].val',
        },
        {
          text: '  moved3: c',
          indent: 2,
          valueText: 'c',
          valueType: ValueType.String,
          diffStatus: DiffStatus.MovedIn,
          movedFrom: 'old.block2[0]', // Different block
          path: 'new.block2[0]',
        },
        {
          text: 'unchanged: 2',
          indent: 0,
          valueText: '2',
          valueType: ValueType.Number,
          diffStatus: DiffStatus.Unchanged,
          path: 'unchanged2',
        },
      ];

      postRender(lines);

      expect(lines[0].moveBlockStart).toBeUndefined();
      expect(lines[0].moveBlockEnd).toBeUndefined();

      // First block (lines[1] to lines[2])
      expect(lines[1].moveBlockStart).toBeTrue();
      expect(lines[1].moveBlockEnd).toBeUndefined();
      expect(lines[2].moveBlockStart).toBeUndefined();
      expect(lines[2].moveBlockEnd).toBeTrue();

      // Second block (lines[3] - single line)
      expect(lines[3].moveBlockStart).toBeTrue();
      expect(lines[3].moveBlockEnd).toBeTrue();

      expect(lines[4].moveBlockStart).toBeUndefined();
      expect(lines[4].moveBlockEnd).toBeUndefined();
    });

    it('should calculate contentWidthCh for moved blocks including indents and value lengths', () => {
      const lines: YamlLine[] = [
        {
          text: '  short: val',
          indent: 2,
          key: 'short',
          valueText: 'val',
          valueType: ValueType.String,
          diffStatus: DiffStatus.MovedOut,
          movedTo: 'dest[0]',
          path: 'short',
        },
        {
          text: '    veryLongKey: ""',
          indent: 4,
          key: 'veryLongKey',
          valueText: '""',
          valueType: ValueType.String,
          diffStatus: DiffStatus.MovedOut,
          movedTo: 'dest[0]',
          path: 'veryLongKey',
        },
      ];

      // Calculations:
      // minIndent = 2
      // Line 0: relativeIndent = 0. contentLength = 5 (short) + 2 (: ) + 3 (val) = 10. visualEnd = 10.
      // Line 1: relativeIndent = 2. contentLength = 11 (veryLongKey) + 2 (: ) + 2 ("") = 15. visualEnd = 17.
      // maxVisualEnd = 17
      // Line 0 contentWidthCh = 17 - 0 + 1 = 18
      // Line 1 contentWidthCh = 17 - 2 + 1 = 16

      postRender(lines);

      expect(lines[0].contentWidthCh).toBe(18);
      expect(lines[1].contentWidthCh).toBe(16);
    });

    it('should account for array element prefix in contentWidthCh calculation', () => {
      const lines: YamlLine[] = [
        {
          text: '  - key: val',
          indent: 4,
          key: 'key',
          valueText: 'val',
          valueType: ValueType.String,
          diffStatus: DiffStatus.MovedIn,
          movedFrom: 'src[0]',
          path: 'arr[0]',
          isArrayElementStart: true,
        },
      ];

      // Calculations:
      // minIndent = 4
      // Line 0: relativeIndent = 0. isArrayElementStart is true -> relativeIndent becomes 2.
      // contentLength = 3 (key) + 2 (: ) + 3 (val) = 8.
      // visualEnd = 2 + 8 = 10.
      // maxVisualEnd = 10.
      // Line 0 contentWidthCh = 10 - 2 + 1 = 9.

      postRender(lines);

      expect(lines[0].contentWidthCh).toBe(9);
    });
  });
});
