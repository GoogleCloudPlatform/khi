# KHI File Format v6 Specification

## 1. Motivation

KHI processes logs on the backend, outputs the result as a single file, and visualizes it by loading this file on the frontend. This file-based approach has the following advantages:

* **Improved log data portability**: It is usually difficult to save huge logs that can fully capture the status at the time of a failure and look back at them later. By saving as a KHI file, past cases can be easily reviewed, which helps in creating post-mortems and preventing failure recurrence.
* **Interactive visualization**: The frontend handles the final display and analysis of logs. Even for large logs, using a single efficiently formatted file enables sufficient analysis on the frontend and achieves fast performance.

### 1.1 What KHI Does Not Aim For

The following are not goals in the design of the KHI file format:

* **Infinite scalability**: We optimize to handle as large logs as possible, but we do not adopt a client-server model design that stores logs in a backend system like a DB and sacrifices latency for infinite scalability. KHI is a tool for analyzing a sufficiently small time window when a problem occurs, not for visualizing long periods.
* **Backward compatibility with v5 and earlier**: This format v6 is a major update that is documented openly for the first time. Since it involves fundamental architectural changes, it is not backward compatible with earlier file formats.

### 1.2 Challenges of the Existing File Format

The KHI file format v5 had the following challenges in terms of maintainability and scalability:

* **JSON metadata size limit**: The previous format had a hybrid structure of a JSON metadata section and gzipped binary sections. Due to the 500MB string size limit in Chrome's JavaScript runtime, the JSON part alone could exceed this limit when inspecting a large-scale cluster, causing parsing failures.
* **Complex schema management**: The file schema interface was manually managed between the frontend and backend, making synchronization and maintenance difficult.
* **Tightly coupled style information**: Frontend style information, such as color definitions and icons, depended on enum variables defined in the backend and was generated at build time. This easily broke backward compatibility when adding new log types or states, making it difficult to extend KHI with plugins.
* **Frontend parser constraints**: Timestamps were handled as Number types, which lacked precision. Additionally, it was not designed with backward compatibility in mind.
* **Protobuf technical limitations**: When migrating to Protobuf, we had to consider the 2GB hard limit on a single string field or the entire file, as well as the 64MB size limit for the parser.
* **Lack of documentation**: There was no official specification document for the file format, which was a barrier to development and maintenance.

## 2. Goals

This format v6 is designed to solve the above problems and achieve the following goals:

* **Improve robustness and scalability**:
  * Migrate all data, including metadata, to Protobuf.
  * Split data into multiple chunks to support huge files and adopt a structure suitable for streaming analysis.
* **Self-containment and expressive power**:
  * Change style data, such as SCSS and color definitions, from build-time generation to direct embedding into the KHI file, enabling representation independent of specific frontend versions.
* **Improve precision and strict schema**:
  * Process timestamps in nanosecond precision.
* **Ensure extensibility**:
  * Adopt a container structure to easily append new data, such as comments or logs, to the end of the file without breaking existing data.

## 3. Container Structure

The KHI file is not a single massive Protobuf message, but rather a binary structure consisting of concatenated multiple chunk data built by Protobuf.

| Offset | Byte Size | Content | Description |
| :--- | :--- | :--- | :--- |
| `0x00` | 1 | Constant "K" (ASCII 75) | Magic bytes to identify the KHI file. |
| `0x01` | 1 | Constant "H" (ASCII 72) | |
| `0x02` | 1 | Constant "I" (ASCII 73) | |
| `0x03` | 1 | Version number `0x06` | The file schema version of the container. This value remains unchanged even if the internal Protobuf schema changes. |

After the header above, the following chunk blocks are repeated until EOF.

| Offset | Byte Size | Content | Description |
| :--- | :--- | :--- | :--- |
| `0x04 + offset` | 4 | Chunk size | The compressed chunk byte length (32-bit unsigned integer). |
| `0x08 + offset` | 4 | Chunk type | The type of chunk data (32-bit unsigned integer). |
| `0x0C + offset` | `length` bytes | Binary chunk data | Binary message built by Protobuf and compressed with gzip. |

*After reading one chunk, the next offset will be `current_offset + 8 + chunk_size`*

* **Endianness**: All integers and floating-point numbers are Little Endian.

## 4. Chunk Structure

Each chunk is a Protobuf binary. The root message type of a chunk depends on the "Chunk type" specified in the container structure.

