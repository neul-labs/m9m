# Cloud Nodes

Cloud nodes enable integration with major cloud platforms.

## AWS

### S3

#### List Objects

```json
{
  "type": "n8n-nodes-base.awsS3",
  "parameters": {
    "operation": "list",
    "bucket": "my-bucket",
    "prefix": "data/"
  },
  "credentials": {
    "aws": {"id": "1", "name": "AWS"}
  }
}
```

#### Download Object

```json
{
  "parameters": {
    "operation": "download",
    "bucket": "my-bucket",
    "key": "={{ $json.objectKey }}",
    "binaryPropertyName": "data"
  }
}
```

#### Upload Object

```json
{
  "parameters": {
    "operation": "upload",
    "bucket": "my-bucket",
    "key": "uploads/{{ $json.filename }}",
    "binaryPropertyName": "data",
    "contentType": "application/json"
  }
}
```

#### Delete Object

```json
{
  "parameters": {
    "operation": "delete",
    "bucket": "my-bucket",
    "key": "={{ $json.objectKey }}"
  }
}
```

### Lambda

#### Invoke Function

```json
{
  "type": "n8n-nodes-base.awsLambda",
  "parameters": {
    "operation": "invoke",
    "functionName": "my-function",
    "payload": "={{ JSON.stringify($json) }}",
    "invocationType": "RequestResponse"
  }
}
```

#### Async Invocation

```json
{
  "parameters": {
    "functionName": "background-processor",
    "invocationType": "Event",
    "payload": "={{ JSON.stringify($json) }}"
  }
}
```

### SQS

#### Send Message

```json
{
  "type": "n8n-nodes-base.awsSqs",
  "parameters": {
    "operation": "sendMessage",
    "queueUrl": "https://sqs.us-east-1.amazonaws.com/123456789/my-queue",
    "messageBody": "={{ JSON.stringify($json) }}"
  }
}
```

#### Receive Messages

```json
{
  "parameters": {
    "operation": "receiveMessage",
    "queueUrl": "https://sqs.us-east-1.amazonaws.com/123456789/my-queue",
    "maxMessages": 10
  }
}
```

### AWS Credential

```json
{
  "type": "aws",
  "data": {
    "accessKeyId": "AKIAIOSFODNN7EXAMPLE",
    "secretAccessKey": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
    "region": "us-east-1"
  }
}
```

## Google Cloud Platform

### Cloud Storage

#### List Objects

```json
{
  "type": "n8n-nodes-base.googleCloudStorage",
  "parameters": {
    "operation": "list",
    "bucket": "my-bucket",
    "prefix": "data/"
  },
  "credentials": {
    "googleCloudStorage": {"id": "1", "name": "GCP"}
  }
}
```

#### Download Object

```json
{
  "parameters": {
    "operation": "download",
    "bucket": "my-bucket",
    "objectName": "={{ $json.filename }}",
    "binaryPropertyName": "data"
  }
}
```

#### Upload Object

```json
{
  "parameters": {
    "operation": "upload",
    "bucket": "my-bucket",
    "objectName": "uploads/{{ $json.filename }}",
    "binaryPropertyName": "data"
  }
}
```

### BigQuery

#### Query

```json
{
  "type": "n8n-nodes-base.googleBigQuery",
  "parameters": {
    "operation": "executeQuery",
    "projectId": "my-project",
    "query": "SELECT * FROM `dataset.table` WHERE date > @date",
    "queryParameters": [
      {"name": "date", "type": "DATE", "value": "={{ $json.startDate }}"}
    ]
  }
}
```

#### Insert Rows

```json
{
  "parameters": {
    "operation": "insert",
    "projectId": "my-project",
    "datasetId": "my_dataset",
    "tableId": "my_table"
  }
}
```

### Cloud Functions

```json
{
  "type": "n8n-nodes-base.googleCloudFunctions",
  "parameters": {
    "operation": "invoke",
    "projectId": "my-project",
    "region": "us-central1",
    "functionName": "my-function",
    "payload": "={{ JSON.stringify($json) }}"
  }
}
```

### GCP Credential

```json
{
  "type": "googleCloudStorage",
  "data": {
    "serviceAccountEmail": "service@project.iam.gserviceaccount.com",
    "privateKey": "-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----\n",
    "projectId": "my-project"
  }
}
```

## Microsoft Azure

### Blob Storage

