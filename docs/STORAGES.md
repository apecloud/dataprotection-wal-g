# WAL-G storage configuration

WAL-G can store backups in S3, Google Cloud Storage, Azure, Swift, remote host (via SSH) or local file system. 

S3
-----------

To connect to Amazon S3, WAL-G requires that this variable be set:

* `WALG_S3_PREFIX`
(e.g. `s3://bucket/path/to/folder`) (alternative form `WALE_S3_PREFIX`)

WAL-G determines AWS credentials [like other AWS tools](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-files.html). You can set `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` (optionally with `AWS_SESSION_TOKEN`), or `~/.aws/credentials` (optionally with `AWS_PROFILE`), or you can set nothing to fetch credentials from the EC2 metadata service automatically.

**Optional variables**

* `AWS_ROLE_ARN`
(e.g. `arn:aws:iam::123123123123:role/s3_access`)

Set this variable along with `AWS_ROLE_SESSION_NAME` if you are using [AWS IAM](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-role.html#cli-configure-role-oidc) roles to access S3 buckets.

* `AWS_ROLE_SESSION_NAME`

Specifies the name to attach to the role session. Useful only when `AWS_ROLE_ARN` is set.

* `AWS_REGION`
(e.g. `us-west-2`)

WAL-G can automatically determine the S3 bucket's region using `s3:GetBucketLocation`, but if you wish to avoid this API call or forbid it from the applicable IAM policy, specify this variable.

* `AWS_ENDPOINT`

Overrides the default hostname to connect to an S3-compatible service. i.e, `http://s3-like-service:9000`

* `AWS_S3_FORCE_PATH_STYLE`

To enable path-style addressing (i.e., `http://s3.amazonaws.com/BUCKET/KEY`) when connecting to an S3-compatible service that lack of support for sub-domain style bucket URLs (i.e., `http://BUCKET.s3.amazonaws.com/KEY`). Defaults to `false`.

* `WALG_S3_STORAGE_CLASS`

To configure the S3 storage class used for backup files, use `WALG_S3_STORAGE_CLASS`. By default, WAL-G uses the "STANDARD" storage class. Other supported values include "STANDARD_IA" for Infrequent Access and "REDUCED_REDUNDANCY" for Reduced Redundancy.

* `WALG_S3_SSE`

To enable S3 server-side encryption, set to the algorithm to use when storing the objects in S3 (i.e., `AES256`, `aws:kms`).

* `WALG_S3_SSE_KMS_ID`

If using S3 server-side encryption with `aws:kms`, the KMS Key ID to use for object encryption.

* `WALG_CSE_KMS_ID`

To configure AWS KMS key for client-side encryption and decryption. By default, no encryption is used. (AWS_REGION or WALG_CSE_KMS_REGION required to be set when using AWS KMS key client-side encryption)

* `WALG_CSE_KMS_REGION`

To configure AWS KMS key for client-side encryption and decryption. By default, no encryption is used. (AWS_REGION or WALG_CSE_KMS_REGION required to be set when using AWS KMS key client-side encryption)

* `WALG_S3_RANGE_BATCH_ENABLED`

Set to TRUE to allow wal-g in case of network problems to continue downloading from the point that was already downloaded using HTTP Range query. This option is useful when download big files more than few hours.

* `WALG_S3_RANGE_MAX_RETRIES`

If `WALG_S3_RANGE_BATCH_ENABLED` enabled, wal-g will try to reconnect N times, by default 10 times

* `WALG_S3_USE_LIST_OBJECTS_V1`

By default, WAL-G uses [ListObjectsV2](https://docs.aws.amazon.com/AmazonS3/latest/API/API_ListObjectsV2.html) to fetch S3 storage folder listings.
However, some S3-compatible storages may not support it.
Set this setting to `true` to use [ListObjects](https://docs.aws.amazon.com/AmazonS3/latest/API/API_ListObjects.html) instead.

* `WALG_S3_MAX_RETRIES`

Overrides the default request retry limit while interacting with S3. Default is 15.

GCS
-----------
To store backups in Google Cloud Storage, WAL-G requires that this variable be set:

* `WALG_GS_PREFIX`
to specify where to store backups (e.g. `gs://x4m-test-bucket/walg-folder`)

WAL-G determines Google Cloud credentials using [application-default credentials](https://cloud.google.com/docs/authentication/production) like other GCP tools. You can set `GOOGLE_APPLICATION_CREDENTIALS` to point to a service account json key from GCP. If you set nothing, WAL-G will attempt to fetch credentials from the GCE/GKE metadata service.

**Optional variables**

* `GCS_CONTEXT_TIMEOUT`

Default: 1 hour.

* `GCS_NORMALIZE_PREFIX`

Controls the trimming of extra slashes in paths. The default is `true`. To allow restoring from WAL-E archives on GCS, set it to `false` and keep double slashes in `WALG_GS_PREFIX` values.

* `GCS_ENCRYPTION_KEY`

To configure GCS Customer Supplied Encryption Key (CSEK) for client-side encryption and decryption. By default, Google-managed keys are used. CSEK must be a 32-byte AES-256 key, encoded in standard Base64.

* `GCS_MAX_CHUNK_SIZE`
(e.g. `16777216`)

Overrides the default `maximum chunk size` of 52428800 bytes (50 MiB). The size of the chunk must be specified in bytes. This parameter could be useful for different types of uploading (e.g. `16777216` (16MiB) would be perfect for `wal-push`, `52428800` (50MiB) is suitable for `backup-push`).

* `GCS_MAX_RETRIES`
(e.g. `1`)

Overrides the default upload and download retry limit while interacting with GCS.  Default: 16.

Azure
-----------
To store backups in Azure Storage, WAL-G requires that these variables be set:

* `WALG_AZ_PREFIX`
to specify where to store backups in Azure storage (e.g. `azure://test-container/walg-folder`)

* `AZURE_STORAGE_ACCOUNT`
to specify the storage account of your backups

WAL-G determines Azure Storage credentials using the [azure default credential chain](https://docs.microsoft.com/en-us/azure/developer/go/azure-sdk-authentication). You can set the standard `AZURE_CLIENT_ID`/`AZURE_TENANT_ID`/`AZURE_CLIENT_SECRET` environment variables to authenticate, or exclude them all if you are using a Managed Identity.

Alternatively, you can set `AZURE_STORAGE_ACCESS_KEY` to authenticate using the storage account's [access keys](https://docs.microsoft.com/en-us/azure/storage/common/storage-account-keys-manage), or set `AZURE_STORAGE_SAS_TOKEN` to make use of [SAS tokens](https://docs.microsoft.com/en-us/azure/storage/common/storage-sas-overview).

For deployments where Azure Storage is not under AzurePuplicCloud environment, WAL-G need to use different Azure Storage endpoint. You can use optional setting `AZURE_STORAGE_SAS_TOKEN` to select the correct Azure Storage endpoint. Available setting values:  `"AzurePublicCloud"`, `"AzureUSGovernmentCloud"`, `"AzureChinaCloud"`, `"AzureGermanCloud"`. If setting is omitted or has a value different to the ones defined here, WAL-G will default to the Azure Storage endpoint for AzurePublicCloud.

WAL-G sets default upload buffer size to 64 Megabytes and uses 3 buffers by default. However, users can choose to override these values by setting optional environment variables.

* `WALG_AZURE_BUFFER_SIZE`
  (e.g. `33554432`)

Overrides the default `upload buffer size` of 8388608 bytes (8 MB). Note that the size of the buffer must be specified in bytes. Therefore, to use 32 MB sized buffers, this variable should be set to 33554432 bytes.

* `WALG_AZURE_MAX_BUFFERS`
  (e.g. `5`)

Overrides the default `maximum number of upload buffers`. By default, at most 4 buffers are used concurrently.

Swift
-----------
To store backups in Swift object storage, WAL-G requires that this variable be set:

* `WALG_SWIFT_PREFIX`
to specify where to store backups in Swift object storage (e.g. `swift://test-container/walg-folder`)

WAL-G determines Swift object storage credentials using [openStack default credentials](https://www.swiftstack.com/docs/cookbooks/swift_usage/auth.html). You can use any of V1, V2, V3 of the SwiftStack Auth middleware to provide Swift object storage credentials.

File system
-----------
To store backups on files system, WAL-G requires that these variables be set:

* `WALG_FILE_PREFIX`
(e.g. `/tmp/wal-g-test-data`)

Please, keep in mind that by default storing backups on disk along with database is not safe. Do not use it as a disaster recovery plan.

SSH
-----------
To store backups via ssh, WAL-G requires that these variables be set:
* `WALG_SSH_PREFIX` (e.g. `ssh://localhost/walg-folder`)
* `SSH_PORT` ssh connection port
* `SSH_USERNAME` connect with username
* `SSH_PASSWORD` connect with password
* `SSH_PRIVATE_KEY_PATH` or connect with a SSH KEY by specifying its full path

Examples
-----------
***Example: Using Minio.io S3-compatible storage***

```bash
AWS_ACCESS_KEY_ID: "<minio-key>"
AWS_SECRET_ACCESS_KEY: "<minio-secret>"
WALG_S3_PREFIX: "s3://my-minio-bucket/sub-dir"
AWS_ENDPOINT: "http://minio:9000"
AWS_S3_FORCE_PATH_STYLE: "true"
AWS_REGION: us-east-1
WALG_S3_CA_CERT_FILE: "/path/to/custom/ca/file"
```