| Chunk Type Name | Type ID | Root Message Type | Reference to Definition File |
| :--- | :--- | :--- | :--- |
| File Metadata Chunk | 1 | `MetadataChunk` | [metadata.proto](../../../proto/khifile/v6/metadata.proto) |
| Interning Pool Chunk | 2 | `InterningPoolChunk` | [intern_pool.proto](../../../proto/khifile/v6/intern_pool.proto) |
| Log Chunk | 3 | `LogChunk` | [log.proto](../../../proto/khifile/v6/log.proto) |
| Timeline Style Chunk | 4 | `TimelineStyleChunk` | [style.proto](../../../proto/khifile/v6/style.proto) |
| Timeline Chunk | 5 | `TimelineChunk` | [timeline.proto](../../../proto/khifile/v6/timeline.proto) |

### 4.1 Chunk Shared Rules

* **No Order Guarantee**: The occurrence order of chunks is not restricted in this file structure. Parsers must load all inspection data into memory or stream it sequentially, considering dependencies, to correctly resolve references, such as string IDs and style IDs, appearing in the file.
* **Size Limit and Chunk Splitting**: To avoid the Protobuf parser size limit, each chunk is automatically split into multiple chunks so that one chunk does not exceed 64MB before compression. Chunks of the same type may appear multiple times in a single file.
* **ID Allocation and Orphaned IDs**: Various IDs, such as string IDs and style IDs, start from 1. The ID value `0` is reserved for "no reference" (NULL). Due to parallel processing on the backend, some IDs might be allocated but never used. Therefore, there is no guarantee that the allocated ID values are completely contiguous.

### 4.2 Interning Pool Chunk Optimization

[InterningPoolChunk](../../../proto/khifile/v6/intern_pool.proto) stores string data by interning it to eliminate duplication. This chunk adopts the following mechanisms to optimize data size and compression ratio:

* **Compression Efficiency via String Sorting**: When outputting the string list, strings are sorted by their values in dictionary order, not by the numerical order of IDs. When a chunk is split every 64MB, this sorting groups similar strings together within the same chunk, dramatically increasing the compression ratio for compression algorithms like gzip.
* **Structured Data Flattening (InternedStruct)**: When serializing nested structured map data, such as log bodies, key paths are joined with a null character (`\x00`) and converted to a flat key set (`FieldPathSet`). A unique ID (`FieldPathSetID`) is allocated to the unique key set and managed in the pool. Values are stored in the `Values` list in the order of the flattened keys. This heavily deduplicates the schema definitions of structured data and significantly reduces the file size.
  * **Example 1: Nested Maps and Lists**

    ```yaml
    map:
      key: true
    list:
      - null
      - hello
    ```

    This data is serialized as `InternedStruct` as follows:
    1. **Key Flattening**: Travesing the map key structure, `key` under `map` is joined as `map\x00key`. The root keys become a set with the order `["map\x00key", "list"]`, and a `FieldPathSetID` is allocated.
    2. **Value Listing**: Values are stored in the `Values` list in the order of the key set above.
       * Index 0 (corresponding to `map\x00key`): `BoolValue(true)`
       * Index 1 (corresponding to `list`): `ListValue` containing `[NullValue(), StringValue(ID of "hello" in the pool)]`
  * **Example 2: Structs in a List**
    If a map exists as an element of a list, its keys are not flattened to the parent but treated as an independent `InternedStruct`.

    ```yaml
    list:
      - a: 1
        b: 2
    ```

    1. The root key becomes `["list"]`.
    2. A `ListValue` is stored at index 0 of the `Values` list.
    3. The first element of that list becomes a `StructValue`, which has its own independent key set `["a", "b"]` and values `[Int64Value(1), Int64Value(2)]`.

### 4.3 File Metadata Chunk Rules

The `MetadataChunk` contains supplementary information generated at file creation time, such as the total size of the analyzed logs, file names, and log filters used, rather than depending on the timeline information itself. Since these metadata schemas may expand in the future, it uses a message type defined with `oneof` fields in the Proto schema.

If multiple `MetadataChunk` messages are present in a single file, they are expected to contain different types of `Metadata`. Since a single `Metadata` message cannot be split during serialization, the size of any single `Metadata` message is limited by the 64MB Protobuf parser constraint.

### 4.4 Timeline Style Chunk Rules

The `TimelineStyleChunk` contains visual style definitions needed for the frontend to render the timeline, including icon maps, color schemes, and labels.
This chunk is not expected to have multiple `TimelineStyleChunk` messages in a single file. Therefore, the total size of this chunk must remain under the 64MB Protobuf limit.

### 4.5 Merging Rules for Other Chunks

Except for `MetadataChunk` and `TimelineStyleChunk`, all other chunks may be split and provided across multiple chunks of the same type if their data size exceeds the 64MB limit.

The root messages of these chunks contain only `repeated` fields. Therefore, the parser can fully reconstruct the original unified messages simply by concatenating the elements of the same fields retrieved from all chunks of that type.
