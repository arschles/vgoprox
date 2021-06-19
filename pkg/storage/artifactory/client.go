package artifactory

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/gomods/athens/pkg/config"
	"github.com/gomods/athens/pkg/errors"
	"github.com/gomods/athens/pkg/storage"

	"golang.org/x/mod/module"
)

type service struct {
	baseURL *url.URL
	repo    string
	user    string
	pass    string
	c       *http.Client
}

// New returns an artifactory storage client
func New(conf *config.ArtifactoryConfig, c *http.Client) (storage.Backend, error) {
	const op errors.Op = "artifactory.New"
	if c == nil {
		c = &http.Client{}
	}
	u, err := url.Parse(conf.URL)
	if err != nil {
		return nil, errors.E(op, err)
	}
	var password string
	switch {
	case conf.Password != "":
		password = conf.Password
	case conf.APIKey != "":
		password = conf.APIKey
	case conf.AccessToken != "":
		password = conf.AccessToken
	}
	return &service{
		baseURL: u,
		repo:    conf.Repository,
		user:    conf.Username,
		pass:    password,
		c:       c,
	}, nil
}

func (s *service) req(ctx context.Context, method string, u *url.URL, body io.Reader) (*http.Request, error) {
	const op errors.Op = "artifactory.req"
	req, err := http.NewRequest(method, s.baseURL.ResolveReference(u).String(), body)
	if err != nil {
		return nil, errors.E(op, err)
	}
	req.WithContext(ctx)
	if s.pass != "" || s.user != "" {
		req.SetBasicAuth(s.user, s.pass)
	}
	return req, nil
}

func (s *service) fileKey(mod, version, ext string) (string, error) {
	const op errors.Op = "artifactory.fileKey"
	var err error
	mod, err = module.EscapePath(mod)
	if err != nil {
		return "", errors.E(op, err)
	}
	if version == "" {
		return path.Join(s.repo, mod), nil
	}
	if ext == "" {
		return path.Join(s.repo, mod, version), nil
	}
	return path.Join(s.repo, mod, version, ext), nil
}

func (s *service) getRequest(ctx context.Context, mod, version, ext string) (*http.Request, error) {
	const op errors.Op = "artifactory.getRequest"
	fileKey, err := s.fileKey(mod, version, ext)
	if err != nil {
		return nil, errors.E(op, err)
	}
	req, err := s.req(ctx, http.MethodGet, &url.URL{
		Path: fileKey,
	}, nil)
	if err != nil {
		return nil, errors.E(op, err)
	}
	return req, nil
}

