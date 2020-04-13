package external

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/gomods/athens/pkg/errors"
	"github.com/gomods/athens/pkg/storage"
	"golang.org/x/mod/module"
)

type service struct {
	url string
	c   *http.Client
}

// NewClient returns an external storage client
func NewClient(url string, c *http.Client) storage.Backend {
	if c == nil {
		c = &http.Client{}
	}
	url = strings.TrimSuffix(url, "/")
	return &service{url, c}
}

func (s *service) List(ctx context.Context, mod string) ([]string, error) {
	const op errors.Op = "external.List"
	body, err := s.getRequest(ctx, mod, "list", "")
	if err != nil {
		return nil, errors.E(op, err)
	}
	list := []string{}
	scnr := bufio.NewScanner(body)
	for scnr.Scan() {
		list = append(list, scnr.Text())
	}
	if scnr.Err() != nil {
		return nil, errors.E(op, scnr.Err())
	}
	return list, nil
}

func (s *service) Info(ctx context.Context, mod, ver string) ([]byte, error) {
	const op errors.Op = "external.Info"
	body, err := s.getRequest(ctx, mod, ver, "info")
	if err != nil {
		return nil, errors.E(op, err)
	}
	info, err := ioutil.ReadAll(body)
	if err != nil {
		return nil, errors.E(op, err)
	}
	return info, nil
}

func (s *service) GoMod(ctx context.Context, mod, ver string) ([]byte, error) {
	const op errors.Op = "external.GoMod"
	body, err := s.getRequest(ctx, mod, ver, "mod")
	if err != nil {
		return nil, errors.E(op, err)
	}
	modFile, err := ioutil.ReadAll(body)
	if err != nil {
		return nil, errors.E(op, err)
	}
	return modFile, nil
}

func (s *service) Zip(ctx context.Context, mod, ver string) (io.ReadCloser, error) {
	const op errors.Op = "external.Zip"
	body, err := s.getRequest(ctx, mod, ver, "zip")
	if err != nil {
		return nil, errors.E(op, err)
	}
	return body, nil
}

func (s *service) Save(ctx context.Context, mod, ver string, modFile []byte, zip io.Reader, info []byte) error {
	const op errors.Op = "external.Save"
	var err error
	mod, err = module.EscapePath(mod)
	if err != nil {
		panic(err)
	}
	url := s.url + "/" + mod + "/@v/" + ver + ".save"
	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)
	go func() {
		err := upload(mw, modFile, info, zip)
		pw.CloseWithError(err)
	}()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, pr)
	if err != nil {
		return errors.E(op, err)
	}
	req.Header.Add("Content-Type", mw.FormDataContentType())
	resp, err := s.c.Do(req)
	if err != nil {
		return errors.E(op, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		bts, _ := ioutil.ReadAll(resp.Body)
		return errors.E(op, fmt.Errorf("unexpected status code: %v - body: %s", resp.StatusCode, bts), resp.StatusCode)
	}
	return nil
}

func (s *service) Delete(ctx context.Context, mod, ver string) error {
	const op errors.Op = "external.Delete"
	body, err := s.doRequest(ctx, "DELETE", mod, ver, "delete")
	if err != nil {
		return errors.E(op, err)
	}
	defer body.Close()
	return nil
}

func upload(mw *multipart.Writer, mod, info []byte, zip io.Reader) error {
	defer mw.Close()
	infoW, err := mw.CreateFormFile("mod.info", "mod.info")
	if err != nil {
		return fmt.Errorf("error creating info file: %v", err)
	}
	_, err = infoW.Write(info)
	if err != nil {
		return fmt.Errorf("error writing info file: %v", err)
	}
	modW, err := mw.CreateFormFile("mod.mod", "mod.mod")
	if err != nil {
		return fmt.Errorf("error creating mod file: %v", err)
	}
	_, err = modW.Write(mod)
	if err != nil {
		return fmt.Errorf("error writing mod file: %v", err)
	}
	zipW, err := mw.CreateFormFile("mod.zip", "mod.zip")
	if err != nil {
		return fmt.Errorf("error creating zip file: %v", err)
	}
	_, err = io.Copy(zipW, zip)
	if err != nil {
		return fmt.Errorf("error writing zip file: %v", err)
	}
	return nil
}

func (s *service) getRequest(ctx context.Context, mod, ver, ext string) (io.ReadCloser, error) {
	return s.doRequest(ctx, "GET", mod, ver, ext)
}

func (s *service) doRequest(ctx context.Context, method, mod, ver, ext string) (io.ReadCloser, error) {
	const op errors.Op = "external.doRequest"
	var err error
	mod, err = module.EscapePath(mod)
	if err != nil {
		return nil, errors.E(op, err)
	}
	url := s.url + "/" + mod + "/@v/" + ver
	if ext != "" {
		url += "." + ext
	}
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, errors.E(op, err)
	}
	resp, err := s.c.Do(req)
	if err != nil {
		return nil, errors.E(op, err)
	}
	if resp.StatusCode != 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, errors.E(op, fmt.Errorf("none 200 status code: %v - body: %s", resp.StatusCode, body), resp.StatusCode)
	}
	return resp.Body, nil
}
