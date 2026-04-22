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

import { fromBinary } from '@bufbuild/protobuf';
import {
  ChunkDefinition,
  IDataAssembler,
  ParserBlueprint,
} from 'src/app/parser/core/interfaces';
import { InspectionDataBuilder } from 'src/app/parser/core/builder';
import {
  InterningPoolChunk,
  InterningPoolChunkSchema,
} from 'src/app/generated/khifile/v6/intern_pool_pb';
import { LogChunk, LogChunkSchema } from 'src/app/generated/khifile/v6/log_pb';
import {
  MetadataChunk,
  MetadataChunkSchema,
} from 'src/app/generated/khifile/v6/metadata_pb';
import {
  HDRColor4 as PbHDRColor4,
  RevisionStateStyle as PbRevisionStateStyle,
  TimelineStyleChunk,
  TimelineStyleChunkSchema,
} from 'src/app/generated/khifile/v6/style_pb';
import {
  HDRColor4 as DomainHDRColor4,
  LogType as DomainLogType,
  RevisionState as DomainRevisionState,
  RevisionStateStyle as DomainRevisionStateStyle,
  Severity as DomainSeverity,
  TimelineType as DomainTimelineType,
  Verb as DomainVerb,
} from 'src/app/store/domain/style';
import {
  Timeline,
  TimelineChunk,
  TimelineChunkSchema,
} from 'src/app/generated/khifile/v6/timeline_pb';
import {
  StringEntryDTO,
  FieldPathSetEntryDTO,
} from 'src/app/store/domain/intern-pool-store';
import { LogDTO } from 'src/app/store/domain/log-store';
import {
  EventDTO,
  RevisionDTO,
  TimelineDTO,
} from 'src/app/store/domain/timeline-store';
import { IconAtlasDTO } from 'src/app/store/domain/style-store';

/**
 * Assembler for the file metadata chunk.
 */
export class V6MetadataAssembler implements IDataAssembler<MetadataChunk> {
  private readonly chunks: MetadataChunk[] = [];

  /**
   * Ingests a decoded metadata chunk.
   */
  ingest(proto: MetadataChunk): void {
    this.chunks.push(proto);
  }

  /**
   * Integrates metadata into the final builder.
   */
  assembleInto(builder: InspectionDataBuilder): void {
    for (const chunk of this.chunks) {
      for (const item of chunk.metadata) {
        if (item.payload.case === 'header') {
          const h = item.payload.value;
          builder.setMetadataHeader({
            inspectionType: h.inspectionType,
            inspectionName: h.inspectionName,
            inspectTimeUnixSeconds: Number(h.inspectTimeUnixSeconds),
            startTimeUnixSeconds: Number(h.startTimeUnixSeconds),
            endTimeUnixSeconds: Number(h.endTimeUnixSeconds),
            suggestedFilename: h.suggestedFilename,
            fileSize: Number(h.fileSize),
          });
        } else if (item.payload.case === 'query') {
          const queries = item.payload.value.queries.map((q) => ({
            id: q.id,
            name: q.name,
            query: q.query,
          }));
          builder.addMetadataQueries(queries);
        }
      }
    }
  }
}

/**
 * Assembler for the interning pool chunk.
 */
export class V6InternPoolAssembler implements IDataAssembler<InterningPoolChunk> {
  private readonly strings: StringEntryDTO[] = [];
  private readonly fieldPathSets: FieldPathSetEntryDTO[] = [];

  /**
   * Ingests a decoded interning pool chunk.
   */
  ingest(proto: InterningPoolChunk): void {
    for (const s of proto.strings) {
      this.strings.push({ id: s.id, value: s.value });
    }
    for (const f of proto.fieldPathSets) {
      this.fieldPathSets.push({
        id: f.id,
        fieldPathStringIds: f.fieldPathStringIds,
      });
    }
  }

  /**
   * Integrates pooled strings and field paths into the final builder.
   */
  assembleInto(builder: InspectionDataBuilder): void {
    builder.addStrings(this.strings);
    builder.addFieldPathSets(this.fieldPathSets);
  }
}

function mapColor(proto?: PbHDRColor4): DomainHDRColor4 {
  return proto
    ? { r: proto.r, g: proto.g, b: proto.b, a: proto.a }
    : { r: 0, g: 0, b: 0, a: 1 };
}

function mapRevisionStyle(
  proto: PbRevisionStateStyle,
): DomainRevisionStateStyle {
  switch (proto) {
    case PbRevisionStateStyle.NORMAL:
      return DomainRevisionStateStyle.NORMAL;
    case PbRevisionStateStyle.DELETED:
      return DomainRevisionStateStyle.DELETED;
    case PbRevisionStateStyle.PARTIAL_INFO:
      return DomainRevisionStateStyle.PARTIAL_INFO;
    default:
      return DomainRevisionStateStyle.NORMAL;
  }
}

