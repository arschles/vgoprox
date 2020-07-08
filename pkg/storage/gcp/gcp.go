package gcp

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"cloud.google.com/go/storage"
	"github.com/gomods/athens/pkg/config"
	"github.com/gomods/athens/pkg/errors"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
)

// Storage implements the (./pkg/storage).Backend interface
type Storage struct {
	bucket  *storage.BucketHandle
	timeout time.Duration
}

// New returns a new Storage instance backed by a Google Cloud Storage bucket.
// The bucket name to be used will be loaded from the
// environment variable ATHENS_STORAGE_GCP_BUCKET.
//
// If you're not running on GCP, set the GOOGLE_APPLICATION_CREDENTIALS environment variable
// to the path of your service account file. If you're running on GCP (e.g. AppEngine),
// credentials will be automatically provided.
// See https://cloud.google.com/docs/authentication/getting-started.
func New(ctx context.Context, gcpConf *config.GCPConfig, timeout time.Duration, httpClient *http.Client) (*Storage, error) {
	const op errors.Op = "gcp.New"
	s, err := newClient(ctx, gcpConf, timeout, httpClient)
	if err != nil {
		return nil, errors.E(op, err)
	}

	if _, err := s.bucket.Attrs(ctx); err != nil {
		if err == storage.ErrBucketNotExist {
			return nil, errors.E(op, "You must manually create a storage bucket for Athens, see https://cloud.google.com/storage/docs/creating-buckets#storage-create-bucket-console")
		}
		return nil, errors.E(op, err)
	}

	return s, nil
}

// newClient handles the GCS client creation but does not check whether the bucket exists or not
// this is so that the unit tests can use this to create their own short-lived buckets.
func newClient(ctx context.Context, gcpConf *config.GCPConfig, timeout time.Duration, httpClient *http.Client) (*Storage, error) {
	const op errors.Op = "gcp.newClient"
	opts := []option.ClientOption{}
	if gcpConf.JSONKey != "" {
		key, err := base64.StdEncoding.DecodeString(gcpConf.JSONKey)
		if err != nil {
			return nil, errors.E(op, fmt.Errorf("could not decode base64 json key: %v", err))
		}
		creds, err := google.CredentialsFromJSON(ctx, key, storage.ScopeReadWrite)
		if err != nil {
			return nil, errors.E(op, fmt.Errorf("could not get GCS credentials: %v", err))
		}
		opts = append(opts, option.WithCredentials(creds))
	}
	if httpClient != nil {
		opts = append(opts, option.WithHTTPClient(httpClient))
	}
	s, err := storage.NewClient(ctx, opts...)
	if err != nil {
		return nil, errors.E(op, fmt.Errorf("could not create new storage client: %s", err))
	}

	return &Storage{
		bucket:  s.Bucket(gcpConf.Bucket),
		timeout: timeout,
	}, nil
}
