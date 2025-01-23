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

import { ParsePrincipalPipe, PrincipalType } from './diff-view-pipes';

describe('ParsePrincipalPipe', () => {
  it('user account', () => {
    const ppp = new ParsePrincipalPipe();
    const principal = ppp.transform('foo@bar.com');
    expect(principal.type).toBe(PrincipalType.User);
    expect(principal.short).toBe('foo@bar.com');
    expect(principal.full).toBe('foo@bar.com');
  });
  it('system account', () => {
    const ppp = new ParsePrincipalPipe();
    const principal = ppp.transform('system:garbage-collector');
    expect(principal.type).toBe(PrincipalType.System);
    expect(principal.short).toBe('garbage-collector');
    expect(principal.full).toBe('system:garbage-collector');
  });
  it('node account', () => {
    const ppp = new ParsePrincipalPipe();
    const principal = ppp.transform('system:node:node-foo-bar');
    expect(principal.type).toBe(PrincipalType.Node);
    expect(principal.short).toBe('node-foo-bar');
    expect(principal.full).toBe('system:node:node-foo-bar');
  });
  it('service account', () => {
    const ppp = new ParsePrincipalPipe();
    const principal = ppp.transform('system:serviceaccount:argocd');
    expect(principal.type).toBe(PrincipalType.ServiceAccount);
    expect(principal.short).toBe('argocd');
    expect(principal.full).toBe('system:serviceaccount:argocd');
  });
});
