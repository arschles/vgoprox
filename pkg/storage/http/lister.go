package http

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/gomods/athens/pkg/errors"
	"github.com/gomods/athens/pkg/observ"
	"golang.org/x/net/html"
)

// List lists all versions of a module
func (s *ModuleStore) List(ctx context.Context, module string) ([]string, error) {
	const op errors.Op = "http.List"
	ctx, span := observ.StartSpan(ctx, op.String())
	defer span.End()

	baseURL := s.moduleRoot(module)

	req, _ := http.NewRequest(http.MethodGet, baseURL, nil)
	req.SetBasicAuth(s.username, s.password)
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, errors.E(op, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		kind := errors.KindUnexpected
		if resp.StatusCode == http.StatusNotFound {
			// This is a unique case because this is actually okay! It means that there
			// are no versions here.
			return []string{}, nil
		}
		return nil, errors.E(op, kind, errors.M(module))
	}

	// Find all the links to ".mod" files within the current module
	//
	// Ok, so admittedly this scheme for listing versions is a little harebrained.
	// But that said it _is_ pretty standard for directory indexes to be formatted
	// very closely to what we expect here. I'm not settled on this but it does
	// work surprisingly well.
	mods, err := collectLinks(resp.Body, func(ref string) bool {
		return strings.HasPrefix(absolute(baseURL, ref), baseURL) && strings.HasSuffix(ref, ".mod")
	})

	// convert mod file references to versions
	versions := make([]string, len(mods))
	for i, m := range mods {
		versions[i] = strings.TrimSuffix(path.Base(m), ".mod")
	}

	return versions, nil
}

// absolute turns href into an absolute url, using the given base if href turns out to be relative.
func absolute(base, href string) string {

	// if it's already absolute, return it
	hrefURL, _ := url.Parse(href)
	if hrefURL.IsAbs() {
		return href
	}

	// if not, resolve relative to the url we fetched
	bu, _ := url.Parse(base)
	bu.Path = path.Join(bu.Path, href)
	return bu.String()

}

// collectLinks traverses the contents of r, which is assumed to be HTML, and gathers
// all <a> tag references that pass filter.
func collectLinks(r io.Reader, filter func(string) bool) ([]string, error) {

	t := html.NewTokenizer(r)

	var links []string
	var tt html.TokenType

	for tt = t.Next(); tt != html.ErrorToken; tt = t.Next() {
		switch tt {
		case html.StartTagToken:
			tn, _ := t.TagName()
			if !bytes.Equal(tn, []byte{'a'}) {
				continue
			}

			// include links that have an href that pass the filter (if any)
			attrs := getAttrs(t)
			if href, ok := attrs["href"]; ok {
				if filter == nil || filter(href) {
					links = append(links, string(href))
				}
			}
		}
	}

	return links, t.Err()
}

// getAttrs gathers the attributes of the current token.
func getAttrs(t *html.Tokenizer) map[string]string {
	attrs := make(map[string]string)
	for {
		k, v, more := t.TagAttr()
		if len(k) > 0 {
			attrs[string(k)] = string(v)
		}
		if !more {
			break
		}
	}
	return attrs
}