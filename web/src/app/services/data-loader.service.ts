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

import { Inject, Injectable } from '@angular/core';
import { InspectionDataStoreService } from './inspection-data-store.service';
import { InspectionData, TimeRange } from '../store/inspection-data';
import {
  KHIFile,
  KHIFileResource,
  KHIFileResourceEvent,
  KHIFileResourceRevision,
  KHIFileTimeline,
} from '../common/schema/khi-file-types';
import { ParentRelationship, RevisionVerb } from '../generated';
import { lastValueFrom } from 'rxjs';
import { BACKEND_API, BackendAPI } from './api/backend-api-interface';
import {
  PROGRESS_DIALOG_STATUS_UPDATOR,
  ProgressDialogStatusUpdator,
} from './progress/progress-interface';
import { LogEntry } from '../store/log';
import { ResourceEvent } from '../store/event';
import { ResourceRevision } from '../store/revision';
import { TimelineEntry } from '../store/timeline';
import {
  EXTENSION_STORE,
  ExtensionStore,
} from '../extensions/extension-common/extension-store';
import {
  KHIFileReferenceResolver,
  NullReferenceResolver,
  ReferenceResolverStore,
} from '../common/loader/reference-resolver';
import { ToTextReferenceFromKHIFileBinary } from '../common/loader/reference-type';
import { ProgressUtil } from './progress/progress-util';

@Injectable()
export class InspectionDataLoaderService {
  constructor(
    @Inject(PROGRESS_DIALOG_STATUS_UPDATOR)
    private progress: ProgressDialogStatusUpdator,
    private inspectionDataStore: InspectionDataStoreService,
    @Inject(BACKEND_API) private backendService: BackendAPI,
    @Inject(EXTENSION_STORE) private extension: ExtensionStore,
  ) {}

  private eventDataToViewEvents(
    events: KHIFileResourceEvent[],
    logs: LogEntry[],
    idToIndexTable: { [key: string]: number },
  ): ResourceEvent[] {
    return events.map((a) => {
      const logEntryIndex = idToIndexTable[a.log];
      return new ResourceEvent(
        logEntryIndex,
        logs[logEntryIndex].time,
        logs[logEntryIndex].logType,
        logs[logEntryIndex].severity,
      );
    });
  }

  private async revisionDataToViewRevisions(
    resource: KHIFileResource,
    revisions: KHIFileResourceRevision[],
    idToIndexTable: { [logId: string]: number },
    startTime: number,
    endTime: number,
    textSource: ReferenceResolverStore,
  ): Promise<ResourceRevision[]> {
    const result: ResourceRevision[] = [];
    if (
      revisions.length === 0 &&
      resource.relationship === ParentRelationship.RelationshipChild &&
      !resource.name.endsWith('unknown')
    ) {
      return result;
    }
    for (let ri = 0; ri < revisions.length; ri++) {
      const revision = revisions[ri];
      let end = endTime;
      if (ri != revisions.length - 1) {
        end = Date.parse(revisions[ri + 1].changeTime);
      }
      result.push(
        new ResourceRevision(
          Date.parse(revisions[ri].changeTime),
          end,
          revisions[ri].state,
          revision.verb,
          await lastValueFrom(
            textSource.getText(ToTextReferenceFromKHIFileBinary(revision.body)),
          ),
          await lastValueFrom(
            textSource.getText(
              ToTextReferenceFromKHIFileBinary(revision.requestor),
            ),
          ),
          revision.verb === RevisionVerb.RevisionVerbDelete,
          false,
          idToIndexTable[revision.log],
        ),
      );
    }
    return result;
  }

