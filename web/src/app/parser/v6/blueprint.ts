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
  DataAssembler,
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
  RevisionStateStyle as DomainRevisionStateStyle,
} from 'src/app/store/domain/style';
import {
  Timeline,
  TimelineChunk,
  TimelineChunkSchema,
} from 'src/app/generated/khifile/v6/timeline_pb';
import { TimelineDTO } from 'src/app/store/domain/timeline-store';
import {
  StructValueDTO,
  StructValueKind,
} from 'src/app/store/domain/struct-store';
import { InternedValue } from 'src/app/generated/khifile/shared_pb';

/**
 * Assembler for the file metadata chunk.
 */
export class V6MetadataAssembler implements DataAssembler<MetadataChunk> {
  constructor(private readonly builder: InspectionDataBuilder) {}

  /**
   * Ingests a decoded metadata chunk directly into the inspection data builder.
   */
  ingest(proto: MetadataChunk): void {
    for (const item of proto.metadata) {
      if (item.payload.case === 'header') {
        const h = item.payload.value;
        this.builder.setMetadataHeader({
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
        this.builder.addMetadataQueries(queries);
      }
    }
  }
}

function mapStructValue(iv: InternedValue): StructValueDTO {
  switch (iv.kind.case) {
    case 'nullValue':
      return { kind: StructValueKind.Null };
    case 'boolValue':
      return { kind: StructValueKind.Bool, value: iv.kind.value };
    case 'int64Value':
      return { kind: StructValueKind.Int64, value: iv.kind.value };
    case 'doubleValue':
      return { kind: StructValueKind.Double, value: iv.kind.value };
    case 'stringValue':
      return { kind: StructValueKind.String, stringId: iv.kind.value };
    case 'structId':
      return { kind: StructValueKind.StructId, structId: iv.kind.value };
    case 'structValue':
      return { kind: StructValueKind.StructId, structId: iv.kind.value.id };
    case 'listValue':
      return {
        kind: StructValueKind.List,
        values: iv.kind.value.values.map(mapStructValue),
      };
    case 'timestampValue':
      return {
        kind: StructValueKind.Timestamp,
        timestampNs:
          BigInt(iv.kind.value.seconds) * 1_000_000_000n +
          BigInt(iv.kind.value.nanos),
      };
    default:
      return { kind: StructValueKind.Null };
  }
}

/**
 * Assembler for the interning pool chunk.
 */
export class V6InternPoolAssembler implements DataAssembler<InterningPoolChunk> {
  constructor(private readonly builder: InspectionDataBuilder) {}

  /**
   * Ingests a decoded interning pool chunk directly into the inspection data builder.
   */
  ingest(proto: InterningPoolChunk): void {
    for (const s of proto.strings) {
      this.builder.addString({ id: s.id, value: s.value });
    }
    for (const fps of proto.fieldPathSets) {
      this.builder.addFieldPathSet({
        id: fps.id,
        fieldPathStringIds: fps.fieldPathStringIds,
      });
    }
    for (const st of proto.structs) {
      this.builder.addStruct({
        id: st.id,
        fieldPathSetId: st.fieldPathSetId,
        values: st.values.map(mapStructValue),
      });
    }
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
export class V6StyleAssembler implements DataAssembler<TimelineStyleChunk> {
  constructor(private readonly builder: InspectionDataBuilder) {}

  /**
   * Ingests a decoded timeline style chunk directly into the inspection data builder.
   */
  ingest(proto: TimelineStyleChunk): void {
    if (proto.severities.length > 0) {
      this.builder.addSeverities(
        proto.severities.map((s) => ({
          id: s.id,
          label: s.label,
          shortLabel: s.shortLabel,
          backgroundColor: mapColor(s.backgroundColor),
          foregroundColor: mapColor(s.foregroundColor),
          order: s.order,
        })),
      );
    }
    if (proto.verbs.length > 0) {
      this.builder.addVerbs(
        proto.verbs.map((v) => ({
          id: v.id,
          label: v.label,
          backgroundColor: mapColor(v.backgroundColor),
          foregroundColor: mapColor(v.foregroundColor),
          visible: v.visible,
        })),
      );
    }
    if (proto.logTypes.length > 0) {
      this.builder.addLogTypes(
        proto.logTypes.map((lt) => ({
          id: lt.id,
          label: lt.label,
          description: lt.description,
          backgroundColor: mapColor(lt.backgroundColor),
          foregroundColor: mapColor(lt.foregroundColor),
        })),
      );
    }
    if (proto.revisionStates.length > 0) {
      this.builder.addRevisionStates(
        proto.revisionStates.map((rs) => ({
          id: rs.id,
          label: rs.label,
          icon: rs.icon,
          description: rs.description,
          backgroundColor: mapColor(rs.backgroundColor),
          style: mapRevisionStyle(rs.style),
        })),
      );
    }
    if (proto.timelineTypes.length > 0) {
      this.builder.addTimelineTypes(
        proto.timelineTypes.map((tt) => ({
          id: tt.id,
          label: tt.label,
          description: tt.description,
          icon: tt.icon,
          backgroundColor: mapColor(tt.backgroundColor),
          foregroundColor: mapColor(tt.foregroundColor),
          typeChipBackgroundColor: mapColor(tt.typeChipBackgroundColor),
          typeChipForegroundColor: mapColor(tt.typeChipForegroundColor),
          visible: tt.visible,
          sortPriority: tt.sortPriority,
          height: tt.height,
        })),
      );
    }
    if (proto.iconAtlas) {
      const msdfIconImage = proto.iconAtlas.msdfIconImage.map((u) =>
        u.buffer.slice(u.byteOffset, u.byteOffset + u.byteLength),
      );
      const bmfontJson = proto.iconAtlas.bmfontJson.buffer.slice(
        proto.iconAtlas.bmfontJson.byteOffset,
        proto.iconAtlas.bmfontJson.byteOffset +
          proto.iconAtlas.bmfontJson.byteLength,
      );
      const nameToCodepoints = new Map<string, string>(
        Object.entries(proto.iconAtlas.nameToCodepoints),
      );
      this.builder.setIconAtlas({
        msdfIconImage,
        bmfontJson,
        nameToCodepoints,
      });
    }
  }
}

/**
 * Assembler for the log chunk.
 */
export class V6LogAssembler implements DataAssembler<LogChunk> {
  constructor(private readonly builder: InspectionDataBuilder) {}

  /**
   * Ingests a decoded log chunk directly into the inspection data builder.
   */
  ingest(proto: LogChunk): void {
    for (const log of proto.logs) {
      const ts = log.ts
        ? BigInt(log.ts.seconds) * 1_000_000_000n + BigInt(log.ts.nanos)
        : 0n;
      this.builder.addLog({
        id: log.id,
        ts,
        logTypeId: log.logTypeId,
        severityTypeId: log.severityTypeId,
        summaryStringId: log.summaryStringId,
        bodyStructId: log.bodyStructId !== 0 ? log.bodyStructId : undefined,
      });
    }
  }
}

/**
 * Assembler for the timeline chunk.
 */
export class V6TimelineAssembler implements DataAssembler<TimelineChunk> {
  private readonly rawTimelines: Timeline[] = [];
  private readonly itemsMap = new Map<
    number,
    { revisionIds: number[]; eventIds: number[] }
  >();

  private nextRevisionId = 1;
  private nextEventId = 1;

  constructor(private readonly builder: InspectionDataBuilder) {}

  /**
   * Ingests a decoded timeline chunk directly into the inspection data builder.
   */
  ingest(proto: TimelineChunk): void {
    // 1. Process timelineItems: stream revisions and events directly to builder
    for (const items of proto.timelineItems) {
      if (this.itemsMap.has(items.id)) {
        throw new Error(`Duplicate timelineItems id: ${items.id}`);
      }
      const revisionIds: number[] = [];
      for (const r of items.revisions) {
        const id = this.nextRevisionId++;
        const changedTime = r.changedTime
          ? BigInt(r.changedTime.seconds) * 1_000_000_000n +
            BigInt(r.changedTime.nanos)
          : 0n;
        this.builder.addRevision({
          id,
          logId: r.logId,
          changedTime,
          principalStringId: r.principalStringId,
          verbTypeId: r.verbType,
          stateTypeId: r.stateType,
          resourceBodyStructId:
            r.resourceBodyStructId !== 0 ? r.resourceBodyStructId : undefined,
          fieldAnnotations: r.fieldAnnotations.map((fa) => {
            if (fa.payload.case === 'mutatingWebhook') {
              return {
                fieldPathStringId: fa.fieldPathStringId,
                mutatingWebhook: {
                  configurationStringId: fa.payload.value.configurationStringId,
                  webhookStringId: fa.payload.value.webhookStringId,
                  round: fa.payload.value.round,
                  index: fa.payload.value.index,
                },
              };
            }
            return {
              fieldPathStringId: fa.fieldPathStringId,
            };
          }),
        });
        revisionIds.push(id);
      }

      const eventIds: number[] = [];
      for (const e of items.events) {
        const id = this.nextEventId++;
        this.builder.addEvent({
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
   * Links timeline items to timelines and flattens into the inspection data builder.
   */
  finalize(): void {
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

    for (const timeline of linkedTimelines) {
      this.builder.addTimeline(timeline);
    }
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
      createAssembler: (builder) => new V6MetadataAssembler(builder),
      priority: 5,
      label: 'metadata',
    },
  ],
  [
    V6ChunkType.InterningPool,
    {
      typeId: V6ChunkType.InterningPool,
      decode: (bytes) => fromBinary(InterningPoolChunkSchema, bytes),
      createAssembler: (builder) => new V6InternPoolAssembler(builder),
      priority: 10,
      label: 'interningPool',
    },
  ],
  [
    V6ChunkType.TimelineStyle,
    {
      typeId: V6ChunkType.TimelineStyle,
      decode: (bytes) => fromBinary(TimelineStyleChunkSchema, bytes),
      createAssembler: (builder) => new V6StyleAssembler(builder),
      priority: 20,
      label: 'timelineStyle',
    },
  ],
  [
    V6ChunkType.Log,
    {
      typeId: V6ChunkType.Log,
      decode: (bytes) => fromBinary(LogChunkSchema, bytes),
      createAssembler: (builder) => new V6LogAssembler(builder),
      priority: 100,
      label: 'log',
    },
  ],
  [
    V6ChunkType.Timeline,
    {
      typeId: V6ChunkType.Timeline,
      decode: (bytes) => fromBinary(TimelineChunkSchema, bytes),
      createAssembler: (builder) => new V6TimelineAssembler(builder),
      priority: 100,
      label: 'timeline',
    },
  ],
]);
