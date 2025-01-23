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

import {
  EnvironmentInjector,
  Type,
  runInInjectionContext,
} from '@angular/core';
import { Observable, map } from 'rxjs';

export type AnnotationDecider<
  T,
  R extends AnnotationDecision = AnnotationDecision,
> = (source?: T | null) => R;

export interface AnnotationDecision {
  hidden?: boolean;
  inputs?: Record<string, unknown>;
}

export const DECISION_HIDDEN: AnnotationDecision = {
  hidden: true,
};

/**
 * Annotator is extensible information added on a specific resource(Log,Timeline,ResourceDiff,...etc)
 */
export class Annotator<T> {
  constructor(
    public readonly component: Type<unknown>,
    protected readonly decider: AnnotationDecider<T>,
  ) {}

  public decision(source?: T | null): AnnotationDecision {
    return this.decider(source);
  }
}

/**
 * A type to be passed to template to render the dynamic components.
 */
export interface ResolvedAnnotator {
  component: Type<unknown>;
  inputs: Record<string, unknown>;
}

export class AnnotatorResolver<T> {
  constructor(public readonly annotators: Annotator<T>[]) {}

  /**
   * Returns an observable of ResolvedAnnotator from an observable of source parameter.
   * @param source
   * @param injectionContext
   * @returns
   */
  public getResolvedAnnotators(
    source: Observable<T | null>,
    injectionContext: EnvironmentInjector,
  ): Observable<ResolvedAnnotator[]> {
    return source.pipe(
      map((source) =>
        runInInjectionContext(injectionContext, () =>
          this.annotators
            .map((annotator) => ({
              decision: annotator.decision(source),
              component: annotator.component,
            }))
            .filter((annotatorDecisions) => !annotatorDecisions.decision.hidden)
            .map((annotatorDecisions) => ({
              component: annotatorDecisions.component,
              inputs: annotatorDecisions.decision.inputs ?? {},
            })),
        ),
      ),
    );
  }
}
