package config

// External specifies configuration for an external http storage
type External struct {
	URL string `validate:"required" envconfig:"ATHENS_EXTERNAL_STORAGE_URL"`
}
