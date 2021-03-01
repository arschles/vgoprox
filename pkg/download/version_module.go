package download

import (
	"net/http"

	"github.com/gomods/athens/pkg/download/mode"
	"github.com/gomods/athens/pkg/errors"
	"github.com/gomods/athens/pkg/log"
	"github.com/gomods/athens/pkg/storage"
)

// PathVersionModule URL.
const PathVersionModule = "/{module:.+}/@v/{version}.mod"

// ModuleHandler implements GET baseURL/module/@v/version.mod
func ModuleHandler(dp Protocol, lggr log.Entry, df *mode.DownloadFile) http.Handler {
	const op errors.Op = "download.VersionModuleHandler"
	f := func(w http.ResponseWriter, r *http.Request) {
		mod, ver, err := getModuleParams(r, op)
		if err != nil {
			err = errors.E(op, errors.M(mod), errors.V(ver), err)
			lggr.SystemErr(err)
			w.WriteHeader(errors.Kind(err))
			return
		}
		modBts, err := dp.GoMod(r.Context(), mod, ver)
		if err != nil {
			severityLevel := errors.Expect(err, errors.KindNotFound, errors.KindRedirect)
			err = errors.E(op, err, severityLevel)
			lggr.SystemErr(err)
			if errors.Kind(err) == errors.KindRedirect {
				url, err := getRedirectURL(df.URL(mod), r.URL.Path)
				if err != nil {
					err = errors.E(op, errors.M(mod), errors.V(ver), err)
					lggr.SystemErr(err)
					w.WriteHeader(errors.Kind(err))
					return
				}
				http.Redirect(w, r, url, errors.KindRedirect)
				return
			}
			w.WriteHeader(errors.Kind(err))
			return
		}

		w.Write(modBts)
	}
	return http.HandlerFunc(f)
}

// OfflineModuleHandler implements GET baseURL/module/@v/version.mod
func OfflineModuleHandler(lggr log.Entry, s storage.Backend) http.Handler {
	const op errors.Op = "download.VersionModuleHandler"
	f := func(w http.ResponseWriter, r *http.Request) {
		mod, ver, err := getModuleParams(r, op)
		if err != nil {
			err = errors.E(op, errors.M(mod), errors.V(ver), err)
			lggr.SystemErr(err)
			w.WriteHeader(errors.Kind(err))
			return
		}
		modBytes, err := s.GoMod(r.Context(), mod, ver)
		w.Write(modBytes)
	}
	return http.HandlerFunc(f)
}