#### List Blobs

```json
{
  "type": "n8n-nodes-base.azureBlobStorage",
  "parameters": {
    "operation": "list",
    "container": "my-container",
    "prefix": "data/"
  },
  "credentials": {
    "azureBlobStorage": {"id": "1", "name": "Azure"}
  }
}
```

#### Download Blob

```json
{
  "parameters": {
    "operation": "download",
    "container": "my-container",
    "blobName": "={{ $json.filename }}",
    "binaryPropertyName": "data"
  }
}
```

#### Upload Blob

```json
{
  "parameters": {
    "operation": "upload",
    "container": "my-container",
    "blobName": "uploads/{{ $json.filename }}",
    "binaryPropertyName": "data"
  }
}
```

### Azure Functions

```json
{
  "type": "n8n-nodes-base.azureFunctions",
  "parameters": {
    "operation": "invoke",
    "functionUrl": "https://my-func.azurewebsites.net/api/process",
    "payload": "={{ JSON.stringify($json) }}"
  }
}
```

### Azure Credential

```json
{
  "type": "azureBlobStorage",
  "data": {
    "accountName": "mystorageaccount",
    "accountKey": "your-storage-account-key"
  }
}
```

## Common Patterns

### Cross-Cloud Sync

Sync files between clouds:

```json
{
  "nodes": [
    {
      "id": "list-s3",
      "type": "n8n-nodes-base.awsS3",
      "parameters": {"operation": "list", "bucket": "source-bucket"}
    },
    {
      "id": "download-s3",
      "type": "n8n-nodes-base.awsS3",
      "parameters": {"operation": "download"}
    },
    {
      "id": "upload-gcs",
      "type": "n8n-nodes-base.googleCloudStorage",
      "parameters": {"operation": "upload", "bucket": "target-bucket"}
    }
  ]
}
```

### Event-Driven Processing

Process uploaded files:

```json
{
  "nodes": [
    {
      "id": "webhook",
      "type": "n8n-nodes-base.webhook",
      "parameters": {"path": "s3-event"}
    },
    {
      "id": "download",
      "type": "n8n-nodes-base.awsS3",
      "parameters": {
        "operation": "download",
        "bucket": "={{ $json.bucket }}",
        "key": "={{ $json.key }}"
      }
    },
    {
      "id": "process",
      "type": "n8n-nodes-base.code"
    },
    {
      "id": "upload-result",
      "type": "n8n-nodes-base.awsS3",
      "parameters": {
        "operation": "upload",
        "bucket": "processed-bucket"
      }
    }
  ]
}
```

### Backup Workflow

Regular backups to cloud storage:

```json
{
  "nodes": [
    {
      "id": "cron",
      "type": "n8n-nodes-base.cron",
      "parameters": {"triggerTimes": {"item": [{"mode": "everyDay", "hour": 2}]}}
    },
    {
      "id": "export-data",
      "type": "n8n-nodes-base.postgres",
      "parameters": {"operation": "executeQuery", "query": "SELECT * FROM important_table"}
    },
    {
      "id": "convert-json",
      "type": "n8n-nodes-base.code",
      "parameters": {"code": "return [{json: {data: items}, binary: {file: Buffer.from(JSON.stringify(items))}}]"}
    },
    {
      "id": "upload-backup",
      "type": "n8n-nodes-base.awsS3",
      "parameters": {
        "operation": "upload",
        "bucket": "backups",
        "key": "db-backup/{{ $now.format('YYYY-MM-DD') }}.json"
      }
    }
  ]
}
```

## Security Best Practices

1. **Use IAM roles** instead of access keys when possible
2. **Apply least privilege** - Only grant necessary permissions
3. **Enable encryption** - Use server-side encryption for storage
4. **Rotate credentials** regularly
5. **Audit access** - Enable cloud audit logging
6. **Use VPC endpoints** for private connectivity

## Cost Optimization

1. **Use appropriate storage classes** (S3 Glacier for archives)
2. **Set lifecycle policies** for automatic cleanup
3. **Monitor usage** with cloud cost tools
4. **Batch operations** to reduce API calls
5. **Use reserved capacity** for predictable workloads

## Next Steps

- [Custom Nodes](custom-nodes.md) - Build custom cloud integrations
- [Deployment](../deployment/kubernetes.md) - Deploy on cloud platforms
- [Production](../deployment/production.md) - Production best practices
