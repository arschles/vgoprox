package config

// ArtifactoryConfig specifies configuration for an artifactory storage
type ArtifactoryConfig struct {
	URL         string `validate:"required" envconfig:"ATHENS_ARTIFACTORY_URL"`
	Repository  string `validate:"required" envconfig:"ATHENS_ARTIFACTORY_REPOSITORY"`
	Username    string `envconfig:"ATHENS_ARTIFACTORY_USERNAME"`
	Password    string `envconfig:"ATHENS_ARTIFACTORY_PASSWORD"`
	APIKey      string `envconfig:"ATHENS_ARTIFACTORY_API_KEY"`
	AccessToken string `envconfig:"ATHENS_ARTIFACTORY_ACCESS_TOKEN"`
}
