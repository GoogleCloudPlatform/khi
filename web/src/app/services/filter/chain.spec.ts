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
  of,
  ReplaySubject,
  take,
  toArray,
} from 'rxjs';
import {
  FilterChain,
  FilterChainElement,
  PropertyMatchRegexFilterChainElement,
  PropertyMatchSetFilterChainElement,
} from './chain';

describe('FilterChain', () => {
  it('should emit the source observable when no filter elements are added', (done) => {
    const source = of([1, 2, 3]);
    const filterChain = new FilterChain(source);

    filterChain.filtered.pipe(take(1), toArray()).subscribe((result) => {
      expect(result).toEqual([[1, 2, 3]]);
      done();
    });
  });

  it('should emit the result if FilterChinElement changed filter result', (done) => {
    const source = of([1, 2, 3, 4, 5, 6]);
    const filterChain = new FilterChain(source);
    const divisorSubscription = new ReplaySubject<number>(1);
    const overThanSubscription = new ReplaySubject<number>(1);
    divisorSubscription.next(2);
    overThanSubscription.next(2);

    const filter1: FilterChainElement<number> = {
      chain: (items: Observable<number[]>) =>
        items.pipe(
          combineLatestWith(divisorSubscription),
          map(([numbers, divisor]) => numbers.filter((n) => n % divisor === 0)),
        ),
    };
    const filter2: FilterChainElement<number> = {
      chain: (items: Observable<number[]>) =>
        items.pipe(
          combineLatestWith(overThanSubscription),
          map(([numbers, overThan]) => numbers.filter((n) => n > overThan)),
        ),
    };

    filterChain.filtered.pipe(take(5), toArray()).subscribe((result) => {
      expect(result).toEqual([
        [1, 2, 3, 4, 5, 6],
        [2, 4, 6],
        [4, 6],
        [3, 6],
        [6],
      ]);
      done();
    });

    filterChain.addFilterElement(filter1);
    filterChain.addFilterElement(filter2);
    divisorSubscription.next(3);
    overThanSubscription.next(4);
  });

  it('should apply filter elements in the order they are added', (done) => {
    const source = of([1, 2, 3, 4, 5]);
    const filterChain = new FilterChain(source);

    const filter1: FilterChainElement<number> = {
      chain: (items: Observable<number[]>) =>
        items.pipe(map((numbers) => numbers.filter((n) => n % 2 === 0))),
    };
    const filter2: FilterChainElement<number> = {
      chain: (items: Observable<number[]>) =>
        items.pipe(map((numbers) => numbers.filter((n) => n > 2))),
    };

    filterChain.filtered.pipe(take(3), toArray()).subscribe((result) => {
      expect(result).toEqual([[1, 2, 3, 4, 5], [2, 4], [4]]);
      done();
    });

    filterChain.addFilterElement(filter1);
    filterChain.addFilterElement(filter2);
  });

  it('should remove filter elements correctly', (done) => {
    const source = of([1, 2, 3, 4, 5]);
    const filterChain = new FilterChain(source);

    const filter1: FilterChainElement<number> = {
      chain: (items: Observable<number[]>) =>
        items.pipe(map((numbers) => numbers.filter((n) => n % 2 === 0))),
    };

    filterChain.filtered.pipe(take(3), toArray()).subscribe((result) => {
      expect(result).toEqual([
        [1, 2, 3, 4, 5],
        [2, 4],
        [1, 2, 3, 4, 5],
      ]);
      done();
    });

    filterChain.addFilterElement(filter1);
    filterChain.removeFilterElement(filter1);
  });
});

