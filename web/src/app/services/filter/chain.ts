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
  BehaviorSubject,
  combineLatestWith,
  map,
  Observable,
  switchMap,
} from 'rxjs';

export interface FilterChainElement<T> {
  chain(items: Observable<T[]>): Observable<T[]>;
}

/**
 * PropertyMatchSetFilterChainElement is a FilterChainElement implementation
 * that filters items based on whether a specified property of the item
 * is present within a given set of properties.
 */
export class PropertyMatchSetFilterChainElement<T, P>
  implements FilterChainElement<T>
{
  /**
   * Creates a new PropertyMatchSetFilterChainElement.
   * @param propertyReader A function that extracts the property value from an item.
   * @param filterProperty An observable that emits a set of property values to filter by.
   * @param needEvaluate A function that determines whether an item needs to be evaluated, the item is always included when the returned value is false. Defaults to always returning true.
   */
  constructor(
    private readonly propertyReader: (item: T) => P,
    private readonly filterProperty: Observable<Set<P>>,
    private readonly needEvaluate: (item: T) => boolean = () => true,
  ) {}
  chain(items: Observable<T[]>): Observable<T[]> {
    return items.pipe(
      combineLatestWith(this.filterProperty),
      map(([items, filterProperty]) =>
        items.filter(
          (item) =>
            !this.needEvaluate(item) ||
            filterProperty.has(this.propertyReader(item)),
        ),
      ),
    );
  }
}

/**
 * PropertyMatchRegexFilterChainElement is a FilterChainElement implementation that filters items based on whether a specified property of the item matches any of the given regular expressions.
 */
export class PropertyMatchRegexFilterChainElement<T>
  implements FilterChainElement<T>
{
  /**
   * Creates a new PropertyMatchRegexFilterChainElement.
   * @param propertyReader A function that extracts the property value from an item.
   * @param filterRegexs An observable that emits an array of regular expressions to filter by.
   * @param needEvaluate A function that determines whether an item needs to be evaluated, the item is always included when the returned value is false. Defaults to always returning true.
   */
  constructor(
    private readonly propertyReader: (item: T) => string,
    private readonly filterRegexs: Observable<RegExp[]>,
    private readonly needEvaluate: (item: T) => boolean = () => true,
  ) {}
  chain(items: Observable<T[]>): Observable<T[]> {
    return items.pipe(
      combineLatestWith(this.filterRegexs),
      map(([items, filterProperty]) =>
        items.filter(
          (item) =>
            !this.needEvaluate(item) ||
            filterProperty.some((regex) =>
              regex.test(this.propertyReader(item)),
            ),
        ),
      ),
    );
  }
}

/**
 * FilterChain generates a single observable that emits an array of T[]
 * which is filtered from the original Observable<T[]> by applying multiple FilterChainElement<T> instances.
 */
export class FilterChain<T> {
  public readonly currentFilterChain = new BehaviorSubject<
    FilterChainElement<T>[]
  >([]);

  public readonly filtered: Observable<T[]>;

  constructor(source: Observable<T[]>) {
    this.filtered = this.currentFilterChain.pipe(
      switchMap((elements) => FilterChain.applyFilters(elements, source)),
    );
  }

  addFilterElement(element: FilterChainElement<T>) {
    this.currentFilterChain.next([...this.currentFilterChain.value, element]);
  }

  removeFilterElement(element: FilterChainElement<T>) {
    this.currentFilterChain.next(
      this.currentFilterChain.value.filter((e) => e !== element),
    );
  }

  private static applyFilters<T>(
    filters: FilterChainElement<T>[],
    source: Observable<T[]>,
  ) {
    return filters.reduce((prev, current) => current.chain(prev), source);
  }
}
