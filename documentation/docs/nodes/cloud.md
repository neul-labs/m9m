# Cloud Storage Nodes

Cloud storage nodes interact with object storage services from major cloud providers.

## AWS S3 Node

Perform operations on AWS S3 buckets and objects.

### Type

```
n8n-nodes-base.awsS3
```

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `region` | string | Yes | AWS region |
| `accessKeyId` | string | Yes | AWS access key |
| `secretAccessKey` | string | Yes | AWS secret key |
| `sessionToken` | string | No | Session token (for temp creds) |
| `bucket` | string | Yes | S3 bucket name |
| `operation` | string | Yes | Operation type |
| `key` | string | Depends | Object key/path |

### Operations

| Operation | Description |
|-----------|-------------|
| `upload` | Upload file to S3 |
| `download` | Download file from S3 |
| `delete` | Delete object |
| `list` | List bucket objects |
| `copy` | Copy object |

### Examples

#### Upload File

```json
{
  "type": "n8n-nodes-base.awsS3",
  "parameters": {
    "region": "us-east-1",
    "accessKeyId": "={{ $credentials.aws.accessKeyId }}",
    "secretAccessKey": "={{ $credentials.aws.secretAccessKey }}",
    "bucket": "my-bucket",
    "operation": "upload",
    "key": "uploads/{{ $json.filename }}",
    "body": "={{ $json.content }}",
    "contentType": "application/json"
  }
}
```

#### Download File

```json
{
  "type": "n8n-nodes-base.awsS3",
  "parameters": {
    "region": "us-east-1",
    "accessKeyId": "={{ $credentials.aws.accessKeyId }}",
    "secretAccessKey": "={{ $credentials.aws.secretAccessKey }}",
    "bucket": "my-bucket",
    "operation": "download",
    "key": "data/report.json"
  }
}
```

#### List Objects

```json
{
  "type": "n8n-nodes-base.awsS3",
  "parameters": {
    "region": "us-east-1",
    "accessKeyId": "={{ $credentials.aws.accessKeyId }}",
    "secretAccessKey": "={{ $credentials.aws.secretAccessKey }}",
    "bucket": "my-bucket",
    "operation": "list",
    "prefix": "uploads/"
  }
}
```

#### Delete Object

```json
{
  "type": "n8n-nodes-base.awsS3",
  "parameters": {
    "region": "us-east-1",
    "accessKeyId": "={{ $credentials.aws.accessKeyId }}",
    "secretAccessKey": "={{ $credentials.aws.secretAccessKey }}",
    "bucket": "my-bucket",
    "operation": "delete",
    "key": "old-files/{{ $json.filename }}"
  }
}
```

### Advanced Options

| Option | Description |
|--------|-------------|
| `acl` | Access control (private, public-read) |
| `storageClass` | Storage class (STANDARD, GLACIER) |
| `encryption` | Server-side encryption settings |
| `metadata` | Custom metadata key-value pairs |

---

## Azure Blob Storage Node

Perform operations on Azure Blob Storage containers.

### Type

```
n8n-nodes-base.azureBlobStorage
```

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `accountName` | string | Yes | Storage account name |
| `accountKey` | string | No* | Account key |
| `sasToken` | string | No* | SAS token |
| `connectionString` | string | No* | Connection string |
| `container` | string | Yes | Container name |
| `operation` | string | Yes | Operation type |
| `blob` | string | Depends | Blob name/path |

*One authentication method required.

### Operations

| Operation | Description |
|-----------|-------------|
| `upload` | Upload blob |
| `download` | Download blob |
| `delete` | Delete blob |
| `list` | List blobs |

### Examples

#### Upload Blob

```json
{
  "type": "n8n-nodes-base.azureBlobStorage",
  "parameters": {
    "accountName": "mystorageaccount",
    "accountKey": "={{ $credentials.azure.accountKey }}",
    "container": "data",
    "operation": "upload",
    "blob": "reports/{{ $json.date }}.json",
    "content": "={{ JSON.stringify($json.data) }}",
    "contentType": "application/json"
  }
}
```