/**
 * Assembler for the timeline style chunk.
 */
export class V6StyleAssembler implements IDataAssembler<TimelineStyleChunk> {
  private readonly severities: DomainSeverity[] = [];
  private readonly verbs: DomainVerb[] = [];
  private readonly logTypes: DomainLogType[] = [];
  private readonly revisionStates: DomainRevisionState[] = [];
  private readonly timelineTypes: DomainTimelineType[] = [];
  private iconAtlas?: IconAtlasDTO;

  /**
   * Ingests a decoded timeline style chunk.
   */
  ingest(proto: TimelineStyleChunk): void {
    for (const s of proto.severities) {
      this.severities.push({
        id: s.id,
        label: s.label,
        shortLabel: s.shortLabel,
        backgroundColor: mapColor(s.backgroundColor),
        foregroundColor: mapColor(s.foregroundColor),
        order: s.order,
      });
    }
    for (const v of proto.verbs) {
      this.verbs.push({
        id: v.id,
        label: v.label,
        backgroundColor: mapColor(v.backgroundColor),
        foregroundColor: mapColor(v.foregroundColor),
        visible: v.visible,
      });
    }
    for (const l of proto.logTypes) {
      this.logTypes.push({
        id: l.id,
        label: l.label,
        description: l.description,
        backgroundColor: mapColor(l.backgroundColor),
        foregroundColor: mapColor(l.foregroundColor),
      });
    }
    for (const r of proto.revisionStates) {
      this.revisionStates.push({
        id: r.id,
        label: r.label,
        icon: r.icon,
        description: r.description,
        backgroundColor: mapColor(r.backgroundColor),
        style: mapRevisionStyle(r.style),
      });
    }
    for (const t of proto.timelineTypes) {
      this.timelineTypes.push({
        id: t.id,
        label: t.label,
        description: t.description,
        backgroundColor: mapColor(t.backgroundColor),
        foregroundColor: mapColor(t.foregroundColor),
        typeChipBackgroundColor: mapColor(t.typeChipBackgroundColor),
        visible: t.visible,
        sortPriority: t.sortPriority,
        icon: t.icon,
        height: t.height,
      });
    }
    if (proto.iconAtlas) {
      const msdfIconImage = proto.iconAtlas.msdfIconImage.map((bytes) =>
        bytes.buffer.slice(
          bytes.byteOffset,
          bytes.byteOffset + bytes.byteLength,
        ),
      );
      const bmfontJson = proto.iconAtlas.bmfontJson.buffer.slice(
        proto.iconAtlas.bmfontJson.byteOffset,
        proto.iconAtlas.bmfontJson.byteOffset +
          proto.iconAtlas.bmfontJson.byteLength,
      );
      const nameToCodepoints = new Map<string, string>(
        Object.entries(proto.iconAtlas.nameToCodepoints),
      );
      this.iconAtlas = {
        msdfIconImage,
        bmfontJson,
        nameToCodepoints,
      };
    }
  }

  /**
   * Integrates timeline styles into the final builder.
   */
  assembleInto(builder: InspectionDataBuilder): void {
    builder.addSeverities(this.severities);
    builder.addVerbs(this.verbs);
    builder.addLogTypes(this.logTypes);
    builder.addRevisionStates(this.revisionStates);
    builder.addTimelineTypes(this.timelineTypes);
    if (this.iconAtlas) {
      builder.setIconAtlas(this.iconAtlas);
    }
  }
}

/**
 * Assembler for the log chunk.
 */
export class V6LogAssembler implements IDataAssembler<LogChunk> {
  private readonly logs: LogDTO[] = [];

  /**
   * Ingests a decoded log chunk.
   */
  ingest(proto: LogChunk): void {
    for (const log of proto.logs) {
      const ts = log.ts
        ? BigInt(log.ts.seconds) * 1_000_000_000n + BigInt(log.ts.nanos)
        : 0n;
      this.logs.push({
        id: log.id,
        ts,
        logTypeId: log.logTypeId,
        severityTypeId: log.severityTypeId,
        summaryStringId: log.summaryStringId,
        body: log.body,
      });
    }
  }

  /**
   * Integrates logs into the final builder.
   */
  assembleInto(builder: InspectionDataBuilder): void {
    builder.addLogs(this.logs);
  }
}

/**
 * Assembler for the timeline chunk.
 */
