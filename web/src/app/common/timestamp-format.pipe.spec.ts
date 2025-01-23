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

import { of } from 'rxjs';
import {
  LongTimestampFormatPipe,
  TimestampFormatPipe,
} from './timestamp-format.pipe';
import { ViewStateService } from '../services/view-state.service';

function generateViewStateServiceWithTimeShift(
  timeshift: number,
): ViewStateService {
  return {
    timezoneShift: of(timeshift),
  } as unknown as ViewStateService;
}

describe('TimestampFormatPipe', () => {
  it('create an instance', () => {
    const pipe = new TimestampFormatPipe(
      generateViewStateServiceWithTimeShift(0),
    );
    expect(pipe).toBeTruthy();
  });
  it('should timestamp transform works valid', () => {
    const pipe = new TimestampFormatPipe(
      generateViewStateServiceWithTimeShift(0),
    );
    const time = new Date('2022-03-04T17:06:07.000+00:00');
    const date = pipe.transform(time.getTime());
    date.subscribe((d) => expect(d).toBe('17:06:07'));
  });
  it('should timestamp transform works valid with positive time offset', () => {
    const pipe = new TimestampFormatPipe(
      generateViewStateServiceWithTimeShift(3),
    );
    const time = new Date('2022-03-04T17:06:07.000+00:00');
    const date = pipe.transform(time.getTime());
    date.subscribe((d) => expect(d).toBe('20:06:07'));
  });

  it('should timestamp transform works valid with negative time offset', () => {
    const pipe = new TimestampFormatPipe(
      generateViewStateServiceWithTimeShift(-5),
    );
    const time = new Date('2022-03-04T17:06:07.000+00:00');
    let date = pipe.transform(time.getTime());
    date = pipe.transform(time.getTime());
    date.subscribe((d) => expect(d).toBe('12:06:07'));
  });
  it('should timestamp transform works valid with float point offset', () => {
    const pipe = new TimestampFormatPipe(
      generateViewStateServiceWithTimeShift(1.5),
    );
    const time = new Date('2022-03-04T17:06:07.000+00:00');
    let date = pipe.transform(time.getTime());
    date = pipe.transform(time.getTime());
    date.subscribe((d) => expect(d).toBe('18:36:07'));
  });
});

describe('LongTimestampFormatPipe', () => {
  it('create an instance', () => {
    const pipe = new LongTimestampFormatPipe(
      generateViewStateServiceWithTimeShift(0),
    );
    expect(pipe).toBeTruthy();
  });
  it('should timestamp transform works valid', () => {
    const pipe = new LongTimestampFormatPipe(
      generateViewStateServiceWithTimeShift(0),
    );
    const time = new Date('2022-03-04T17:06:07.800+00:00');
    const date = pipe.transform(time.getTime());
    date.subscribe((d) => expect(d).toBe('2022-03-04T17:06:07.800+00:00'));
  });
  it('should timestamp transform works valid with positive time offset', () => {
    const pipe = new LongTimestampFormatPipe(
      generateViewStateServiceWithTimeShift(3),
    );
    const time = new Date('2022-03-04T17:06:07.800+00:00');
    const date = pipe.transform(time.getTime());
    date.subscribe((d) => expect(d).toBe('2022-03-04T20:06:07.800+03:00'));
  });

  it('should timestamp transform works valid with negative time offset', () => {
    const pipe = new LongTimestampFormatPipe(
      generateViewStateServiceWithTimeShift(-5),
    );
    const time = new Date('2022-03-04T17:06:07.800+00:00');
    let date = pipe.transform(time.getTime());
    date = pipe.transform(time.getTime());
    date.subscribe((date) =>
      expect(date).toBe('2022-03-04T12:06:07.800-05:00'),
    );
  });
  it('should timestamp transform works valid with float point offset', () => {
    const pipe = new LongTimestampFormatPipe(
      generateViewStateServiceWithTimeShift(1.5),
    );
    const time = new Date('2022-03-04T17:06:07.800+00:00');
    let date = pipe.transform(time.getTime());
    date = pipe.transform(time.getTime());
    date.subscribe((date) =>
      expect(date).toBe('2022-03-04T18:36:07.800+01:30'),
    );
  });
});
