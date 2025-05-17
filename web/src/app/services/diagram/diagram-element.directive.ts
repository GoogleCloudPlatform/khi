/**
 * Copyright 2025 Google LLC
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
  Directive,
  ElementRef,
  inject,
  InjectionToken,
  input,
  OnDestroy,
  OnInit,
  signal,
} from '@angular/core';
import { distinctUntilChanged, map, Subject, take, takeUntil } from 'rxjs';
import { LOD } from './lod.service';
import { DiagramViewportService } from './diagram-viewport.service';

/**
 * InjectionToken to receive DiagramElementRole.
 */
export const DIAGRAM_ELEMENT_ROLE = new InjectionToken<DiagramElementRole>(
  'DIAGRAM_ELEMENT_ROLE',
);

/**
 * DiagramElementRole is the type of roles of the element in its children.
 * This decides if the element should decide its LOD or receive LOD from the other with the same ID.
 */
export enum DiagramElementRole {
  Invalid = 0,
  /**
   * The element of this children is in the minimap.
   * These read LODs from another element with CONTENT role and same ID.
   */
  Minimap = 1,
  /**
   * The element of this children is in the actual diagram contents.
   * It calculates its LOD and notify it to the other with same ID via DiagramViewportService.
   */
  Content = 2,
}

export const MAX_LOD = new InjectionToken<LOD>('MAX_LOD');

/**
 * DiagramElement directive monitor the location of the element and report LOD of the element.
 */
@Directive({
  selector: '[diagramElement]',
  exportAs: 'diagramElement',
})
export class DiagramElementDirective implements OnInit, OnDestroy {
  public readonly id = input.required<string>();

  private readonly destroyed = new Subject();

  private readonly role = inject(DIAGRAM_ELEMENT_ROLE);

  private readonly maxLOD = inject(MAX_LOD);

  private readonly viewportService = inject(DiagramViewportService);

  /**
   * LOD of the element.
   */
  public readonly lod = signal(LOD.CONTAINER_ONLY);

  constructor(private el: ElementRef<HTMLElement>) {}

  ngOnInit() {
    if (this.role === DiagramElementRole.Content) {
      // Use intersection observer to monitor if the element is inside of the visible area or not.
      const observer = new IntersectionObserver((elements) => {
        const isIntersecting = elements[0].isIntersecting;
        this.viewportService.notifyLOD(
          this.id(),
          isIntersecting ? LOD.DETAILED : LOD.CONTAINER_ONLY,
        );
      });
      observer.observe(this.el.nativeElement);
      this.destroyed.pipe(take(1)).subscribe(() => {
        observer.disconnect();
        observer.unobserve(this.el.nativeElement);
      });
    }
    this.viewportService
      .monitorLOD(this.id())
      .pipe(
        takeUntil(this.destroyed),
        map((givenLOD) => Math.min(givenLOD, this.maxLOD)),
        distinctUntilChanged(),
      )
      .subscribe((lod) => {
        this.lod.set(lod);
      });
  }

  ngOnDestroy() {
    this.destroyed.next(undefined);
    this.destroyed.complete();
    if (this.role === DiagramElementRole.Content) {
      this.viewportService.removeDiagramElement(this.id());
    }
  }
}
