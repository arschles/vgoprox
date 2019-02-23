package azureblob

import (
	"context"

	"github.com/gomods/athens/pkg/errors"
	"github.com/gomods/athens/pkg/observ"
	"github.com/gomods/athens/pkg/storage"
	moduploader "github.com/gomods/athens/pkg/storage/module"
)

// Save implements the (./pkg/storage).Saver interface.
func (s *Storage) Save(ctx context.Context, module, version string, mod []byte, zip storage.Zip, info []byte) error {
	const op errors.Op = "azureblob.Save"
	ctx, span := observ.StartSpan(ctx, op.String())
	defer span.End()

	zipStream := moduploader.Stream{zip.Reader, zip.Size}
	err := moduploader.Upload(ctx, module, version, moduploader.NewStreamFromBytes(info), moduploader.NewStreamFromBytes(mod), zipStream, s.client.UploadWithContext, s.timeout)
	if err != nil {
		return errors.E(op, err, errors.M(module), errors.V(version))
	}

	return nil
}
