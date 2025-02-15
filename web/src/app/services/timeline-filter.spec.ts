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

import { debounceTime, NEVER, ReplaySubject, take } from 'rxjs';
import { InspectionDataStore } from './inspection-data-store.service';
import { TimelineFilter } from './timeline-filter.service';
import { ParentRelationship } from '../generated';
import { ResourceTimeline } from '../store/timeline';
import { ViewStateService } from './view-state.service';

describe('TimelineFilter', () => {
  describe('kindTimelineFilter', () => {
    it('emit new kind timeline filter on setKindFilter', () => {
      const store = <InspectionDataStore>{};
      const availableKinds = new ReplaySubject<Set<string>>(1);
      store.availableKinds = availableKinds;
      store.allTimelines = NEVER;
      const filter = new TimelineFilter(store, new ViewStateService());
      const gotFilters: Set<string>[] = [];
      filter.kindTimelineFilter.subscribe((kinds) => {
        gotFilters.push(kinds);
      });

      availableKinds.next(new Set(['kind1', 'kind2', 'kind3']));
      filter.setKindFilter(new Set(['kind1', 'kind2']));

      expect(gotFilters).toEqual([
        new Set(['kind1', 'kind2', 'kind3']),
        new Set(['kind1', 'kind2']),
      ]);
    });
  });
  it('emit new kind timeline filter on the list of available kind names are changed', () => {
    const store = <InspectionDataStore>{};
    const availableKinds = new ReplaySubject<Set<string>>(1);
    store.availableKinds = availableKinds;
    store.allTimelines = NEVER;
    const filter = new TimelineFilter(store, new ViewStateService());
    const gotFilters: Set<string>[] = [];
    filter.kindTimelineFilter.subscribe((kinds) => {
      gotFilters.push(kinds);
    });

    availableKinds.next(new Set(['kind1', 'kind2', 'kind3']));
    filter.setKindFilter(new Set(['kind1', 'kind2']));
    availableKinds.next(new Set(['kind1', 'kind4', 'kind5']));

    expect(gotFilters).toEqual([
      new Set(['kind1', 'kind2', 'kind3']),
      new Set(['kind1', 'kind2']),
      new Set(['kind1', 'kind4', 'kind5']),
    ]);
  });

  it('emits kind timeline filter set before subscription', () => {
    const store = <InspectionDataStore>{};
    const availableKinds = new ReplaySubject<Set<string>>(1);
    store.availableKinds = availableKinds;
    store.allTimelines = NEVER;
    const filter = new TimelineFilter(store, new ViewStateService());
    const gotFilters: Set<string>[] = [];
    availableKinds.next(new Set(['kind1', 'kind2', 'kind3']));
    filter.setKindFilter(new Set(['kind1', 'kind2']));

    filter.kindTimelineFilter.subscribe((kinds) => {
      gotFilters.push(kinds);
    });

    expect(gotFilters).toEqual([new Set(['kind1', 'kind2'])]);
  });

  describe('namespaceTimelineFilter', () => {
    it('emit new namespace timeline filter on setNamespaceFilter', () => {
      const store = <InspectionDataStore>{};
      const availableNamespaces = new ReplaySubject<Set<string>>(1);
      store.allTimelines = NEVER;
      store.availableNamespaces = availableNamespaces;
      const filter = new TimelineFilter(store, new ViewStateService());
      const gotFilters: Set<string>[] = [];
      filter.namespaceTimelineFilter.subscribe((namespaces) => {
        gotFilters.push(namespaces);
      });

      availableNamespaces.next(new Set(['ns1', 'ns2', 'ns3']));
      filter.setNamespaceFilter(new Set(['ns1', 'ns2']));

      expect(gotFilters).toEqual([
        new Set(['ns1', 'ns2', 'ns3']),
        new Set(['ns1', 'ns2']),
      ]);
    });

    it('emit new namespace timeline filter on the list of available namespace names are changed', () => {
      const store = <InspectionDataStore>{};
      const availableNamespaces = new ReplaySubject<Set<string>>(1);
      store.allTimelines = NEVER;
      store.availableNamespaces = availableNamespaces;
      const filter = new TimelineFilter(store, new ViewStateService());
      const gotFilters: Set<string>[] = [];
      filter.namespaceTimelineFilter.subscribe((namespaces) => {
        gotFilters.push(namespaces);
      });

      availableNamespaces.next(new Set(['ns1', 'ns2', 'ns3']));
      filter.setNamespaceFilter(new Set(['ns1', 'ns2']));
      availableNamespaces.next(new Set(['ns1', 'ns4', 'ns5']));

      expect(gotFilters).toEqual([
        new Set(['ns1', 'ns2', 'ns3']),
        new Set(['ns1', 'ns2']),
        new Set(['ns1', 'ns4', 'ns5']),
      ]);
    });

    it('emits namespace timeline filter set before subscription', () => {
      const store = <InspectionDataStore>{};
      const availableNamespaces = new ReplaySubject<Set<string>>(1);
      store.allTimelines = NEVER;
      store.availableNamespaces = availableNamespaces;
      const filter = new TimelineFilter(store, new ViewStateService());
      const gotFilters: Set<string>[] = [];
      availableNamespaces.next(new Set(['ns1', 'ns2', 'ns3']));
      filter.setNamespaceFilter(new Set(['ns1', 'ns2']));

      filter.namespaceTimelineFilter.subscribe((namespaces) => {
        gotFilters.push(namespaces);
      });

      expect(gotFilters).toEqual([new Set(['ns1', 'ns2'])]);
    });
  });

  describe('subresourceParentRelationshipFilter', () => {
    it('emit new subresource parent relationship filter on setSubresourceParentRelationshipFilter', () => {
      const store = <InspectionDataStore>{};
      const availableSubresourceParentRelationships = new ReplaySubject<
        Set<ParentRelationship>
      >(1);
      store.allTimelines = NEVER;
      store.availableSubresourceParentRelationships =
        availableSubresourceParentRelationships;
      const filter = new TimelineFilter(store, new ViewStateService());
      const gotFilters: Set<ParentRelationship>[] = [];
      filter.subresourceParentRelationshipFilter.subscribe((relationships) => {
        gotFilters.push(relationships);
      });

      availableSubresourceParentRelationships.next(
        new Set([
          ParentRelationship.RelationshipChild,
          ParentRelationship.RelationshipPodBinding,
          ParentRelationship.RelationshipNodeComponent,
        ]),
      );
      filter.setSubresourceParentRelationshipFilter(
        new Set([
          ParentRelationship.RelationshipPodBinding,
          ParentRelationship.RelationshipNodeComponent,
        ]),
      );

      expect(gotFilters).toEqual([
        new Set([
          ParentRelationship.RelationshipChild,
          ParentRelationship.RelationshipPodBinding,
          ParentRelationship.RelationshipNodeComponent,
        ]),
        new Set([
          ParentRelationship.RelationshipPodBinding,
          ParentRelationship.RelationshipNodeComponent,
        ]),
      ]);
    });

    it('emit new subresource parent relationship filter on the list of available subresource parent relationships are changed', () => {
      const store = <InspectionDataStore>{};
      const availableSubresourceParentRelationships = new ReplaySubject<
        Set<ParentRelationship>
      >(1);
      store.allTimelines = NEVER;
      store.availableSubresourceParentRelationships =
        availableSubresourceParentRelationships;
      const filter = new TimelineFilter(store, new ViewStateService());
      const gotFilters: Set<ParentRelationship>[] = [];
      filter.subresourceParentRelationshipFilter.subscribe((relationships) => {
        gotFilters.push(relationships);
      });

      availableSubresourceParentRelationships.next(
        new Set([
          ParentRelationship.RelationshipChild,
          ParentRelationship.RelationshipPodBinding,
          ParentRelationship.RelationshipNodeComponent,
        ]),
      );
      filter.setSubresourceParentRelationshipFilter(
        new Set([
          ParentRelationship.RelationshipPodBinding,
          ParentRelationship.RelationshipNodeComponent,
        ]),
      );
      availableSubresourceParentRelationships.next(
        new Set([
          ParentRelationship.RelationshipContainer,
          ParentRelationship.RelationshipEndpointSlice,
        ]),
      );

      expect(gotFilters).toEqual([
        new Set([
          ParentRelationship.RelationshipChild,
          ParentRelationship.RelationshipPodBinding,
          ParentRelationship.RelationshipNodeComponent,
        ]),
        new Set([
          ParentRelationship.RelationshipPodBinding,
          ParentRelationship.RelationshipNodeComponent,
        ]),
        new Set([
          ParentRelationship.RelationshipContainer,
          ParentRelationship.RelationshipEndpointSlice,
        ]),
      ]);
    });

    it('emits subresource parent relationship filter set before subscription', () => {
      const store = <InspectionDataStore>{};
      const availableSubresourceParentRelationships = new ReplaySubject<
        Set<ParentRelationship>
      >(1);
      store.allTimelines = NEVER;
      store.availableSubresourceParentRelationships =
        availableSubresourceParentRelationships;
      const filter = new TimelineFilter(store, new ViewStateService());
      const gotFilters: Set<ParentRelationship>[] = [];
      availableSubresourceParentRelationships.next(
        new Set([
          ParentRelationship.RelationshipChild,
          ParentRelationship.RelationshipPodBinding,
          ParentRelationship.RelationshipNodeComponent,
        ]),
      );
      filter.setSubresourceParentRelationshipFilter(
        new Set([
          ParentRelationship.RelationshipPodBinding,
          ParentRelationship.RelationshipNodeComponent,
        ]),
      );

      filter.subresourceParentRelationshipFilter.subscribe((relationships) => {
        gotFilters.push(relationships);
      });

      expect(gotFilters).toEqual([
        new Set([
          ParentRelationship.RelationshipPodBinding,
          ParentRelationship.RelationshipNodeComponent,
        ]),
      ]);
    });
  });

  describe('resourceNameTimelineRegexFilter', () => {
    it('emit new resource name timeline regex filter on setResourceNameRegexFilter', () => {
      const store = <InspectionDataStore>{};
      const allTimelines = new ReplaySubject<ResourceTimeline[]>(1);
      store.allTimelines = allTimelines;
      const filter = new TimelineFilter(store, new ViewStateService());
      const gotFilters: string[] = [];
      filter.resourceNameTimelineRegexFilter.subscribe((regex) => {
        gotFilters.push(regex);
      });

      allTimelines.next([]);
      filter.setResourceNameRegexFilter('test');

      expect(gotFilters).toEqual(['', 'test']);
    });

    it('reset the filter when a new data loaded on the data store', () => {
      const store = <InspectionDataStore>{};
      const allTimelines = new ReplaySubject<ResourceTimeline[]>(1);
      store.allTimelines = allTimelines;
      const filter = new TimelineFilter(store, new ViewStateService());
      const gotFilters: string[] = [];
      filter.resourceNameTimelineRegexFilter.subscribe((regex) => {
        gotFilters.push(regex);
      });

      allTimelines.next([]);
      filter.setResourceNameRegexFilter('test');
      allTimelines.next([]);

      expect(gotFilters).toEqual(['', 'test', '']);
    });

    it('emits resource name timeline regex filter set before subscription', () => {
      const store = <InspectionDataStore>{};
      const allTimelines = new ReplaySubject<ResourceTimeline[]>(1);
      store.allTimelines = allTimelines;
      const filter = new TimelineFilter(store, new ViewStateService());
      const gotFilters: string[] = [];
      allTimelines.next([]);
      filter.setResourceNameRegexFilter('test');

      filter.resourceNameTimelineRegexFilter.subscribe((regex) => {
        gotFilters.push(regex);
      });

      expect(gotFilters).toEqual(['test']);
    });
  });

  describe('filterTimelines', () => {
    const timelines: ResourceTimeline[] = [
      new ResourceTimeline(
        'test',
        'apiVersion1#kind1',
        [],
        [],
        ParentRelationship.RelationshipChild,
      ),
      new ResourceTimeline(
        'test',
        'apiVersion1#kind1#namespace1',
        [],
        [],
        ParentRelationship.RelationshipChild,
      ),
      new ResourceTimeline(
        'test',
        'apiVersion1#kind1#namespace1#name1',
        [],
        [],
        ParentRelationship.RelationshipChild,
      ),
      new ResourceTimeline(
        'test',
        'apiVersion1#kind1#namespace1#name2',
        [],
        [],
        ParentRelationship.RelationshipChild,
      ),
      new ResourceTimeline(
        'test',
        'apiVersion1#kind1#namespace1#name2#subresource1',
        [],
        [],
        ParentRelationship.RelationshipChild,
      ),
      new ResourceTimeline(
        'test',
        'apiVersion1#kind1#namespace1#name2#subresource-bidning',
        [],
        [],
        ParentRelationship.RelationshipPodBinding,
      ),
      new ResourceTimeline(
        'test',
        'apiVersion1#kind1#namespace2',
        [],
        [],
        ParentRelationship.RelationshipChild,
      ),
      new ResourceTimeline(
        'test',
        'apiVersion1#kind1#namespace2#name2',
        [],
        [],
        ParentRelationship.RelationshipChild,
      ),
      new ResourceTimeline(
        'test',
        'apiVersion1#kind2',
        [],
        [],
        ParentRelationship.RelationshipChild,
      ),
      new ResourceTimeline(
        'test',
        'apiVersion1#kind2#namespace2',
        [],
        [],
        ParentRelationship.RelationshipChild,
      ),
      new ResourceTimeline(
        'test',
        'apiVersion1#kind2#namespace2#name3',
        [],
        [],
        ParentRelationship.RelationshipChild,
      ),
    ];

    let store: InspectionDataStore;
    let allTimelines: ReplaySubject<ResourceTimeline[]>;
    let availableKinds: ReplaySubject<Set<string>>;
    let availableNamespaces: ReplaySubject<Set<string>>;
    let availableSubresourceParentRelationships: ReplaySubject<
      Set<ParentRelationship>
    >;
    let filteredOutLogIndicesSet: ReplaySubject<Set<number>>;
    beforeEach(() => {
      store = <InspectionDataStore>{};
      allTimelines = new ReplaySubject<ResourceTimeline[]>(1);
      allTimelines.next(timelines);
      availableKinds = new ReplaySubject<Set<string>>(1);
      availableKinds.next(new Set(['kind1', 'kind2']));
      availableNamespaces = new ReplaySubject<Set<string>>(1);
      availableNamespaces.next(new Set(['namespace1', 'namespace2']));
      availableSubresourceParentRelationships = new ReplaySubject<
        Set<ParentRelationship>
      >(1);
      filteredOutLogIndicesSet = new ReplaySubject<Set<number>>(1);
      filteredOutLogIndicesSet.next(new Set());
      availableSubresourceParentRelationships.next(
        new Set([
          ParentRelationship.RelationshipChild,
          ParentRelationship.RelationshipPodBinding,
        ]),
      );
      store.allTimelines = allTimelines;
      store.availableKinds = availableKinds;
      store.availableNamespaces = availableNamespaces;
      store.availableSubresourceParentRelationships =
        availableSubresourceParentRelationships;
      store.filteredOutLogIndicesSet = filteredOutLogIndicesSet;
    });
    it('must emit filteredOutLogIndicesSet at first', (done) => {
      store.filteredOutLogIndicesSet.subscribe((set) => {
        console.log(set);
        done();
      });
    });
    it('should emit filter result on subscribe', (done) => {
      const filter = new TimelineFilter(store, new ViewStateService());
      filter.filteredTimeline.subscribe((timelines) => {
        expect(timelines).toEqual(timelines);
        done();
      });
    });

    it('filters timelines with regex filter', (done) => {
      const filter = new TimelineFilter(store, new ViewStateService());
      filter.filteredTimeline
        .pipe(debounceTime(10), take(1))
        .subscribe((timelines) => {
          expect(timelines).toEqual([
            new ResourceTimeline(
              'test',
              'apiVersion1#kind1',
              [],
              [],
              ParentRelationship.RelationshipChild,
            ),
            new ResourceTimeline(
              'test',
              'apiVersion1#kind1#namespace1',
              [],
              [],
              ParentRelationship.RelationshipChild,
            ),
            new ResourceTimeline(
              'test',
              'apiVersion1#kind1#namespace1#name1',
              [],
              [],
              ParentRelationship.RelationshipChild,
            ),
            new ResourceTimeline(
              'test',
              'apiVersion1#kind2',
              [],
              [],
              ParentRelationship.RelationshipChild,
            ),
            new ResourceTimeline(
              'test',
              'apiVersion1#kind2#namespace2',
              [],
              [],
              ParentRelationship.RelationshipChild,
            ),
            new ResourceTimeline(
              'test',
              'apiVersion1#kind2#namespace2#name3',
              [],
              [],
              ParentRelationship.RelationshipChild,
            ),
          ]);
          done();
        });
      filter.setResourceNameRegexFilter('name1|name3');
    });

    it('filters timelines with kind', (done) => {
      const filter = new TimelineFilter(store, new ViewStateService());
      filter.filteredTimeline
        .pipe(debounceTime(10), take(1))
        .subscribe((timelines) => {
          expect(timelines).toEqual([
            new ResourceTimeline(
              'test',
              'apiVersion1#kind2',
              [],
              [],
              ParentRelationship.RelationshipChild,
            ),
            new ResourceTimeline(
              'test',
              'apiVersion1#kind2#namespace2',
              [],
              [],
              ParentRelationship.RelationshipChild,
            ),
            new ResourceTimeline(
              'test',
              'apiVersion1#kind2#namespace2#name3',
              [],
              [],
              ParentRelationship.RelationshipChild,
            ),
          ]);
          done();
        });
      filter.setKindFilter(new Set(['kind2']));
    });

    it('filters result with namespace', (done) => {
      const filter = new TimelineFilter(store, new ViewStateService());
      filter.filteredTimeline
        .pipe(debounceTime(10), take(1))
        .subscribe((timelines) => {
          expect(timelines).toEqual([
            new ResourceTimeline(
              'test',
              'apiVersion1#kind1',
              [],
              [],
              ParentRelationship.RelationshipChild,
            ),
            new ResourceTimeline(
              'test',
              'apiVersion1#kind1#namespace2',
              [],
              [],
              ParentRelationship.RelationshipChild,
            ),
            new ResourceTimeline(
              'test',
              'apiVersion1#kind1#namespace2#name2',
              [],
              [],
              ParentRelationship.RelationshipChild,
            ),
            new ResourceTimeline(
              'test',
              'apiVersion1#kind2',
              [],
              [],
              ParentRelationship.RelationshipChild,
            ),
            new ResourceTimeline(
              'test',
              'apiVersion1#kind2#namespace2',
              [],
              [],
              ParentRelationship.RelationshipChild,
            ),
            new ResourceTimeline(
              'test',
              'apiVersion1#kind2#namespace2#name3',
              [],
              [],
              ParentRelationship.RelationshipChild,
            ),
          ]);
          done();
        });
      filter.setNamespaceFilter(new Set(['namespace2']));
    });

    it('filters result with parent relationship of subresource', (done) => {
      const filter = new TimelineFilter(store, new ViewStateService());
      filter.filteredTimeline
        .pipe(debounceTime(10), take(1))
        .subscribe((timelines) => {
          expect(timelines).toEqual([
            new ResourceTimeline(
              'test',
              'apiVersion1#kind1',
              [],
              [],
              ParentRelationship.RelationshipChild,
            ),
            new ResourceTimeline(
              'test',
              'apiVersion1#kind1#namespace1',
              [],
              [],
              ParentRelationship.RelationshipChild,
            ),
            new ResourceTimeline(
              'test',
              'apiVersion1#kind1#namespace1#name1',
              [],
              [],
              ParentRelationship.RelationshipChild,
            ),
            new ResourceTimeline(
              'test',
              'apiVersion1#kind1#namespace1#name2',
              [],
              [],
              ParentRelationship.RelationshipChild,
            ),
            new ResourceTimeline(
              'test',
              'apiVersion1#kind1#namespace1#name2#subresource-bidning',
              [],
              [],
              ParentRelationship.RelationshipPodBinding,
            ),
            new ResourceTimeline(
              'test',
              'apiVersion1#kind1#namespace2',
              [],
              [],
              ParentRelationship.RelationshipChild,
            ),
            new ResourceTimeline(
              'test',
              'apiVersion1#kind1#namespace2#name2',
              [],
              [],
              ParentRelationship.RelationshipChild,
            ),
            new ResourceTimeline(
              'test',
              'apiVersion1#kind2',
              [],
              [],
              ParentRelationship.RelationshipChild,
            ),
            new ResourceTimeline(
              'test',
              'apiVersion1#kind2#namespace2',
              [],
              [],
              ParentRelationship.RelationshipChild,
            ),
            new ResourceTimeline(
              'test',
              'apiVersion1#kind2#namespace2#name3',
              [],
              [],
              ParentRelationship.RelationshipChild,
            ),
          ]);
          done();
        });
      filter.setSubresourceParentRelationshipFilter(
        new Set([ParentRelationship.RelationshipPodBinding]),
      );
    });
  });
});
