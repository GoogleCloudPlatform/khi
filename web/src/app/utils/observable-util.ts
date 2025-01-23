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
  BehaviorSubject,
  Observable,
  animationFrames,
  distinctUntilChanged,
  endWith,
  map,
  shareReplay,
  takeWhile,
} from 'rxjs';

/**
 *
 * @deprecated prefer not to store the actual value with BehaviorSubject. Use pipe and shareReplay instead to support the late registered observable.
 */
export function asBehaviorSubject<T>(
  ob: Observable<T>,
  initialValue: T,
): BehaviorSubject<T> {
  const bs = new BehaviorSubject<T>(initialValue);
  ob.subscribe(bs);
  return bs;
}

/**
 * Returns an observable emitting number of tweening between start to end with duration milliseconds.
 */
export function tweenNumber(start: number, end: number, duration: number) {
  const diff = end - start;
  return animationFrames().pipe(
    map(({ elapsed }) => elapsed / duration),
    takeWhile((v) => v < 1),
    endWith(1),
    map((v) => v * diff + start),
  );
}

/**
 * Returns an observable that emits height of given element on its resize.
 */
export function monitorElementHeight(element: HTMLElement): Observable<number> {
  return new Observable<number>((subscriber) => {
    const box = element.getBoundingClientRect();
    subscriber.next(box.height);
    const observer = new ResizeObserver(() => {
      const box = element.getBoundingClientRect();
      subscriber.next(box.height);
    });
    observer.observe(element);
    return () => {
      observer.disconnect();
    };
  }).pipe(distinctUntilChanged(), shareReplay(1));
}
