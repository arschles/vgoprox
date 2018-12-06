package gcp

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"cloud.google.com/go/storage"
	"github.com/gomods/athens/pkg/config"
	"github.com/gomods/athens/pkg/errors"
)

// Storage implements the (./pkg/storage).Backend interface
type Storage struct {
	bucket       Bucket
	baseURI      *url.URL
	closeStorage func() error
	projectID    string
	timeout      time.Duration
}

// New returns a new Storage instance backed by a Google Cloud Storage bucket.
// The bucket name to be used will be loaded from the
// environment variable ATHENS_STORAGE_GCP_BUCKET.
//
// If you're not running on GCP, set the GOOGLE_APPLICATION_CREDENTIALS environment variable
// to the path of your service account file. If you're running on GCP (e.g. AppEngine),
// credentials will be automatically provided.
// See https://cloud.google.com/docs/authentication/getting-started.
func New(ctx context.Context, gcpConf *config.GCPConfig, timeout time.Duration) (*Storage, error) {
	const op errors.Op = "gcp.New"
	s, err := storage.NewClient(ctx)
	if err != nil {
		return nil, errors.E(op, fmt.Errorf("could not create new storage client: %s", err))
	}

	u, err := url.Parse(fmt.Sprintf("https://storage.googleapis.com/%s", gcpConf.Bucket))
	if err != nil {
		return nil, errors.E(op, err)
	}

	bkt := s.Bucket(gcpConf.Bucket)
	if _, err := bkt.Attrs(ctx); err != nil {
		if err == storage.ErrBucketNotExist {
			return nil, errors.E(op, "You must manually create a storage bucket for Athens, see https://cloud.google.com/storage/docs/creating-buckets#storage-create-bucket-console")
		}

		return nil, errors.E(op, err)
	}

	return &Storage{
		bucket:       &gcpBucket{bkt},
		baseURI:      u,
		closeStorage: s.Close,
		timeout:      timeout,
	}, nil
}

func newWithBucket(bkt Bucket, uri *url.URL, timeout time.Duration) *Storage {
	return &Storage{
		bucket:       bkt,
		baseURI:      uri,
		closeStorage: func() error { return nil },
		timeout:      timeout,
	}
}

// Close calls the underlying storage client's close method
// It is not required to be called on program exit but provided here
// for completness.
func (s *Storage) Close() error {
	return s.closeStorage()
}
