package actions

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/gomods/athens/pkg/log"
	"github.com/gorilla/mux"
)

const healthWarning = "/healthz received none or incorrect Basic-Auth headers"

func basicAuth(user, pass string) mux.MiddlewareFunc {
	return func(h http.Handler) http.Handler {
		f := func(w http.ResponseWriter, r *http.Request) {
			if !checkAuth(r, user, pass) {
				// Helpful hint for Kubernetes users:
				// if they forget to send auth headers
				// kubernetes silently fails, so a log
				// might help them.
				if strings.HasSuffix(r.URL.Path, "/healthz") {
					lggr := log.EntryFromContext(r.Context())
					lggr.Warnf(healthWarning)
				}
				w.Header().Set("WWW-Authenticate", `Basic realm="basic auth required"`)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			h.ServeHTTP(w, r)
		}
		return http.HandlerFunc(f)
	}
}

func checkAuth(r *http.Request, user, pass string) bool {
	givenUser, givenPass, ok := r.BasicAuth()
	if !ok {
		return false
	}

	isUser := subtle.ConstantTimeCompare([]byte(user), []byte(givenUser))
	if isUser != 1 {
		return false
	}

	isPass := subtle.ConstantTimeCompare([]byte(pass), []byte(givenPass))
	if isPass != 1 {
		return false
	}

	return true
}