func (s *service) do(req *http.Request) ([]byte, error) {
	const op errors.Op = "artifactory.do"
	resp, err := s.c.Do(req)
	if err != nil {
		return nil, errors.E(op, err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.E(op, err)
	}
	if resp.StatusCode != 200 {
		return nil, errors.E(op, fmt.Errorf("non 200 status code: %v - body: %s", resp.StatusCode, body), resp.StatusCode)
	}
	return body, nil
}

func (s *service) List(ctx context.Context, mod string) ([]string, error) {
	const op errors.Op = "artifactory.List"
	fileKey, err := s.fileKey(mod, "", "")
	if err != nil {
		return nil, errors.E(op, err)
	}
	req, err := s.req(ctx, http.MethodGet, &url.URL{
		Path: path.Join("api", "storage", fileKey),
	}, nil)
	if err != nil {
		return nil, errors.E(op, err)
	}
	req.Header.Add("Accept", "application/vnd.org.jfrog.artifactory.storage.FolderInfo+json")
	resp, err := s.c.Do(req)
	if err != nil {
		return nil, errors.E(op, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		return nil, nil
	} else if resp.StatusCode != 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, errors.E(op, fmt.Errorf("non 200 status code: %v - body: %s", resp.StatusCode, body), resp.StatusCode)
	}
	var folderResp struct {
		Children []struct {
			URI    string `json:"uri"`
			Folder bool   `json:"folder"`
		} `json:"children"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&folderResp); err != nil {
		return nil, errors.E(op, err)
	}
	var versions []string
	for _, child := range folderResp.Children {
		if !child.Folder {
			continue
		}
		version := strings.Trim(child.URI, "/")
		versions = append(versions, version)
	}
	return versions, nil
}

func (s *service) Info(ctx context.Context, mod, ver string) ([]byte, error) {
	const op errors.Op = "artifactory.Info"
	req, err := s.getRequest(ctx, mod, ver, "mod.info")
	if err != nil {
		return nil, errors.E(op, err)
	}
	modFile, err := s.do(req)
	if err != nil {
		return nil, errors.E(op, err)
	}
	return modFile, nil
}

func (s *service) GoMod(ctx context.Context, mod, ver string) ([]byte, error) {
	const op errors.Op = "artifactory.GoMod"
	req, err := s.getRequest(ctx, mod, ver, "mod.mod")
	if err != nil {
		return nil, errors.E(op, err)
	}
	modFile, err := s.do(req)
	if err != nil {
		return nil, errors.E(op, err)
	}
	return modFile, nil
}

func (s *service) Zip(ctx context.Context, mod, ver string) (io.ReadCloser, error) {
	const op errors.Op = "artifactory.Zip"
	req, err := s.getRequest(ctx, mod, ver, "mod.zip")
	if err != nil {
		return nil, errors.E(op, err)
	}
	modFile, err := s.do(req)
	if err != nil {
		return nil, errors.E(op, err)
	}
	return ioutil.NopCloser(bytes.NewReader(modFile)), nil
}

func (s *service) Save(ctx context.Context, mod, ver string, modFile []byte, modZip io.Reader, info []byte) error {
	const op errors.Op = "artifactory.Save"
	var err error
	fileKey, err := s.fileKey(mod, ver, ".zip")
	if err != nil {
		return errors.E(op, err)
	}
	pr, pw := io.Pipe()
	zw := zip.NewWriter(pw)
	go func() {
		err := upload(zw, modFile, info, modZip)
		pw.CloseWithError(err)
	}()
	req, err := s.req(ctx, http.MethodPut, &url.URL{
		Path: fileKey,
	}, pr)
	if err != nil {
		return errors.E(op, err)
	}
	req.Header.Add("Content-Type", "application/zip")
	req.Header.Add("X-Explode-Archive", "true")
	req.Header.Add("X-Explode-Archive-Atomic", "true")
	if _, err := s.do(req); err != nil {
		return errors.E(op, err)
	}
	return nil
}

func (s *service) Delete(ctx context.Context, mod, ver string) error {
	const op errors.Op = "artifactory.Delete"
	fileKey, err := s.fileKey(mod, ver, "")
	if err != nil {
		return errors.E(op, err)
	}
	req, err := s.req(ctx, http.MethodDelete, &url.URL{
		Path: fileKey,
	}, nil)
	if _, err := s.do(req); err != nil {
		return errors.E(op, err)
	}
	return nil
}

func upload(zw *zip.Writer, mod, info []byte, zip io.Reader) error {
	defer zw.Close()
	infoW, err := zw.Create("mod.info")
	if err != nil {
		return fmt.Errorf("error creating info file: %v", err)
	}
	_, err = infoW.Write(info)
	if err != nil {
		return fmt.Errorf("error writing info file: %v", err)
	}
	modW, err := zw.Create("mod.mod")
	if err != nil {
		return fmt.Errorf("error creating mod file: %v", err)
	}
	_, err = modW.Write(mod)
	if err != nil {
		return fmt.Errorf("error writing mod file: %v", err)
	}
	zipW, err := zw.Create("mod.zip")
	if err != nil {
		return fmt.Errorf("error creating zip file: %v", err)
	}
	_, err = io.Copy(zipW, zip)
	if err != nil {
		return fmt.Errorf("error writing zip file: %v", err)
	}
	return nil
}
