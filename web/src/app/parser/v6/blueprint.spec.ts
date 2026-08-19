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
  V6_BLUEPRINT,
  V6ChunkType,
  V6InternPoolAssembler,
  V6LogAssembler,
  V6MetadataAssembler,
  V6StyleAssembler,
  V6TimelineAssembler,
} from 'src/app/parser/v6/blueprint';
import {
  HeaderMetadataSchema,
  MetadataChunkSchema,
  MetadataItemSchema,
  QueryMetadataSchema,
} from 'src/app/generated/khifile/v6/metadata_pb';
import {
  EventSchema,
  RevisionSchema,
  TimelineChunkSchema,
  TimelineItemsSchema,
  TimelineSchema,
} from 'src/app/generated/khifile/v6/timeline_pb';
import {
  HDRColor4Schema,
  IconAtlasSchema,
  SeveritySchema,
  TimelineStyleChunkSchema,
} from 'src/app/generated/khifile/v6/style_pb';
import { InspectionDataBuilder } from 'src/app/parser/core/builder';
import { InternedStructSchema } from 'src/app/generated/khifile/shared_pb';
import {
  InterningPoolChunkSchema,
  InternStringSchema,
  InternFieldPathSetSchema,
} from 'src/app/generated/khifile/v6/intern_pool_pb';
import { LogChunkSchema, LogSchema } from 'src/app/generated/khifile/v6/log_pb';
import { create } from '@bufbuild/protobuf';
import { TimestampSchema } from '@bufbuild/protobuf/wkt';

describe('V6_BLUEPRINT', () => {
  it('should register all required chunk definitions with correct priorities', () => {
    expect(V6_BLUEPRINT.has(V6ChunkType.Metadata)).toBeTrue();
    expect(V6_BLUEPRINT.has(V6ChunkType.InterningPool)).toBeTrue();
    expect(V6_BLUEPRINT.has(V6ChunkType.Log)).toBeTrue();
    expect(V6_BLUEPRINT.has(V6ChunkType.TimelineStyle)).toBeTrue();
    expect(V6_BLUEPRINT.has(V6ChunkType.Timeline)).toBeTrue();

    expect(V6_BLUEPRINT.get(V6ChunkType.Metadata)!.priority).toBe(5);
    expect(V6_BLUEPRINT.get(V6ChunkType.InterningPool)!.priority).toBe(10);
    expect(V6_BLUEPRINT.get(V6ChunkType.TimelineStyle)!.priority).toBe(20);
    expect(V6_BLUEPRINT.get(V6ChunkType.Log)!.priority).toBe(100);
    expect(V6_BLUEPRINT.get(V6ChunkType.Timeline)!.priority).toBe(100);
  });
});

describe('V6InternPoolAssembler', () => {
  it('should ingest strings and assemble them into builder', () => {
    const assembler = new V6InternPoolAssembler();
    const mockChunk = create(InterningPoolChunkSchema, {
      strings: [
        create(InternStringSchema, { id: 1, value: 'foo' }),
        create(InternStringSchema, { id: 2, value: 'bar' }),
      ],
      fieldPathSets: [
        create(InternFieldPathSetSchema, {
          id: 10,
          fieldPathStringIds: [1, 2],
        }),
      ],
      structs: [
        create(InternedStructSchema, {
          id: 100,
          fieldPathSetId: 10,
          values: [],
        }),
      ],
    });

    assembler.ingest(mockChunk);

    const builder = jasmine.createSpyObj<InspectionDataBuilder>(
      'InspectionDataBuilder',
      ['addStrings'],
    );
    assembler.assembleInto(builder);

    expect(builder.addStrings).toHaveBeenCalledWith([
      { id: 1, value: 'foo' },
      { id: 2, value: 'bar' },
    ]);
  });
});

