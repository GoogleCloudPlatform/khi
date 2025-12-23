/**
 * Copyright 2025 Google LLC
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

/**
 * The type of parameter form field.
 */
export enum ParameterInputType {
  Group = 'group',
  Text = 'text',
  File = 'file',
  Set = 'set',
}

/**
 * Types of hint message shown at the bottom of parameter forms.
 */
export enum ParameterHintType {
  None = 'none',
  Error = 'error',
  Warning = 'warning',
  Info = 'info',
}

/**
 * The base type of parameter form fields.
 */
export interface ParameterFormFieldBase {
  /**
   * Unique ID of this parameter field.
   */
  id: string;
  /**
   * Type of this parameter field.
   */
  type: ParameterInputType;
  /**
   * Label of the parameter.
   */
  label: string;
  /**
   * Description of this parameter.
   */
  description: string;
  /**
   * Type of hint message (error, warning or info)
   */
  hintType?: ParameterHintType;
  /**
   * The hint message shown at the bottom of inputs.
   */
  hint: string;
}

/**
 * Group type parameter specific data.
 */
export interface GroupParameterFormField extends ParameterFormFieldBase {
  type: ParameterInputType.Group;

  /**
   * List of child parameters.
   */
  children: ParameterFormField[];

  /**
   * If this group is collapsible or not.
   */
  collapsible: boolean;

  /**
   * If this group is collapsed by default.
   * `collapsible` must be true when this value is true.
   */
  collapsedByDefault: boolean;
}

/**
 * Timings to validate the parameter input form.
 */
export enum ParameterFormValidationTiming {
  /**
   * Triggers validations on every input value change.
   */
  Change = 'change',
  /**
   * Triggers validations on blur.
   */
  Blur = 'blur',
}

/**
 * Text type parameter specific data.
 */
export interface TextParameterFormField extends ParameterFormFieldBase {
  type: ParameterInputType.Text;
  /**
   * If this text form field is readonly or not.
   */
  readonly: boolean;
  /**
   * The default value of this text form field.
   */
  default: string;

  /**
   * List of strings popped up as the autocomplete list.
   */
  suggestions: string[];

  /**
   * Type of the validation timing of this field.
   */
  validationTiming: ParameterFormValidationTiming;
}

/**
 * The identifier used for uploading file.
 */
export interface UploadToken {
  id: string;
}

/**
 * The types of UploadStatus given from the backend.
 */
export enum UploadStatus {
  Waiting = 0,
  Uploading = 1,
  Verifying = 2,
  Done = 3,
}

/**
 * File type parameter specific data.
 */
export interface FileParameterFormField extends ParameterFormFieldBase {
  type: ParameterInputType.File;
  /**
   * The unqiue token to be used for uploading the target file.
   */
  token: UploadToken;

  /**
   * The status of file reported from the server side.
   */
  status: UploadStatus;
}
/**
 * An option item of set type parameter.
 */
export interface SetParameterFormFieldOptionItem {
  /**
   * A unique value and the value used as the label of the option.
   */
  id: string;
  /**
   * The description of the option.
   */
  description: string;
}

/**
 * Set type parameter specific data.
 */
export interface SetParameterFormField extends ParameterFormFieldBase {
  type: ParameterInputType.Set;
  /**
   * List of available options.
   */
  options: SetParameterFormFieldOptionItem[];

  /**
   * Default selected values.
   */
  default: string[];

  /**
   * If the "Add All" button is enabled or not.
   */
  allowAddAll: boolean;

  /**
   * If the "Remove All" button is enabled or not.
   */
  allowRemoveAll: boolean;

  /**
   * If the text input allows custom values or not.
   */
  allowCustomValue: boolean;
}

export type ParameterFormField =
  | GroupParameterFormField
  | TextParameterFormField
  | FileParameterFormField
  | SetParameterFormField;
