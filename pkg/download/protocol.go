package download

import (
	"context"
	"regexp"
	"strings"
	"sync"

	"github.com/gomods/athens/pkg/download/mode"
	"github.com/gomods/athens/pkg/errors"
	"github.com/gomods/athens/pkg/module"
	"github.com/gomods/athens/pkg/observ"
	"github.com/gomods/athens/pkg/stash"
	"github.com/gomods/athens/pkg/storage"
)

// Protocol is the download protocol which mirrors
// the http requests that cmd/go makes to the proxy.
type Protocol interface {
	// List implements GET /{module}/@v/list
	List(ctx context.Context, mod string) ([]string, error)

	// Info implements GET /{module}/@v/{version}.info
	Info(ctx context.Context, mod, ver string) ([]byte, error)

	// Latest implements GET /{module}/@latest
	Latest(ctx context.Context, mod string) (*storage.RevInfo, error)

	// GoMod implements GET /{module}/@v/{version}.mod
	GoMod(ctx context.Context, mod, ver string) ([]byte, error)

	// Zip implements GET /{module}/@v/{version}.zip
	Zip(ctx context.Context, mod, ver string) (storage.SizeReadCloser, error)
}

// Wrapper helps extend the main protocol's functionality with addons.
type Wrapper func(Protocol) Protocol

// Opts specifies download protocol options to avoid long func signature.
type Opts struct {
	Storage      storage.Backend
	Stasher      stash.Stasher
	Lister       module.UpstreamLister
	DownloadFile *mode.DownloadFile
	NetworkMode  string
}

// NetworkMode constants
const (
	Strict   = "strict"
	Offline  = "offline"
	Fallback = "fallback"
)

// New returns a full implementation of the download.Protocol
// that the proxy needs. New also takes a variadic list of wrappers
// to extend the protocol's functionality (see addons package).
// The wrappers are applied in order, meaning the last wrapper
// passed is the Protocol that gets hit first.
func New(opts *Opts, wrappers ...Wrapper) Protocol {
	if opts.DownloadFile == nil {
		opts.DownloadFile = &mode.DownloadFile{Mode: mode.Sync}
	}
	var p Protocol = &protocol{opts.DownloadFile, opts.Storage, opts.Stasher, opts.Lister, opts.NetworkMode}
	for _, w := range wrappers {
		p = w(p)
	}

	return p
}

type protocol struct {
	df          *mode.DownloadFile
	storage     storage.Backend
	stasher     stash.Stasher
	lister      module.UpstreamLister
	networkMode string
}

func (p *protocol) List(ctx context.Context, mod string) ([]string, error) {
	const op errors.Op = "protocol.List"
	ctx, span := observ.StartSpan(ctx, op.String())
	defer span.End()

	var strList, goList []string
	var sErr, goErr error
	var wg sync.WaitGroup

	/*
		TODO: potential refactor:

		Storage Lister: just return stuff from storage, or error otherwise.

		FallbackVCS lister: list from VCS, return empty list if error.

		StrictVCS Lister: list from VCS, error if it doesn't succeed.

		UnionLister(listers ...Lister): combines any number of listers.
	*/
	wg.Add(1)
	go func() {
		defer wg.Done()
		strList, sErr = p.storage.List(ctx, mod)
	}()

	if p.networkMode != Offline {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, goList, goErr = p.lister.List(ctx, mod)
		}()
	}

	wg.Wait()

	// if we got an unexpected storage err then we can not guarantee that the end result contains all versions
	// a tag or repo could have been deleted
	if sErr != nil {
		return nil, errors.E(op, sErr)
	}

	// if we're in offline mode, just return what came from storage.
	if p.networkMode == Offline {
		return strList, nil
	}

	// if i.e. github is unavailable we should fail as well so that the behavior of the proxy is stable.
	// otherwise we will get different results the next time because i.e. GH is up again
	isUnexpGoErr := goErr != nil && !errors.IsRepoNotFoundErr(goErr)
	if isUnexpGoErr && p.networkMode == Strict {
		return nil, errors.E(op, goErr)
	}

	// if we're in fallback mode, and VCS is down, just return what we have in storage,
	// don't remove any pseudo versions.
	if isUnexpGoErr && p.networkMode == Fallback {
		return strList, nil
	}

	isRepoNotFoundErr := goErr != nil && errors.IsRepoNotFoundErr(goErr)
	storageEmpty := len(strList) == 0
	// if storage has no versions, and the repo was deleted/not-found, we know for sure
	// there are no versions that Athens can serve, so just return an error.
	if isRepoNotFoundErr && storageEmpty {
		return nil, errors.E(op, errors.M(mod), errors.KindNotFound, goErr)
	}

	strListSemVers := removePseudoVersions(strList)
	// if the repo does not exist but athens already saved some versions
	// return those so that running go get github.com/my/mod gives us the newest saved version
	// we should only do that if exclusively pseudo-versions have been saved
	// otherwise @latest would not return the latest stable version but latest commit
	if isRepoNotFoundErr && len(strListSemVers) == 0 {
		return strList, nil
	}
	// if the repo exists we have to filter out pseudo versions to prevent following scenario:
	// user does go get github.com/my/mod
	// there is no sem-ver and so the /list endpoint returns nothing, then /latests gets hit
	// Athens saves the pseudo version x1
	// from now on every time user runs go get github.com/my/mod she/he will get pseudo version x1 even if a newer version x2 exists
	// this is because /list returns non-empty list of versions (x1) and so /latest wont get hit
	return union(goList, strListSemVers), nil
}

