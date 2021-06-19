package config

// Storage provides configs for various storage backends
type Storage struct {
	Disk        *DiskConfig
	GCP         *GCPConfig
	Minio       *MinioConfig
	Mongo       *MongoConfig
	S3          *S3Config
	AzureBlob   *AzureBlobConfig
	External    *External
	Artifactory *ArtifactoryConfig
}