describe('V6LogAssembler', () => {
  it('should ingest logs and assemble them into builder', () => {
    const assembler = new V6LogAssembler();
    const mockChunk = create(LogChunkSchema, {
      logs: [
        create(LogSchema, {
          id: 100,
          ts: create(TimestampSchema, { seconds: 123n, nanos: 456 }),
          logTypeId: 10,
          severityTypeId: 20,
          summaryStringId: 30,
          bodyStructId: 5,
        }),
      ],
    });

    assembler.ingest(mockChunk);

    const builder = jasmine.createSpyObj<InspectionDataBuilder>(
      'InspectionDataBuilder',
      ['addLogs'],
    );
    assembler.assembleInto(builder);

    expect(builder.addLogs).toHaveBeenCalledWith([
      {
        id: 100,
        ts: 123000000456n,
        logTypeId: 10,
        severityTypeId: 20,
        summaryStringId: 30,
        bodyStructId: 5,
      },
    ]);
  });
});

describe('V6StyleAssembler', () => {
  it('should ingest timeline styles and assemble them into builder', () => {
    const assembler = new V6StyleAssembler();
    const mockChunk = create(TimelineStyleChunkSchema, {
      severities: [
        create(SeveritySchema, {
          id: 1,
          label: 'INFO',
          shortLabel: 'I',
          backgroundColor: create(HDRColor4Schema, { r: 1, g: 1, b: 1, a: 1 }),
          foregroundColor: create(HDRColor4Schema, { r: 0, g: 0, b: 0, a: 1 }),
          order: 0,
        }),
      ],
      verbs: [],
      logTypes: [],
      revisionStates: [],
      timelineTypes: [],
    });

    assembler.ingest(mockChunk);

    const builder = jasmine.createSpyObj<InspectionDataBuilder>(
      'InspectionDataBuilder',
      [
        'addSeverities',
        'addVerbs',
        'addLogTypes',
        'addRevisionStates',
        'addTimelineTypes',
      ],
    );
    assembler.assembleInto(builder);

    expect(builder.addSeverities).toHaveBeenCalledWith([
      {
        id: 1,
        label: 'INFO',
        shortLabel: 'I',
        backgroundColor: { r: 1, g: 1, b: 1, a: 1 },
        foregroundColor: { r: 0, g: 0, b: 0, a: 1 },
        order: 0,
      },
    ]);
  });

  it('should correctly extract ArrayBuffer slices for iconAtlas with subarray views', () => {
    const assembler = new V6StyleAssembler();

    // Create Uint8Array views with non-zero byteOffset inside a shared ArrayBuffer
    const fullBuffer = new Uint8Array([0, 1, 2, 3, 4, 5, 6, 7, 8, 9]).buffer;
    const msdfSubarray = new Uint8Array(fullBuffer, 2, 3); // bytes [2, 3, 4]
    const bmfontSubarray = new Uint8Array(fullBuffer, 6, 4); // bytes [6, 7, 8, 9]

    const mockChunk = create(TimelineStyleChunkSchema, {
      severities: [],
      verbs: [],
      logTypes: [],
      revisionStates: [],
      timelineTypes: [],
      iconAtlas: create(IconAtlasSchema, {
        msdfIconImage: [msdfSubarray],
        bmfontJson: bmfontSubarray,
        nameToCodepoints: { testIcon: 'e001' },
      }),
    });

    assembler.ingest(mockChunk);

    const builder = jasmine.createSpyObj<InspectionDataBuilder>(
      'InspectionDataBuilder',
      [
        'addSeverities',
        'addVerbs',
        'addLogTypes',
        'addRevisionStates',
        'addTimelineTypes',
        'setIconAtlas',
      ],
    );
    assembler.assembleInto(builder);

    expect(builder.setIconAtlas).toHaveBeenCalledTimes(1);
    const passedAtlas = builder.setIconAtlas.calls.mostRecent().args[0];

    expect(passedAtlas.msdfIconImage.length).toBe(1);
    expect(new Uint8Array(passedAtlas.msdfIconImage[0] as ArrayBuffer)).toEqual(
      new Uint8Array([2, 3, 4]),
    );
    expect(new Uint8Array(passedAtlas.bmfontJson as ArrayBuffer)).toEqual(
      new Uint8Array([6, 7, 8, 9]),
    );
    expect(passedAtlas.nameToCodepoints.get('testIcon')).toBe('e001');
  });
});

