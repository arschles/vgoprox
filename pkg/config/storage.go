package config

// StorageConfig provides configs for various storage backends
type StorageConfig struct {
	Disk      *DiskConfig
	GCP       *GCPConfig
	Minio     *MinioConfig
	Mongo     *MongoConfig
	S3        *S3Config
	AzureBlob *AzureBlobConfig
	External  *External
}
