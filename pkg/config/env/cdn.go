package env

import (
	"net/url"

	"github.com/gobuffalo/envy"
)

// CdnEndpointWithDefault returns CDN endpoint if set
// if not it should default to clouds default blob storage endpoint e.g
func CdnEndpointWithDefault(value *url.URL) *url.URL {
	rawURI, err := envy.MustGet("CDN_ENDPOINT")
	if err != nil {
		return value
	}

	uri, err := url.Parse(rawURI)
	if err != nil {
		return value
	}

	return uri
}
