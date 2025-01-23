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

import * as yaml from 'js-yaml';
import { K8sResource } from './k8s-types';
import {
  RevisionState,
  RevisionStateMetadata,
  RevisionVerb,
  RevisionVerbMetadata,
} from '../generated';

export class ResourceRevision {
  private parsedManifestCache: K8sResource | null = null;
  public get parsedManifest(): K8sResource | undefined {
    if (this.parsedManifestCache === null) {
      this.parsedManifestCache = yaml.load(this.resourceContent) as K8sResource;
    }
    return this.parsedManifestCache;
  }

  public revisionStateCssSelector =
    RevisionStateMetadata[this.stateRaw].cssSelector;

  public revisionStateLabel = RevisionStateMetadata[this.stateRaw].label;

  public readonly verbCSSClass =
    RevisionVerbMetadata[this.lastMutationVerb].selector;

  public readonly verbLabel = RevisionVerbMetadata[this.lastMutationVerb].label;

  constructor(
    public readonly startAt: number,
    public readonly endAt: number,
    private readonly stateRaw: RevisionState,
    public readonly lastMutationVerb: RevisionVerb,
    public readonly resourceContent: string,
    public readonly requestor: string,
    public readonly isDeletion: boolean,
    public readonly isInferred: boolean,
    public readonly logIndex: number,
  ) {}

  /**
   * Get the duration of this revision being active.
   */
  get duration(): number {
    return this.endAt - this.startAt;
  }

  public static clone(revision: ResourceRevision): ResourceRevision {
    return new ResourceRevision(
      revision.startAt,
      revision.endAt,
      revision.stateRaw,
      revision.lastMutationVerb,
      revision.resourceContent,
      revision.requestor,
      revision.isDeletion,
      revision.isInferred,
      revision.logIndex,
    );
  }
}
