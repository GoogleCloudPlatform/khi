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

import { ParentRelationship } from '../generated';
import { ResourceTimeline } from '../store/timeline';
import { SelectOnlyDeeperOrEqual } from './timeline-collection-util';

describe('TimelineCollectionUtility', () => {
  it('SelectOnlyDeeperOrEqual', () => {
    const timelines = [
      new ResourceTimeline(
        'test',
        '1',
        [],
        [],
        ParentRelationship.RelationshipChild,
      ),
      new ResourceTimeline(
        'test',
        '1#1',
        [],
        [],
        ParentRelationship.RelationshipChild,
      ),
      new ResourceTimeline(
        'test',
        '1#2',
        [],
        [],
        ParentRelationship.RelationshipChild,
      ),
      new ResourceTimeline(
        'test',
        '1#2#1',
        [],
        [],
        ParentRelationship.RelationshipChild,
      ),
      new ResourceTimeline(
        'test',
        '1#2#2',

        [],
        [],
        ParentRelationship.RelationshipChild,
      ),
      new ResourceTimeline(
        'test',
        '2',
        [],
        [],
        ParentRelationship.RelationshipChild,
      ),
      new ResourceTimeline(
        'test',
        '3',
        [],
        [],
        ParentRelationship.RelationshipChild,
      ),
    ];
    const result = SelectOnlyDeeperOrEqual(timelines, 2);
    expect(result.length).toBe(4);
    expect(result[0].resourcePath).toBe('1');
    expect(result[1].resourcePath).toBe('1#2');
    expect(result[2].resourcePath).toBe('1#2#1');
    expect(result[3].resourcePath).toBe('1#2#2');
  });
});
