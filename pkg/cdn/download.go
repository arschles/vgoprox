package cdn

import (
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"sync"

	"github.com/gomods/athens/pkg/config"
	"github.com/gomods/athens/pkg/config/env"
	"github.com/gomods/athens/pkg/storage"
)

// ModVerDownloader downloads a module version from a URL
type ModVerDownloader func(ctx context.Context, baseURL, module, version string) (*storage.Version, error)

// Download downloads the module/version from url. Returns a storage.Version
// representing the downloaded module/version or a non-nil error if something went wrong
func Download(ctx context.Context, baseURL, module, version string) (*storage.Version, error) {
	tctx, cancel := context.WithTimeout(ctx, env.Timeout())
	defer cancel()

	var info []byte
	var infoErr error

	var mod []byte
	var modErr error

	var zip io.ReadCloser
	var zipErr error

	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		infoReq, err := getRequest(tctx, baseURL, module, version, ".info")
		if err != nil {
			info, infoErr = nil, err
			return
		}
		infoStream, err := getResBody(infoReq)
		if err != nil {
			info, infoErr = nil, err
			return
		}
		info, infoErr = getBytes(infoStream)
	}()

	go func() {
		defer wg.Done()
		modReq, err := getRequest(tctx, baseURL, module, version, ".mod")
		if err != nil {
			mod, modErr = nil, err
			return
		}
		modStream, err := getResBody(modReq)
		if err != nil {
			mod, modErr = nil, err
			return
		}
		mod, modErr = getBytes(modStream)
	}()

	go func() {
		defer wg.Done()
		zipReq, err := getRequest(tctx, baseURL, module, version, ".zip")
		if err != nil {
			zip, zipErr = nil, err
			return
		}
		zip, zipErr = getResBody(zipReq)
	}()
	wg.Wait()

	if infoErr != nil {
		return nil, infoErr
	}
	if modErr != nil {
		return nil, modErr
	}
	if zipErr != nil {
		return nil, zipErr
	}
	ver := storage.Version{
		Info: info,
		Mod:  mod,
		Zip:  zip,
	}
	return &ver, nil
}

func getBytes(rb io.ReadCloser) ([]byte, error) {
	defer rb.Close()
	return ioutil.ReadAll(rb)
}

func getResBody(req *http.Request) (io.ReadCloser, error) {
	client := http.Client{Timeout: env.Timeout()}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	return res.Body, nil
}

func getRequest(ctx context.Context, baseURL, module, version, ext string) (*http.Request, error) {
	u, err := join(baseURL, module, version, ext)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	return req, nil
}

func join(baseURL string, module, version, ext string) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	packageVersionedName := config.PackageVersionedName(module, version, ext)
	u.Path = path.Join(u.Path, packageVersionedName)
	return u.String(), nil
}