#### Download Blob

```json
{
  "type": "n8n-nodes-base.azureBlobStorage",
  "parameters": {
    "connectionString": "={{ $credentials.azure.connectionString }}",
    "container": "documents",
    "operation": "download",
    "blob": "config.json"
  }
}
```

#### List Blobs

```json
{
  "type": "n8n-nodes-base.azureBlobStorage",
  "parameters": {
    "accountName": "mystorageaccount",
    "sasToken": "={{ $credentials.azure.sasToken }}",
    "container": "uploads",
    "operation": "list",
    "prefix": "2024/"
  }
}
```

### Access Tiers

| Tier | Use Case |
|------|----------|
| `Hot` | Frequently accessed data |
| `Cool` | Infrequently accessed |
| `Archive` | Rarely accessed, low cost |

---

## GCP Cloud Storage Node

Perform operations on Google Cloud Storage buckets.

### Type

```
n8n-nodes-base.gcpCloudStorage
```

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `projectId` | string | Yes | GCP project ID |
| `keyFile` | string | No* | Path to service account key |
| `keyFileContent` | string | No* | Key file JSON content |
| `useADC` | boolean | No | Use Application Default Credentials |
| `bucket` | string | Yes | Bucket name |
| `operation` | string | Yes | Operation type |
| `object` | string | Depends | Object name |

### Operations

| Operation | Description |
|-----------|-------------|
| `upload` | Upload object |
| `download` | Download object |
| `delete` | Delete object |
| `list` | List objects |

### Examples

#### Upload Object

```json
{
  "type": "n8n-nodes-base.gcpCloudStorage",
  "parameters": {
    "projectId": "my-project",
    "keyFileContent": "={{ $credentials.gcp.keyFileContent }}",
    "bucket": "my-bucket",
    "operation": "upload",
    "object": "data/{{ $json.filename }}",
    "content": "={{ $json.content }}",
    "contentType": "text/plain"
  }
}
```

#### Download Object

```json
{
  "type": "n8n-nodes-base.gcpCloudStorage",
  "parameters": {
    "projectId": "my-project",
    "useADC": true,
    "bucket": "my-bucket",
    "operation": "download",
    "object": "config/settings.json"
  }
}
```

#### List Objects

```json
{
  "type": "n8n-nodes-base.gcpCloudStorage",
  "parameters": {
    "projectId": "my-project",
    "keyFile": "/path/to/service-account.json",
    "bucket": "my-bucket",
    "operation": "list",
    "prefix": "backups/"
  }
}
```

---

## AWS Lambda Node

Invoke AWS Lambda functions.

### Type

```
n8n-nodes-base.awsLambda
```

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `region` | string | Yes | AWS region |
| `accessKeyId` | string | Yes | AWS access key |
| `secretAccessKey` | string | Yes | AWS secret key |
| `functionName` | string | Yes | Lambda function name/ARN |
| `payload` | object | No | Function input payload |

### Example

```json
{
  "type": "n8n-nodes-base.awsLambda",
  "parameters": {
    "region": "us-east-1",
    "accessKeyId": "={{ $credentials.aws.accessKeyId }}",
    "secretAccessKey": "={{ $credentials.aws.secretAccessKey }}",
    "functionName": "process-data",
    "payload": {
      "action": "transform",
      "data": "={{ $json }}"
    }
  }
}
```

---

## Quick Reference

| Node | Type | Provider |
|------|------|----------|
| AWS S3 | `n8n-nodes-base.awsS3` | Amazon Web Services |
| Azure Blob | `n8n-nodes-base.azureBlobStorage` | Microsoft Azure |
| GCP Storage | `n8n-nodes-base.gcpCloudStorage` | Google Cloud Platform |
| AWS Lambda | `n8n-nodes-base.awsLambda` | Amazon Web Services |

### Authentication Comparison

| Provider | Methods |
|----------|---------|
| AWS | Access Key + Secret, Session Token |
| Azure | Account Key, SAS Token, Connection String, Managed Identity |
| GCP | Service Account Key, ADC |
