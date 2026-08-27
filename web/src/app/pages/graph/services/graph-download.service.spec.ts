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

import { TestBed } from '@angular/core/testing';
import { GraphDownloadService } from './graph-download.service';
import { GraphRenderer } from '../architecture-graph/graph/renderer';

describe('GraphDownloadService', () => {
  let service: GraphDownloadService;
  let mockRenderer: jasmine.SpyObj<GraphRenderer>;

  beforeEach(() => {
    mockRenderer = jasmine.createSpyObj('GraphRenderer', [
      'downloadSvg',
      'downloadPng',
    ]);
    TestBed.configureTestingModule({
      providers: [GraphDownloadService],
    });
    service = TestBed.inject(GraphDownloadService);
    service.registerRenderer(mockRenderer);
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });

  it('should delegate downloadSvg to registered renderer with custom filename', () => {
    service.downloadSvg('custom.svg');
    expect(mockRenderer.downloadSvg).toHaveBeenCalledWith('custom.svg');
  });

  it('should delegate downloadSvg without filename to renderer default', () => {
    service.downloadSvg();
    expect(mockRenderer.downloadSvg).toHaveBeenCalledWith(undefined);
  });

  it('should delegate downloadPng to registered renderer with custom filename', () => {
    service.downloadPng('custom.png');
    expect(mockRenderer.downloadPng).toHaveBeenCalledWith('custom.png');
  });

  it('should delegate downloadPng without filename to renderer default', () => {
    service.downloadPng();
    expect(mockRenderer.downloadPng).toHaveBeenCalledWith(undefined);
  });

  it('should safely do nothing if renderer is not registered', () => {
    const unregService = new GraphDownloadService();
    expect(() => {
      unregService.downloadSvg();
      unregService.downloadPng();
    }).not.toThrow();
  });
});
