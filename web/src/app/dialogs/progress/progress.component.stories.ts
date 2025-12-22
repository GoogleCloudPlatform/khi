import { Meta, StoryObj, moduleMetadata } from '@storybook/angular';
import { ProgressDialogComponent } from './progress.component';
import {
  PROGRESS_DIALOG_STATUS_OBSERVER,
  CurrentProgress,
} from 'src/app/services/progress/progress-interface';
import { of } from 'rxjs';

const createProgressDecorator = (progress: CurrentProgress) => moduleMetadata({
  providers: [
    {
      provide: PROGRESS_DIALOG_STATUS_OBSERVER,
      useValue: {
        status: () => of(progress as CurrentProgress),
      },
    },
  ],
});


const meta: Meta<ProgressDialogComponent> = {
  title: 'Dialogs/ProgressDialog',
  component: ProgressDialogComponent,
  tags: ['autodocs'],
  decorators: [
    createProgressDecorator({
      mode: 'indeterminate',
      percent: 0,
      message: 'Loading resources...',
    } as CurrentProgress),
  ],
};

export default meta;
type Story = StoryObj<ProgressDialogComponent>;

export const Indeterminate: Story = {};

export const DeterminateStart: Story = {
  decorators: [
    createProgressDecorator({
      mode: 'determinate',
      percent: 0,
      message: 'Starting download...',
    } as CurrentProgress),
  ],
};

export const DeterminateHalf: Story = {
  decorators: [
    createProgressDecorator({
      mode: 'determinate',
      percent: 50,
      message: 'Processing data (50%)...',
    } as CurrentProgress),
  ],
};

export const DeterminateComplete: Story = {
  decorators: [
    createProgressDecorator({
      mode: 'determinate',
      percent: 100,
      message: 'Completed!',
    } as CurrentProgress),
  ],
};