  private async responseDataToViewInspection(
    response: KHIFile,
    referenceResolver: ReferenceResolverStore,
  ): Promise<InspectionData> {
    if (typeof response.version === 'undefined') {
      const errorMessage =
        'Unsupported KHI version schema. Maybe this file was exported for older KHI version. Please use older version to use the file or query the range again with this newer version';
      alert(errorMessage);
      throw new Error(errorMessage);
    }

    const logs: LogEntry[] = [];
    const startTime = response.metadata.header.startTimeUnixSeconds * 1000;
    const endTime = response.metadata.header.endTimeUnixSeconds * 1000;
    const logIdToLogIndex: { [logId: string]: number } = {
      '': -1,
    };
    const timelineIdToTimeline: {
      [timelineId: string]: KHIFileTimeline | undefined;
    } = {};
    this.progress.updateProgress({
      message: 'Processing logs...',
      percent: 0,
      mode: 'indeterminate',
    });
    // Process logs
    for (const l of response.logs) {
      const time = Date.parse(l.ts);
      logIdToLogIndex[l.id] = logs.length;
      logs.push(
        new LogEntry(
          logs.length,
          l.displayId,
          l.type,
          l.severity,
          time,
          await lastValueFrom(
            referenceResolver.getText(
              ToTextReferenceFromKHIFileBinary(l.summary),
            ),
          ),
          ToTextReferenceFromKHIFileBinary(l.body),
          l.annotations,
        ),
      );
    }
    this.progress.updateProgress({
      message: 'Processing timelines...',
      percent: 0,
      mode: 'indeterminate',
    });
    for (const timeline of response.timelines) {
      timelineIdToTimeline[timeline.id] = timeline;
    }
    // Process resource hierarchy
    const timelines: TimelineEntry[] = [];
    for (
      let apiVersionIndex = 0;
      apiVersionIndex < response.resources.length;
      apiVersionIndex++
    ) {
      const apiVersionResource = response.resources[apiVersionIndex];
      for (
        let kindIndex = 0;
        kindIndex < apiVersionResource.children.length;
        kindIndex++
      ) {
        const kindResource = apiVersionResource.children[kindIndex];
        const kindTimeline = new TimelineEntry(
          kindResource.path,
          [],
          [],
          kindResource.relationship,
        );
        timelines.push(kindTimeline);
        for (
          let namespaceIndex = 0;
          namespaceIndex < kindResource.children.length;
          namespaceIndex++
        ) {
          const namespaceResource = kindResource.children[namespaceIndex];
          const namespaceTimeline = new TimelineEntry(
            namespaceResource.path,
            [],
            [],
            namespaceResource.relationship,
          );
          timelines.push(namespaceTimeline);
          kindTimeline.addChildTimeline(namespaceTimeline);
          for (
            let nameIndex = 0;
            nameIndex < namespaceResource.children.length;
            nameIndex++
          ) {
            const nameResource = namespaceResource.children[nameIndex];
            // timeline can be null when the children is defined but no event/revisions are included
            const timeline =
              timelineIdToTimeline[nameResource.timeline] ?? null;
            const nameTimeline = new TimelineEntry(
              nameResource.path,
              await this.revisionDataToViewRevisions(
                nameResource,
                timeline?.revisions ?? [],
                logIdToLogIndex,
                startTime,
                endTime,
                referenceResolver,
              ),
              this.eventDataToViewEvents(
                timeline?.events ?? [],
                logs,
                logIdToLogIndex,
              ),
              nameResource.relationship,
            );
            timelines.push(nameTimeline);
            namespaceTimeline.addChildTimeline(nameTimeline);
            for (
              let subresoruceIndex = 0;
              subresoruceIndex < nameResource.children.length;
              subresoruceIndex++
            ) {
              const subResourceResource =
                nameResource.children[subresoruceIndex];
              // timeline can be null when the children is defined but no event/revisions are included
              const timeline =
                timelineIdToTimeline[subResourceResource.timeline] ?? null;
              const subresourceTimeline = new TimelineEntry(
                subResourceResource.path,
                await this.revisionDataToViewRevisions(
                  subResourceResource,
                  timeline?.revisions ?? [],
                  logIdToLogIndex,
                  startTime,
                  endTime,
                  referenceResolver,
                ),
                this.eventDataToViewEvents(
                  timeline?.events ?? [],
                  logs,
                  logIdToLogIndex,
                ),
                subResourceResource.relationship,
              );
              timelines.push(subresourceTimeline);
              nameTimeline.addChildTimeline(subresourceTimeline);
            }
          }
        }
      }
    }

    // Create cache of relationship from log entry to timeline
    for (const timeline of timelines) {
      for (const event of timeline.events) {
        logs[event.logIndex].relatedTimelines.add(timeline);
      }
      for (const revision of timeline.revisions) {
        if (revision.logIndex !== -1)
          logs[revision.logIndex].relatedTimelines.add(timeline);
      }
    }

    return new InspectionData(
      response.metadata.header,
      new TimeRange(startTime, endTime),
      referenceResolver,
      timelines,
      logs,
    );
  }

  /**
   * Open a dialog to open local file and accept that JSON as the inspection data.
   */
  public uploadFromFile() {
    const fileInput = document.createElement('input');
    fileInput.type = 'file';
    fileInput.style.display = 'none';
    document.body.appendChild(fileInput);
    fileInput.oninput = () => {
      const fileReader = new FileReader();
      fileReader.onload = () => {
        this.loadInspectionDataDirect(fileReader.result as ArrayBuffer);
        location.hash = '';
        fileInput.remove();
      };
      fileReader.readAsArrayBuffer(fileInput.files![0]);
    };
    fileInput.click();
  }