describe('V6TimelineAssembler', () => {
  it('should ingest timelines and timeline items and assemble them into builder', () => {
    const assembler = new V6TimelineAssembler();
    const mockChunk = create(TimelineChunkSchema, {
      timelineItems: [
        create(TimelineItemsSchema, {
          id: 100,
          revisions: [
            create(RevisionSchema, {
              logId: 10,
              changedTime: create(TimestampSchema, { seconds: 1n, nanos: 0 }),
              principalStringId: 5,
              verbType: 2,
              stateType: 3,
              resourceBodyStructId: 99,
              fieldAnnotations: [],
            }),
          ],
          events: [
            create(EventSchema, {
              logId: 20,
            }),
          ],
        }),
      ],
      timelines: [
        create(TimelineSchema, {
          id: 1,
          timelineType: 10,
          nameStringId: 20,
          timelineItemsId: 100,
          parentTimelineId: 0,
        }),
      ],
    });

    assembler.ingest(mockChunk);

    const builder = jasmine.createSpyObj<InspectionDataBuilder>(
      'InspectionDataBuilder',
      ['addRevisions', 'addEvents', 'addTimelines'],
    );
    assembler.assembleInto(builder);

    expect(builder.addRevisions).toHaveBeenCalledWith([
      {
        id: 1,
        logId: 10,
        changedTime: 1000000000n,
        principalStringId: 5,
        verbTypeId: 2,
        stateTypeId: 3,
        resourceBodyStructId: 99,
        fieldAnnotations: [],
      },
    ]);
    expect(builder.addEvents).toHaveBeenCalledWith([
      {
        id: 1,
        logId: 20,
      },
    ]);
    expect(builder.addTimelines).toHaveBeenCalledWith([
      {
        id: 1,
        timelineTypeId: 10,
        nameStringId: 20,
        parentTimelineId: 0,
        revisionIds: [1],
        eventIds: [1],
      },
    ]);
  });
});

describe('V6MetadataAssembler', () => {
  it('should ingest metadata chunk and decode oneof items into builder', () => {
    const assembler = new V6MetadataAssembler();

    const headerPayload = {
      inspectionType: 'type-a',
      inspectionName: 'name-a',
      inspectionTypeIconPath: 'path-a',
      startTimeUnixSeconds: 10n,
      endTimeUnixSeconds: 20n,
      inspectTimeUnixSeconds: 100n,
      suggestedFilename: 'file-a',
      fileSize: 0n,
    };

    const mockChunk = create(MetadataChunkSchema, {
      metadata: [
        create(MetadataItemSchema, {
          payload: {
            case: 'header',
            value: create(HeaderMetadataSchema, headerPayload),
          },
        }),
        create(MetadataItemSchema, {
          payload: {
            case: 'query',
            value: create(QueryMetadataSchema, {
              queries: [
                {
                  id: 'q1',
                  name: 'query-a',
                  query: 'select *',
                },
              ],
            }),
          },
        }),
      ],
    });

    assembler.ingest(mockChunk);

    const builder = jasmine.createSpyObj<InspectionDataBuilder>(
      'InspectionDataBuilder',
      ['setMetadataHeader', 'addMetadataQueries'],
    );
    assembler.assembleInto(builder);

    expect(builder.setMetadataHeader).toHaveBeenCalledWith({
      inspectionType: 'type-a',
      inspectionName: 'name-a',
      inspectTimeUnixSeconds: 100,
      startTimeUnixSeconds: 10,
      endTimeUnixSeconds: 20,
      suggestedFilename: 'file-a',
      fileSize: 0,
    });
    expect(builder.addMetadataQueries).toHaveBeenCalledWith([
      {
        id: 'q1',
        name: 'query-a',
        query: 'select *',
      },
    ]);
  });
});