var pseudoVersionRE = regexp.MustCompile(`^v[0-9]+\.(0\.0-|\d+\.\d+-([^+]*\.)?0\.)\d{14}-[A-Za-z0-9]+(\+incompatible)?$`)

func removePseudoVersions(allVersions []string) []string {
	var vers []string
	for _, v := range allVersions {
		// copied from go cmd https://github.com/golang/go/blob/master/src/cmd/go/internal/modfetch/pseudo.go#L93
		isPseudoVersion := strings.Count(v, "-") >= 2 && pseudoVersionRE.MatchString(v)
		if !isPseudoVersion {
			vers = append(vers, v)
		}
	}
	return vers
}

func (p *protocol) Latest(ctx context.Context, mod string) (*storage.RevInfo, error) {
	const op errors.Op = "protocol.Latest"
	ctx, span := observ.StartSpan(ctx, op.String())
	defer span.End()
	if p.networkMode == Offline {
		// Go never pings the /@latest endpoint _first_. It always tries /list and if that
		// endpoint returns an empty list then it fallsback to calling /@latest.
		return nil, errors.E(op, "Athens is in offline mode, use /list endpoint", errors.KindNotFound)
	}
	lr, _, err := p.lister.List(ctx, mod)
	if err != nil {
		return nil, errors.E(op, err)
	}

	return lr, nil
}

func (p *protocol) Info(ctx context.Context, mod, ver string) ([]byte, error) {
	const op errors.Op = "protocol.Info"
	ctx, span := observ.StartSpan(ctx, op.String())
	defer span.End()
	info, err := p.storage.Info(ctx, mod, ver)
	if errors.IsNotFoundErr(err) {
		err = p.processDownload(ctx, mod, ver, func(newVer string) error {
			info, err = p.storage.Info(ctx, mod, newVer)
			return err
		})
	}
	if err != nil {
		return nil, errors.E(op, err)
	}

	return info, nil
}

func (p *protocol) GoMod(ctx context.Context, mod, ver string) ([]byte, error) {
	const op errors.Op = "protocol.GoMod"
	ctx, span := observ.StartSpan(ctx, op.String())
	defer span.End()
	goMod, err := p.storage.GoMod(ctx, mod, ver)
	if errors.IsNotFoundErr(err) {
		err = p.processDownload(ctx, mod, ver, func(newVer string) error {
			goMod, err = p.storage.GoMod(ctx, mod, newVer)
			return err
		})
	}
	if err != nil {
		return nil, errors.E(op, err)
	}
	return goMod, nil
}

func (p *protocol) Zip(ctx context.Context, mod, ver string) (storage.SizeReadCloser, error) {
	const op errors.Op = "protocol.Zip"
	ctx, span := observ.StartSpan(ctx, op.String())
	defer span.End()
	zip, err := p.storage.Zip(ctx, mod, ver)
	if errors.IsNotFoundErr(err) {
		err = p.processDownload(ctx, mod, ver, func(newVer string) error {
			zip, err = p.storage.Zip(ctx, mod, newVer)
			return err
		})
	}
	if err != nil {
		return nil, errors.E(op, err)
	}

	return zip, nil
}

func (p *protocol) processDownload(ctx context.Context, mod, ver string, f func(newVer string) error) error {
	const op errors.Op = "protocol.processDownload"
	switch p.df.Match(mod) {
	case mode.Sync:
		newVer, err := p.stasher.Stash(ctx, mod, ver)
		if err != nil {
			return errors.E(op, err)
		}
		return f(newVer)
	case mode.Async:
		go p.stasher.Stash(ctx, mod, ver)
		return errors.E(op, "async: module not found", errors.KindNotFound)
	case mode.Redirect:
		return errors.E(op, "redirect", errors.KindRedirect)
	case mode.AsyncRedirect:
		go p.stasher.Stash(ctx, mod, ver)
		return errors.E(op, "async_redirect: module not found", errors.KindRedirect)
	case mode.None:
		return errors.E(op, "none", errors.KindNotFound)
	}
	return nil
}

// union concatenates two version lists and removes duplicates
func union(list1, list2 []string) []string {
	if list1 == nil {
		list1 = []string{}
	}
	if list2 == nil {
		list2 = []string{}
	}
	list := append(list1, list2...)
	unique := []string{}
	m := make(map[string]struct{})
	for _, v := range list {
		if _, ok := m[v]; !ok {
			unique = append(unique, v)
			m[v] = struct{}{}
		}
	}
	return unique
}
