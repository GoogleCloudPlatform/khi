---
trigger: glob
globs: **/*.proto
---

# Proto Coding Rules

When developing or modifying Proto files in the KHI project, you **must** adhere to the following rules and best practices.

## Verifications

1. **Linting**: Before submitting changes, you must verify that your proto files follow the style guide defined in `buf.yaml`.
   - Run `make lint-proto` to check for linting errors.
2. **Formatting**:
   - Run `make format-proto` to format the proto files.
3. **Breaking Changes**:
   - Run `make breaking-proto` (or the equivalent command defined in Makefile) to ensure no breaking changes are introduced against `main` branch.
4. **Code Generation**:
   - Run `make build-proto` to ensure that the generated code (Go, TS, etc.) can be created without errors.

## General Coding Rules

1. **File Naming**:
   - Proto files must be named in `snake_case.proto` (e.g., `intern_pool.proto`).
2. **Comments**:
   - All comments must be written in English.
   - **Focus on Intent**: Explain *why* a message, field, or enum exists, especially if the usage is not obvious. Do not simply restate the name in English.
   - **Avoid Redundancy**: Do not repeat information that is obvious from the name or type.
   - **Sentence Structure**: End all comments with with a period (.).
   - **Documentation**:
     - Messages, enums, and fields should be documented to explain their purpose.
     - Example (Good):

       ```proto
       // LogChunk aggregates multiple logs to optimize I/O performance during rendering.
       message LogChunk {
         // The logs are grouped by resource path to enable efficient timeline filtering.
         repeated Log logs = 1;
       }
       ```

3. **Package and Options**:
   - Package names must follow the pattern `khifile.vX` where X is the version (e.g., `khifile.v7`).
   - Always specify `go_package` option correctly, matching the project structure.
     - Example: `option go_package = "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v7;khifilev7";`

## Best Practices

1. **Backward Compatibility**:
   - Never change the type or field number of an existing field.
   - If a field is no longer needed, use `reserved` to prevent its number or name from being reused.
     - Example: `reserved 5;` or `reserved "old_field_name";`
2. **Field Naming**:
   - Use `snake_case` for field names.
3. **Message Naming**:
   - Use `CamelCase` for message names.
4. **Enum Naming and Rules**:
   - Use `CamelCase` for enum names.
   - Use `CAPS_WITH_UNDERSCORES` for enum values.
   - The first value of an enum (tag `0`) **must** be the default value and should represent an unspecified or unknown state (e.g., `NULL_VALUE = 0;` or `VERB_TYPE_UNKNOWN = 0;`).
   - Prefix enum values with the enum name (in uppercase) to avoid name collisions.
5. **Field Number Optimization**:
   - Use field numbers `1` through `15` for the most frequently used fields and `repeated` fields, as they take less space when encoded.
