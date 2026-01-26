# File Operation Nodes

File operation nodes read and write files on the filesystem.

## Read Binary File Node

Read files from the filesystem with encoding options and metadata.

### Type

```
n8n-nodes-base.readBinaryFile
```

### Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `filePath` | string | Yes | - | Path to file (supports expressions) |
| `encoding` | string | No | `binary` | Output encoding |
| `includeHashes` | boolean | No | `false` | Calculate file hashes |

### Encoding Options

| Encoding | Description |
|----------|-------------|
| `binary` | Raw binary data |
| `base64` | Base64 encoded |
| `hex` | Hexadecimal encoded |
| `utf8` | UTF-8 text |
| `text` | Plain text |

### Examples

#### Read Text File

```json
{
  "id": "read-1",
  "name": "Read Config",
  "type": "n8n-nodes-base.readBinaryFile",
  "position": [450, 300],
  "parameters": {
    "filePath": "/app/config/settings.json",
    "encoding": "utf8"
  }
}
```

#### Read with Dynamic Path

```json
{
  "type": "n8n-nodes-base.readBinaryFile",
  "parameters": {
    "filePath": "/data/uploads/{{ $json.filename }}",
    "encoding": "base64"
  }
}
```

#### Read with Hashes

```json
{
  "type": "n8n-nodes-base.readBinaryFile",
  "parameters": {
    "filePath": "/data/document.pdf",
    "encoding": "binary",
    "includeHashes": true
  }
}
```

### Output

```json
{
  "json": {
    "data": "file content or encoded data",
    "fileName": "settings.json",
    "fileSize": 1024,
    "mimeType": "application/json",
    "modifiedTime": "2024-01-26T10:00:00Z",
    "hashes": {
      "md5": "abc123...",
      "sha1": "def456...",
      "sha256": "ghi789...",
      "sha512": "jkl012..."
    }
  }
}
```

| Field | Description |
|-------|-------------|
| `data` | File content (in requested encoding) |
| `fileName` | File name |
| `fileSize` | Size in bytes |
| `mimeType` | Detected MIME type |
| `modifiedTime` | Last modification time |
| `hashes` | File hashes (if requested) |

### Security

The node enforces security restrictions:

- **Directory restrictions** - Only reads from allowed directories
- **Size limits** - Default 100MB maximum
- **Path traversal protection** - Prevents `../` attacks

---

## Write Binary File Node

Write data to files on the filesystem.

### Type

```
n8n-nodes-base.writeBinaryFile
```

### Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `filePath` | string | Yes | - | Destination path |
| `dataSource` | string | No | `binary` | Data source type |
| `encoding` | string | No | `auto` | Input encoding |
| `createDirectory` | boolean | No | `false` | Create parent dirs |
| `overwrite` | boolean | No | `false` | Overwrite existing |
| `keepOriginalData` | boolean | No | `false` | Preserve input data |

### Data Source Options

| Source | Description |
|--------|-------------|
| `binary` | Binary data from input |
| `json` | JSON data to write |
| `expression` | Value from expression |

### Encoding Options

| Encoding | Description |
|----------|-------------|
| `auto` | Auto-detect |
| `utf8` | UTF-8 text |
| `base64` | Decode from base64 |
| `hex` | Decode from hex |

### Examples

#### Write JSON File

```json
{
  "id": "write-1",
  "name": "Save Results",
  "type": "n8n-nodes-base.writeBinaryFile",
  "position": [450, 300],
  "parameters": {
    "filePath": "/data/output/results.json",
    "dataSource": "json",
    "encoding": "utf8",
    "createDirectory": true,
    "overwrite": true
  }
}
```

#### Write with Dynamic Name

```json
{
  "type": "n8n-nodes-base.writeBinaryFile",
  "parameters": {
    "filePath": "/data/exports/{{ $json.reportName }}_{{ $now.toISOString().split('T')[0] }}.csv",
    "dataSource": "expression",
    "content": "={{ $json.csvContent }}",
    "encoding": "utf8",
    "createDirectory": true
  }
}
```

#### Write Base64 Image

```json
{
  "type": "n8n-nodes-base.writeBinaryFile",
  "parameters": {
    "filePath": "/data/images/{{ $json.imageId }}.png",
    "dataSource": "expression",
    "content": "={{ $json.base64Image }}",
    "encoding": "base64",
    "overwrite": true
  }
}
```

#### Preserve Original Data

```json
{
  "type": "n8n-nodes-base.writeBinaryFile",
  "parameters": {
    "filePath": "/data/backup/{{ $json.id }}.json",
    "dataSource": "json",
    "encoding": "utf8",
    "keepOriginalData": true
  }
}
```

### Output

```json
{
  "json": {
    "filePath": "/data/output/results.json",
    "fileSize": 2048,
    "bytesWritten": 2048,
    "timestamp": "2024-01-26T10:00:00Z"
  }
}
```

| Field | Description |
|-------|-------------|
| `filePath` | Written file path |
| `fileSize` | Total file size |
| `bytesWritten` | Bytes written |
| `timestamp` | Write timestamp |

### Security

Same restrictions as Read node:

- **Directory restrictions** - Only writes to allowed directories
- **Size limits** - Maximum file size enforced
- **Path traversal protection** - Prevents `../` attacks

---

## Common Patterns

### Read, Process, Write

```json
[
  {
    "type": "n8n-nodes-base.readBinaryFile",
    "parameters": {
      "filePath": "/data/input.json",
      "encoding": "utf8"
    }
  },
  {
    "type": "n8n-nodes-base.code",
    "parameters": {
      "code": "const data = JSON.parse(items[0].json.data);\ndata.processed = true;\nreturn [{json: {content: JSON.stringify(data, null, 2)}}];"
    }
  },
  {
    "type": "n8n-nodes-base.writeBinaryFile",
    "parameters": {
      "filePath": "/data/output.json",
      "dataSource": "expression",
      "content": "={{ $json.content }}"
    }
  }
]
```

### Backup Files

```json
{
  "type": "n8n-nodes-base.writeBinaryFile",
  "parameters": {
    "filePath": "/backups/{{ $json.fileName }}_{{ Date.now() }}.bak",
    "dataSource": "binary",
    "createDirectory": true
  }
}
```

### Check File Exists

Use a Try/Catch pattern:

```json
{
  "type": "n8n-nodes-base.code",
  "parameters": {
    "code": "try {\n  // Read file\n  return items;\n} catch(e) {\n  return [{json: {exists: false}}];\n}"
  }
}
```

---

## Quick Reference

| Node | Type | Direction |
|------|------|-----------|
| Read Binary File | `n8n-nodes-base.readBinaryFile` | File → Workflow |
| Write Binary File | `n8n-nodes-base.writeBinaryFile` | Workflow → File |

### File Size Limits

| Environment | Default Limit |
|-------------|---------------|
| Default | 100 MB |
| Configurable | Via server settings |

### Supported Paths

| Path Type | Example |
|-----------|---------|
| Absolute | `/data/file.txt` |
| Relative | `./data/file.txt` |
| Expression | `{{ $json.path }}` |
