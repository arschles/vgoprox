package azurecdn

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"

	"github.com/Azure/azure-storage-blob-go/2017-07-29/azblob"
	multierror "github.com/hashicorp/go-multierror"
)

// Storage implements (github.com/gomods/athens/pkg/storage).Saver and
// also provides a function to fetch the location of a module
type Storage struct {
	accountURL *url.URL
	cred       azblob.Credential
}

// New creates a new azure CDN saver
func New(accountName, accountKey string) (*Storage, error) {
	u, err := url.Parse(fmt.Sprintf("https://%s.blob.core.windows.net", accountName))
	if err != nil {
		return nil, err
	}
	cred := azblob.NewSharedKeyCredential(accountName, accountKey)
	return &Storage{accountURL: u, cred: cred}, nil
}

// BaseURL returns the base URL that stores all modules. It can be used
// in the "meta" tag redirect response to vgo.
//
// For example:
//
//	<meta name="go-import" content="gomods.com/athens mod BaseURL()">
func (s Storage) BaseURL() *url.URL {
	return s.accountURL
}

// Save implements the (github.com/gomods/athens/pkg/storage).Saver interface.
func (s *Storage) Save(ctx context.Context, module, version string, mod []byte, zip io.Reader, info []byte) error {
	pipe := azblob.NewPipeline(s.cred, azblob.PipelineOptions{})
	serviceURL := azblob.NewServiceURL(*s.accountURL, pipe)
	// rules on container names:
	// https://docs.microsoft.com/en-us/rest/api/storageservices/naming-and-referencing-containers--blobs--and-metadata#container-names
	//
	// This container must exist
	containerURL := serviceURL.NewContainerURL("gomodules")

	infoBlobURL := containerURL.NewBlockBlobURL(fmt.Sprintf("%s/@v/%s.info", module, version))
	modBlobURL := containerURL.NewBlockBlobURL(fmt.Sprintf("%s/@v/%s.mod", module, version))
	zipBlobURL := containerURL.NewBlockBlobURL(fmt.Sprintf("%s/@v/%s.zip", module, version))

	httpHeaders := func(contentType string) azblob.BlobHTTPHeaders {
		return azblob.BlobHTTPHeaders{
			ContentType: contentType,
		}
	}
	emptyMeta := map[string]string{}
	emptyBlobAccessCond := azblob.BlobAccessConditions{}

	const numUpload = 3
	uploadErrs := make(chan error, numUpload)

	upload := func(url azblob.BlockBlobURL, content io.ReadSeeker, contentType string) {
		_, err := url.Upload(ctx, content, httpHeaders(contentType), emptyMeta, emptyBlobAccessCond)
		uploadErrs <- err
	}
	zipBytes, err := ioutil.ReadAll(zip)
	if err != nil {
		return err
	}

	go upload(infoBlobURL, bytes.NewReader(info), "application/json")
	go upload(modBlobURL, bytes.NewReader(mod), "text/plain")
	go upload(zipBlobURL, bytes.NewReader(zipBytes), "application/octet-stream")

	var errors error
	for i := 0; i < numUpload; i++ {
		select {
		case err := <-uploadErrs:
			if err != nil {
				errors = multierror.Append(errors, err)
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	// TODO: take out lease on the /list file and add the version to it
	//
	// Do that only after module source+metadata is uploaded

	return errors
}
