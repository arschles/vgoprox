package actions

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/gomods/athens/pkg/paths"
)

func sumdbProxy(url *url.URL, nosumPatterns []string) http.Handler {
	rp := httputil.NewSingleHostReverseProxy(url)
	rp.Director = func(req *http.Request) {
		req.Host = url.Host
		req.URL.Scheme = url.Scheme
		req.URL.Host = url.Host
	}
	if len(nosumPatterns) > 0 {
		return noSumWrapper(rp, url.Host, nosumPatterns)
	}
	return rp
}

func noSumWrapper(h http.Handler, host string, patterns []string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/lookup/") {
			for _, p := range patterns {
				if paths.MatchesPattern(p, r.URL.Path[len("/lookup/"):]) {
					w.WriteHeader(http.StatusForbidden)
					return
				}
			}
		}
		h.ServeHTTP(w, r)
	})
}
