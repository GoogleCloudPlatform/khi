import { Meta, StoryObj, moduleMetadata } from '@storybook/angular';

import { ManagedFieldTooltipComponent } from './managed-field-tooltip.component';

const meta: Meta<ManagedFieldTooltipComponent> = {
  title: 'Shared/YamlViewer/ManagedFieldTooltip',
  component: ManagedFieldTooltipComponent,
  decorators: [
    moduleMetadata({
      imports: [ManagedFieldTooltipComponent],
    }),
  ],
  argTypes: {
    manager: { control: 'text' },
    timezoneShift: { control: 'number' },
  },
};
export default meta;
type Story = StoryObj<ManagedFieldTooltipComponent>;

export const Default: Story = {
  args: {
    manager: 'kubectl-client-side-apply',
    time: 1696939200000000000n, // 2023-10-10T12:00:00Z in ns
    timezoneShift: 9,
  },
};
