package download

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gomods/athens/pkg/download/mode"
	"github.com/gomods/athens/pkg/errors"
	"github.com/gomods/athens/pkg/log"
	"github.com/gomods/athens/pkg/paths"
	"github.com/gomods/athens/pkg/storage"
	"github.com/sirupsen/logrus"
)

// PathList URL.
const PathList = "/{module:.+}/@v/list"

// ListHandler implements GET baseURL/module/@v/list
func ListHandler(dp Protocol, lggr log.Entry, df *mode.DownloadFile) http.Handler {
	const op errors.Op = "download.ListHandler"
	f := func(w http.ResponseWriter, r *http.Request) {
		mod, err := paths.GetModule(r)
		if err != nil {
			lggr.SystemErr(errors.E(op, err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		versions, err := dp.List(r.Context(), mod)
		if err != nil {
			severityLevel := errors.Expect(err, errors.KindNotFound)
			err = errors.E(op, err, severityLevel)
			lggr.SystemErr(err)
			w.WriteHeader(errors.Kind(err))
			return
		}

		fmt.Fprint(w, strings.Join(versions, "\n"))
	}
	return http.HandlerFunc(f)
}

// OfflineListHandler returns an http.Handler capable of serving /@v/list endpoints
// directly from storage. No network traffic to anywhere other than the storage
// backend will be attempted
func OfflineListHandler(lggr log.Entry, s storage.Backend) http.Handler {
	const op errors.Op = "download.OfflineListHandler"
	f := func(w http.ResponseWriter, r *http.Request) {
		mod, err := paths.GetModule(r)
		if err != nil {
			lggr.SystemErr(errors.E(op, err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		versions, err := s.List(r.Context(), mod)
		if err != nil {
			err = errors.E(op, err, logrus.ErrorLevel)
			lggr.SystemErr(err)
			w.WriteHeader(errors.Kind(err))
			return
		}
		fmt.Fprint(w, strings.Join(versions, "\n"))
	}

	return http.HandlerFunc(f)

}