describe('PropertyMatchSetFilterChainElement', () => {
  it('should filter items based on the provided set', (done) => {
    const items$ = of([{ id: 1 }, { id: 2 }, { id: 3 }]);
    const filterSet$ = new BehaviorSubject(new Set([1, 3]));
    const filter = new PropertyMatchSetFilterChainElement(
      (item: { id: number }) => item.id,
      filterSet$,
    );

    filter.chain(items$).subscribe((filteredItems) => {
      expect(filteredItems).toEqual([{ id: 1 }, { id: 3 }]);
      done();
    });
  });

  it('should not filter any items if the set is empty', (done) => {
    const items$ = of([{ id: 1 }, { id: 2 }, { id: 3 }]);
    const filterSet$ = new BehaviorSubject(new Set<number>());
    const filter = new PropertyMatchSetFilterChainElement(
      (item: { id: number }) => item.id,
      filterSet$,
    );

    filter.chain(items$).subscribe((filteredItems) => {
      expect(filteredItems).toEqual([]);
      done();
    });
  });

  it('should filter items based on the updated set', (done) => {
    const items$ = of([{ id: 1 }, { id: 2 }, { id: 3 }]);
    const filterSet$ = new BehaviorSubject(new Set([1, 3]));
    const filter = new PropertyMatchSetFilterChainElement(
      (item: { id: number }) => item.id,
      filterSet$,
    );

    let i = 0;
    filter.chain(items$).subscribe((filteredItems) => {
      if (i === 0) {
        expect(filteredItems).toEqual([{ id: 1 }, { id: 3 }]);
        i++;
        filterSet$.next(new Set([2]));
      } else if (i === 1) {
        expect(filteredItems).toEqual([{ id: 2 }]);
        done();
      }
    });
  });

  it('should include items when needEvaluate returns false', (done) => {
    const items$ = of([
      { id: 1, name: 'a' },
      { id: 2, name: 'b' },
      { id: 3, name: 'c' },
    ]);
    const filterSet$ = new BehaviorSubject(new Set([1, 3]));
    const filter = new PropertyMatchSetFilterChainElement(
      (item: { id: number; name: string }) => item.id,
      filterSet$,
      (item) => item.name !== 'b',
    );

    filter.chain(items$).subscribe((filteredItems) => {
      expect(filteredItems).toEqual([
        { id: 1, name: 'a' },
        { id: 2, name: 'b' },
        { id: 3, name: 'c' },
      ]);
      done();
    });
  });

  it('should filter items when needEvaluate returns true', (done) => {
    const items$ = of([
      { id: 1, name: 'a' },
      { id: 2, name: 'b' },
      { id: 3, name: 'c' },
    ]);
    const filterSet$ = new BehaviorSubject(new Set([1, 3]));
    const filter = new PropertyMatchSetFilterChainElement(
      (item: { id: number; name: string }) => item.id,
      filterSet$,
      (item) => item.name !== 'd',
    );

    filter.chain(items$).subscribe((filteredItems) => {
      expect(filteredItems).toEqual([
        { id: 1, name: 'a' },
        { id: 3, name: 'c' },
      ]);
      done();
    });
  });

  it('should handle different property types', (done) => {
    const items$ = of([{ value: 'a' }, { value: 'b' }, { value: 'c' }]);
    const filterSet$ = new BehaviorSubject(new Set(['a', 'c']));
    const filter = new PropertyMatchSetFilterChainElement(
      (item: { value: string }) => item.value,
      filterSet$,
    );

    filter.chain(items$).subscribe((filteredItems) => {
      expect(filteredItems).toEqual([{ value: 'a' }, { value: 'c' }]);
      done();
    });
  });
});

describe('PropertyMatchRegexFilterChainElement', () => {
  it('should filter items based on the provided regexes', (done) => {
    const items$ = of([{ name: 'foo' }, { name: 'bar' }, { name: 'baz' }]);
    const regexes$ = new BehaviorSubject([/ba+r/, /baz/]);
    const filter = new PropertyMatchRegexFilterChainElement(
      (item: { name: string }) => item.name,
      regexes$,
    );

    filter.chain(items$).subscribe((filteredItems) => {
      expect(filteredItems).toEqual([{ name: 'bar' }, { name: 'baz' }]);
      done();
    });
  });

  it('should not filter any items if the regex array is empty', (done) => {
    const items$ = of([{ name: 'foo' }, { name: 'bar' }, { name: 'baz' }]);
    const regexes$ = new BehaviorSubject([]);
    const filter = new PropertyMatchRegexFilterChainElement(
      (item: { name: string }) => item.name,
      regexes$,
    );

    filter.chain(items$).subscribe((filteredItems) => {
      expect(filteredItems).toEqual([]);
      done();
    });
  });

  it('should filter items based on the updated regexes', (done) => {
    const items$ = of([{ name: 'foo' }, { name: 'bar' }, { name: 'baz' }]);
    const regexes$ = new BehaviorSubject([/ba+r/, /baz/]);
    const filter = new PropertyMatchRegexFilterChainElement(
      (item: { name: string }) => item.name,
      regexes$,
    );

    let i = 0;
    filter.chain(items$).subscribe((filteredItems) => {
      if (i === 0) {
        expect(filteredItems).toEqual([{ name: 'bar' }, { name: 'baz' }]);
        i++;
        regexes$.next([/foo/]);
      } else if (i === 1) {
        expect(filteredItems).toEqual([{ name: 'foo' }]);
        done();
      }
    });
  });

  it('should include items when needEvaluate returns false', (done) => {
    const items$ = of([
      { name: 'foo', value: 1 },
      { name: 'bar', value: 2 },
      { name: 'baz', value: 3 },
    ]);
    const regexes$ = new BehaviorSubject([/ba+r/, /baz/]);
    const filter = new PropertyMatchRegexFilterChainElement(
      (item: { name: string; value: number }) => item.name,
      regexes$,
      (item) => item.value !== 2,
    );

    filter.chain(items$).subscribe((filteredItems) => {
      expect(filteredItems).toEqual([
        { name: 'bar', value: 2 },
        { name: 'baz', value: 3 },
      ]);
      done();
    });
  });

  it('should filter items when needEvaluate returns true', (done) => {
    const items$ = of([
      { name: 'foo', value: 1 },
      { name: 'bar', value: 2 },
      { name: 'baz', value: 3 },
    ]);
    const regexes$ = new BehaviorSubject([/ba+r/, /baz/]);
    const filter = new PropertyMatchRegexFilterChainElement(
      (item: { name: string; value: number }) => item.name,
      regexes$,
      (item) => item.value !== 4,
    );

    filter.chain(items$).subscribe((filteredItems) => {
      expect(filteredItems).toEqual([
        { name: 'bar', value: 2 },
        { name: 'baz', value: 3 },
      ]);
      done();
    });
  });
});
