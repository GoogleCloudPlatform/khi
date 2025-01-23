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

import { concatMap, of, toArray } from 'rxjs';
import {
  KHIFileReferenceResolver,
  NullReferenceResolver,
  ReferenceResolverStore,
} from './reference-resolver';
import { ToTextReferenceFromKHIFileBinary } from './reference-type';
import { ReferenceType, SingleTypeReferenceResolver } from './interface';

describe('ReferenceReslverStore', () => {
  describe('getText', () => {
    it('should return the valid text and complete with seaching multiple stores', (done) => {
      const resolver1 = jasmine.createSpyObj<SingleTypeReferenceResolver>(
        'KHIFileReferenceResolver',
        ['getText', 'isSupportedReferenceType'],
      );
      const resolver2 = jasmine.createSpyObj<SingleTypeReferenceResolver>(
        'NullReferenceResolver',
        ['getText', 'isSupportedReferenceType'],
      );
      resolver1.getText.and.returnValue(of('foo'));
      resolver1.isSupportedReferenceType.and.callFake(
        (type) => type === ReferenceType.KHIFileBinary,
      );
      resolver2.getText.and.returnValue(of('bar'));
      resolver2.isSupportedReferenceType.and.callFake(
        (type) => type === ReferenceType.NullReference,
      );
      const store = new ReferenceResolverStore([resolver1, resolver2]);

      of(
        { type: ReferenceType.KHIFileBinary },
        { type: ReferenceType.NullReference },
      )
        .pipe(
          concatMap((ref) => store.getText(ref)),
          toArray(),
        )
        .subscribe((result) => {
          expect(result).toEqual(['foo', 'bar']);
          done();
        });
    });
  });
});

describe('KHIFileReferenceResolver', () => {
  describe('getText', () => {
    it('should return the valid text and complete', (done) => {
      const encoder = new TextEncoder();
      const firstText = encoder.encode('foo');
      const secondText = encoder.encode('bar');
      const thirdText = encoder.encode('baz');
      const secondBufferFirstText = encoder.encode('qux');
      const secondBufferSecondText = encoder.encode('quux');
      const sourceBuffer1 = new Uint8Array(
        firstText.length + secondText.length + thirdText.length,
      );
      sourceBuffer1.set(firstText, 0);
      sourceBuffer1.set(secondText, firstText.length);
      sourceBuffer1.set(thirdText, firstText.length + secondText.length);
      const sourceBuffer2 = new Uint8Array(
        secondBufferFirstText.length + secondBufferSecondText.length,
      );
      sourceBuffer2.set(secondBufferFirstText, 0);
      sourceBuffer2.set(secondBufferSecondText, secondBufferFirstText.length);

      const resolver = new KHIFileReferenceResolver([
        sourceBuffer1.buffer,
        sourceBuffer2.buffer,
      ]);

      of(
        {
          buffer: 0,
          offset: 0,
          len: firstText.length,
        },
        {
          buffer: 0,
          offset: firstText.length,
          len: secondText.length,
        },
        {
          buffer: 0,
          offset: firstText.length + secondText.length,
          len: thirdText.length,
        },
        {
          buffer: 1,
          offset: 0,
          len: secondBufferFirstText.length,
        },
        {
          buffer: 1,
          offset: secondBufferFirstText.length,
          len: secondBufferSecondText.length,
        },
      )
        .pipe(
          concatMap((r) =>
            resolver.getText(ToTextReferenceFromKHIFileBinary(r)),
          ),
          toArray(),
        )
        .subscribe((result) => {
          expect(result).toEqual(['foo', 'bar', 'baz', 'qux', 'quux']);
          done();
        });
    });
  });
});

describe('NullReferenceResolver', () => {
  describe('getText', () => {
    it('should return empty string and complete', (done) => {
      const resolver = new NullReferenceResolver();
      resolver.getText().subscribe((result) => {
        expect(result).toEqual('');
        done();
      });
    });
  });
});
