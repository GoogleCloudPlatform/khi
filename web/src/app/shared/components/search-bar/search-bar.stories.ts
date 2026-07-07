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

import { Meta, StoryObj } from '@storybook/angular';
import { SearchBarComponent } from 'src/app/shared/components/search-bar/search-bar.component';

export default {
  title: 'Shared/SearchBar',
  component: SearchBarComponent,
} as Meta<SearchBarComponent>;

type Story = StoryObj<SearchBarComponent>;

export const Default: Story = {
  args: {
    query: '',
    matchCount: 0,
    currentMatchIndex: 0,
  },
};

export const WithMatches: Story = {
  args: {
    query: 'error',
    matchCount: 5,
    currentMatchIndex: 2,
  },
};

export const NoMatches: Story = {
  args: {
    query: 'notfound',
    matchCount: 0,
    currentMatchIndex: 0,
  },
};
