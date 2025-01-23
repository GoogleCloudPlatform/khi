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

import { Pipe, PipeTransform } from '@angular/core';

export enum PrincipalType {
  System = 'System',
  Node = 'Node',
  ServiceAccount = 'SA',
  User = 'User',
  Invalid = 'Invalid',
  NotAvailable = 'N/A',
}

export interface ResourceOperatorPrincipal {
  type: PrincipalType;
  full: string;
  short: string;
}

/**
 * Parse the principal string modifying K8s resource into structured representation
 */
@Pipe({
  name: 'parsePrincipal',
})
export class ParsePrincipalPipe implements PipeTransform {
  transform(value: string): ResourceOperatorPrincipal {
    const result: ResourceOperatorPrincipal = {
      type: PrincipalType.User,
      full: value,
      short: value,
    };
    if (value === '') {
      result.type = PrincipalType.NotAvailable;
      result.full = '';
      result.short = '';
    }
    if (value.startsWith('system:serviceaccount:')) {
      result.type = PrincipalType.ServiceAccount;
      result.short = value.split('system:serviceaccount:')[1];
    } else if (value.startsWith('system:node:')) {
      result.type = PrincipalType.Node;
      result.short = value.split('system:node:')[1];
    } else if (value.startsWith('system:')) {
      result.type = PrincipalType.System;
      result.short = value.split('system:')[1];
    }
    return result;
  }
}
