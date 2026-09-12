import { Meta, StoryObj, moduleMetadata } from '@storybook/angular';
import { CheckboxParameterComponent } from './checkbox-parameter.component';
import {
  DefaultParameterStore,
  PARAMETER_STORE,
} from './service/parameter-store';
import {
  ParameterHintType,
  ParameterInputType,
} from 'src/app/common/schema/form-types';

const createParameterStore = (paramId: string, initialValue = false) => {
  const store = new DefaultParameterStore();
  store.setDefaultValues({ [paramId]: initialValue });
  return store;
};

const meta: Meta<CheckboxParameterComponent> = {
  title: 'Dialogs/NewInspection/CheckboxParameter',
  component: CheckboxParameterComponent,
  tags: ['autodocs'],
  decorators: [
    moduleMetadata({
      imports: [],
      providers: [
        {
          provide: PARAMETER_STORE,
          useValue: new DefaultParameterStore(),
        },
      ],
    }),
  ],
};

export default meta;
type Story = StoryObj<CheckboxParameterComponent>;

export const Default: Story = {
  args: {
    parameter: {
      id: 'test-checkbox-default',
      type: ParameterInputType.Checkbox,
      label: 'Enable Audit Log Ingestion',
      description: 'Ingest Kubernetes API audit logs into timeline.',
      hint: '',
      hintType: ParameterHintType.None,
      default: false,
      readonly: false,
    },
  },
  decorators: [
    moduleMetadata({
      providers: [
        {
          provide: PARAMETER_STORE,
          useValue: createParameterStore('test-checkbox-default', false),
        },
      ],
    }),
  ],
};

export const Checked: Story = {
  args: {
    parameter: {
      id: 'test-checkbox-checked',
      type: ParameterInputType.Checkbox,
      label: 'Collect Metrics',
      description: 'Collect node and pod CPU/Memory metrics.',
      hint: '',
      hintType: ParameterHintType.None,
      default: true,
      readonly: false,
    },
  },
  decorators: [
    moduleMetadata({
      providers: [
        {
          provide: PARAMETER_STORE,
          useValue: createParameterStore('test-checkbox-checked', true),
        },
      ],
    }),
  ],
};

export const WithWarningHint: Story = {
  args: {
    parameter: {
      id: 'test-checkbox-warning',
      type: ParameterInputType.Checkbox,
      label: 'Verbose Mode',
      description: 'Includes debug-level logs in the inspection output.',
      hint: 'Enabling this may increase log inspection time.',
      hintType: ParameterHintType.Warning,
      default: false,
      readonly: false,
    },
  },
  decorators: [
    moduleMetadata({
      providers: [
        {
          provide: PARAMETER_STORE,
          useValue: createParameterStore('test-checkbox-warning', false),
        },
      ],
    }),
  ],
};

export const Readonly: Story = {
  args: {
    parameter: {
      id: 'test-checkbox-readonly',
      type: ParameterInputType.Checkbox,
      label: 'System Mandatory Feature',
      description: 'This setting is enforced and cannot be changed.',
      hint: '',
      hintType: ParameterHintType.None,
      default: true,
      readonly: true,
    },
  },
  decorators: [
    moduleMetadata({
      providers: [
        {
          provide: PARAMETER_STORE,
          useValue: createParameterStore('test-checkbox-readonly', true),
        },
      ],
    }),
  ],
};

export const WithErrorHint: Story = {
  args: {
    parameter: {
      id: 'test-checkbox-error',
      type: ParameterInputType.Checkbox,
      label: 'Cloud Logging Export',
      description: 'Export inspection results to Google Cloud Logging.',
      hint: 'Service account does not have logging.logEntries.create permission.',
      hintType: ParameterHintType.Error,
      default: false,
      readonly: false,
    },
  },
  decorators: [
    moduleMetadata({
      providers: [
        {
          provide: PARAMETER_STORE,
          useValue: createParameterStore('test-checkbox-error', false),
        },
      ],
    }),
  ],
};
