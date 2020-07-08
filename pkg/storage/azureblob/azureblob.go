package azureblob

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/gomods/athens/pkg/config"
	"github.com/gomods/athens/pkg/errors"
	"github.com/gomods/athens/pkg/observ"
)

type client interface {
	UploadWithContext(ctx context.Context, path, contentType string, content io.Reader) error
	BlobExists(ctx context.Context, path string) (bool, error)
	ReadBlob(ctx context.Context, path string) (io.ReadCloser, error)
	ListBlobs(ctx context.Context, prefix string) ([]string, error)
	DeleteBlob(ctx context.Context, path string) error
}

type azureBlobStoreClient struct {
	containerURL *azblob.ContainerURL
}

func newBlobStoreClient(accountURL *url.URL, accountName, accountKey, containerName string) (*azureBlobStoreClient, error) {
	const op errors.Op = "azureblob.newBlobStoreClient"
	cred, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		return nil, errors.E(op, err)
	}
	pipe := azblob.NewPipeline(cred, azblob.PipelineOptions{})
	serviceURL := azblob.NewServiceURL(*accountURL, pipe)
	// rules on container names:
	// https://docs.microsoft.com/en-us/rest/api/storageservices/naming-and-referencing-containers--blobs--and-metadata#container-names
	//
	// This container must exist
	containerURL := serviceURL.NewContainerURL(containerName)
	cl := &azureBlobStoreClient{containerURL: &containerURL}
	return cl, nil
}

// Storage implements (github.com/gomods/athens/pkg/storage).Saver and
// also provides a function to fetch the location of a module
type Storage struct {
	client  *azureBlobStoreClient
	timeout time.Duration
}

// New creates a new azure blobs storage
func New(conf *config.AzureBlobConfig, timeout time.Duration) (*Storage, error) {
	const op errors.Op = "azureblob.New"
	portalURL := conf.PortalURL
	if portalURL == "" {
		portalURL = fmt.Sprintf("https://%s.blob.core.windows.net", conf.AccountName)
	}
	u, err := url.Parse(portalURL)
	if err != nil {
		return nil, errors.E(op, err)
	}
	cl, err := newBlobStoreClient(u, conf.AccountName, conf.AccountKey, conf.ContainerName)
	if err != nil {
		return nil, errors.E(op, err)
	}
	return &Storage{client: cl, timeout: timeout}, nil
}

// BlobExists checks if a particular blob exists in the container
func (c *azureBlobStoreClient) BlobExists(ctx context.Context, path string) (bool, error) {
	const op errors.Op = "azureblob.BlobExists"
	// TODO: Any better way of doing this ?
	blobURL := c.containerURL.NewBlockBlobURL(path)
	_, err := blobURL.GetProperties(ctx, azblob.BlobAccessConditions{})
	if err != nil {
		var serr azblob.StorageError
		var ok bool

		if serr, ok = err.(azblob.StorageError); !ok {
			return false, errors.E(op, fmt.Errorf("Error in casting to azure error type %v", err))
		}
		if serr.Response().StatusCode == http.StatusNotFound {
			return false, nil
		}

		return false, errors.E(op, err)
	}
	return true, nil

}

// ReadBlob returns an io.ReadCloser for the contents of a blob
func (c *azureBlobStoreClient) ReadBlob(ctx context.Context, path string) (io.ReadCloser, error) {
	const op errors.Op = "azureblob.ReadBlob"
	blobURL := c.containerURL.NewBlockBlobURL(path)
	downloadResponse, err := blobURL.Download(ctx, 0, 0, azblob.BlobAccessConditions{}, false)
	if err != nil {
		return nil, errors.E(op, err)
	}
	return downloadResponse.Body(azblob.RetryReaderOptions{}), nil
}

// ListBlobs will list all blobs which has the given prefix
func (c *azureBlobStoreClient) ListBlobs(ctx context.Context, prefix string) ([]string, error) {
	const op errors.Op = "azureblob.ListBlobs"
	var blobs []string
	for marker := (azblob.Marker{}); marker.NotDone(); {
		listBlob, err := c.containerURL.ListBlobsFlatSegment(ctx, marker, azblob.ListBlobsSegmentOptions{
			Prefix: prefix,
		})
		if err != nil {
			return nil, errors.E(op, err)
		}
		marker = listBlob.NextMarker

		for _, blob := range listBlob.Segment.BlobItems {
			blobs = append(blobs, blob.Name)
		}
	}

	return blobs, nil
}

// DeleteBlob deletes the blob with the given path
func (c *azureBlobStoreClient) DeleteBlob(ctx context.Context, path string) error {
	const op errors.Op = "azureblob.DeleteBlob"
	blobURL := c.containerURL.NewBlockBlobURL(path)
	_, err := blobURL.Delete(ctx, azblob.DeleteSnapshotsOptionNone, azblob.BlobAccessConditions{})
	if err != nil {
		return errors.E(op, err)
	}
	return nil
}

// UploadWithContext uploads a blob to the container
func (c *azureBlobStoreClient) UploadWithContext(ctx context.Context, path, contentType string, content io.Reader) error {
	const op errors.Op = "azureblob.UploadWithContext"
	ctx, span := observ.StartSpan(ctx, op.String())
	defer span.End()
	blobURL := c.containerURL.NewBlockBlobURL(path)
	bufferSize := 1 * 1024 * 1024 // Size of the rotating buffers that are used when uploading
	maxBuffers := 3               // Number of rotating buffers that are used when uploading

	uploadStreamOpts := azblob.UploadStreamToBlockBlobOptions{
		BufferSize: bufferSize,
		MaxBuffers: maxBuffers,
		BlobHTTPHeaders: azblob.BlobHTTPHeaders{
			ContentType: contentType,
		},
	}
	_, err := azblob.UploadStreamToBlockBlob(ctx, content, blobURL, uploadStreamOpts)
	if err != nil {
		return errors.E(op, err)
	}
	return nil
}