  public async loadInspectionDataDirect(rawInspectionData: ArrayBuffer) {
    this.progress.show();
    this.progress.updateProgress({
      message: 'Parsing inspection data...',
      percent: 0,
      mode: 'determinate',
    });
    try {
      if (!this.verifyMagicBytes(rawInspectionData)) {
        alert(
          'Given file seems not a KHI inspection data of current version. \nIf you want to load KHI inspection data collected with previous KHI version, please use the older version instead.',
        );
        return;
      }
      const jsonSizeOffset = 3;
      const jsonDataOffset = jsonSizeOffset + Uint32Array.BYTES_PER_ELEMENT;
      const fileDataView = new DataView(rawInspectionData);
      const metaDataPart = fileDataView.getUint32(jsonSizeOffset, true);
      const jsonPartBytes = new Uint8Array(
        rawInspectionData,
        jsonDataOffset,
        metaDataPart,
      );
      const textDecoder = new TextDecoder();
      const parsedJsonData = JSON.parse(textDecoder.decode(jsonPartBytes));
      const textBuffers = await this.decodeBuffers(
        rawInspectionData,
        jsonDataOffset + metaDataPart,
      );

      const resolver = new ReferenceResolverStore([
        new KHIFileReferenceResolver(textBuffers),
        new NullReferenceResolver(),
      ]);
      const khiInspectionViewModel = await this.responseDataToViewInspection(
        parsedJsonData,
        resolver,
      );

      this.extension.notifyLifecycleOnInspectionDataOpen(
        khiInspectionViewModel,
        resolver,
        rawInspectionData,
      );

      this.inspectionDataStore.setNewInspectionData(khiInspectionViewModel);
    } catch (e) {
      console.error(e);
      alert(
        `Failed to parse the inspection data. The given data was invalid or too big for this environment. \nPlease consider limiting the inspection duration shorter.`,
      );
    }
    this.progress.dismiss();
  }

  private async decodeBuffers(
    source: ArrayBuffer,
    initialOffset: number,
  ): Promise<ArrayBuffer[]> {
    const result: ArrayBuffer[] = [];
    const dv = new DataView(source);
    let currentOffset = initialOffset;
    while (currentOffset < source.byteLength) {
      this.progress.updateProgress({
        message: `Decompressing inspection data(${currentOffset}/${source.byteLength})`,
        percent: (currentOffset / source.byteLength) * 100,
        mode: 'determinate',
      });
      const size = dv.getUint32(currentOffset);
      currentOffset += Uint32Array.BYTES_PER_ELEMENT;
      const decompressedBuffer = await this.decompressGzip(
        new Uint8Array(source, currentOffset, size),
      );
      currentOffset += size;
      result.push(decompressedBuffer);
    }
    return result;
  }

  public async loadInspectionDataFromBackend(taskId: string) {
    this.progress.show();
    this.progress.updateProgress({
      message: 'Downloading inspection data...',
      percent: 0,
      mode: 'determinate',
    });
    const metadata = await lastValueFrom(
      this.backendService.getInspectionMetadata(taskId),
    );
    const allSize = metadata.header.fileSize ?? 0;
    const data = await lastValueFrom(
      this.backendService.getInspectionData(taskId, (done) => {
        this.progress.updateProgress({
          message: `Downloading inspection data...(${ProgressUtil.formatPogressMessageByBytes(
            done,
            allSize,
          )})`,
          percent: (done / allSize) * 100,
          mode: 'determinate',
        });
      }),
    );
    this.progress.dismiss();
    if (data === null) {
      alert(
        `Failed to load the inspection data. The inspection data may exceed 500MB. \nPlease try query with shorter duration.`,
      );
      return;
    }
    this.loadInspectionDataDirect(await data.arrayBuffer());
  }

  private verifyMagicBytes(source: ArrayBuffer): boolean {
    const dv = new DataView(source);
    const magic = [dv.getUint8(0), dv.getUint8(1), dv.getUint8(2)];
    const expected = [75, 72, 73];
    return magic.every((v, i) => expected[i] === v);
  }

  private decompressGzip(source: Uint8Array): Promise<ArrayBuffer> {
    // Predefined DecompressionStream only accepts an argument. This is not aligning with the actual scheme.
    // Casting the constructor to any once to call it with a valid constructor argument.
    /* eslint-disable-next-line @typescript-eslint/no-explicit-any */
    const decompressionStream = new (window as any).DecompressionStream('gzip');
    const sourceBlob = new Blob([source]);
    const textDecompressionStream = sourceBlob
      .stream()
      .pipeThrough(decompressionStream);
    return new Response(textDecompressionStream).arrayBuffer();
  }
}