export class V6TimelineAssembler implements IDataAssembler<TimelineChunk> {
  private readonly rawTimelines: Timeline[] = [];
  private readonly revisions: RevisionDTO[] = [];
  private readonly events: EventDTO[] = [];
  private readonly itemsMap = new Map<
    number,
    { revisionIds: number[]; eventIds: number[] }
  >();

  private nextRevisionId = 1;
  private nextEventId = 1;

  /**
   * Ingests a decoded timeline chunk.
   */
  ingest(proto: TimelineChunk): void {
    // 1. Process timelineItems
    for (const items of proto.timelineItems) {
      const revisionIds: number[] = [];
      for (const r of items.revisions) {
        const id = this.nextRevisionId++;
        const changedTime = r.changedTime
          ? BigInt(r.changedTime.seconds) * 1_000_000_000n +
            BigInt(r.changedTime.nanos)
          : 0n;
        this.revisions.push({
          id,
          logId: r.logId,
          changedTime,
          principalStringId: r.principalStringId,
          verbTypeId: r.verbType,
          stateTypeId: r.stateType,
          body: r.resourceBody,
        });
        revisionIds.push(id);
      }

      const eventIds: number[] = [];
      for (const e of items.events) {
        const id = this.nextEventId++;
        this.events.push({
          id,
          logId: e.logId,
        });
        eventIds.push(id);
      }

      this.itemsMap.set(items.id, { revisionIds, eventIds });
    }

    // 2. Store raw timelines
    for (const t of proto.timelines) {
      this.rawTimelines.push(t);
    }
  }

  /**
   * Integrates timelines into the final builder.
   */
  assembleInto(builder: InspectionDataBuilder): void {
    builder.addRevisions(this.revisions);
    builder.addEvents(this.events);

    // 1. Build flat TimelineDTO list with items linked
    const linkedTimelines: TimelineDTO[] = [];
    for (const t of this.rawTimelines) {
      const items = this.itemsMap.get(t.timelineItemsId) ?? {
        revisionIds: [],
        eventIds: [],
      };
      linkedTimelines.push({
        id: t.id,
        timelineTypeId: t.timelineType,
        nameStringId: t.nameStringId,
        parentTimelineId: t.parentTimelineId,
        revisionIds: items.revisionIds,
        eventIds: items.eventIds,
      });
    }

    builder.addTimelines(linkedTimelines);
  }
}

/**
 * Identifies the specific type of chunk in a KHI v6 container file.
 */
export enum V6ChunkType {
  /**
   * Contains file-level metadata.
   */
  Metadata = 1,
  /**
   * Contains optimized strings and field path pools.
   */
  InterningPool = 2,
  /**
   * Contains log entries.
   */
  Log = 3,
  /**
   * Contains visual timeline style definitions.
   */
  TimelineStyle = 4,
  /**
   * Contains resource timeline data.
   */
  Timeline = 5,
}

/**
 * The parser blueprint registry for KHI file version 6.
 */
export const V6_BLUEPRINT: ParserBlueprint = new Map<
  number,
  ChunkDefinition<unknown>
>([
  [
    V6ChunkType.Metadata,
    {
      typeId: V6ChunkType.Metadata,
      decode: (bytes) => fromBinary(MetadataChunkSchema, bytes),
      createAssembler: () => new V6MetadataAssembler(),
      priority: 5,
      label: 'metadata',
    },
  ],
  [
    V6ChunkType.InterningPool,
    {
      typeId: V6ChunkType.InterningPool,
      decode: (bytes) => fromBinary(InterningPoolChunkSchema, bytes),
      createAssembler: () => new V6InternPoolAssembler(),
      priority: 10,
      label: 'interningPool',
    },
  ],
  [
    V6ChunkType.TimelineStyle,
    {
      typeId: V6ChunkType.TimelineStyle,
      decode: (bytes) => fromBinary(TimelineStyleChunkSchema, bytes),
      createAssembler: () => new V6StyleAssembler(),
      priority: 20,
      label: 'timelineStyle',
    },
  ],
  [
    V6ChunkType.Log,
    {
      typeId: V6ChunkType.Log,
      decode: (bytes) => fromBinary(LogChunkSchema, bytes),
      createAssembler: () => new V6LogAssembler(),
      priority: 100,
      label: 'log',
    },
  ],
  [
    V6ChunkType.Timeline,
    {
      typeId: V6ChunkType.Timeline,
      decode: (bytes) => fromBinary(TimelineChunkSchema, bytes),
      createAssembler: () => new V6TimelineAssembler(),
      priority: 100,
      label: 'timeline',
    },
  ],
]);
