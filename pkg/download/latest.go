package download

import (
	"encoding/json"
	"net/http"

	"github.com/gomods/athens/pkg/download/mode"
	"github.com/gomods/athens/pkg/errors"
	"github.com/gomods/athens/pkg/log"
	"github.com/gomods/athens/pkg/paths"
	"github.com/gomods/athens/pkg/storage"
	"github.com/sirupsen/logrus"
)

// PathLatest URL.
const PathLatest = "/{module:.+}/@latest"

// LatestHandler implements GET baseURL/module/@latest
func LatestHandler(dp Protocol, lggr log.Entry, df *mode.DownloadFile) http.Handler {
	const op errors.Op = "download.LatestHandler"
	f := func(w http.ResponseWriter, r *http.Request) {
		mod, err := paths.GetModule(r)
		if err != nil {
			lggr.SystemErr(errors.E(op, err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		info, err := dp.Latest(r.Context(), mod)
		if err != nil {
			severityLevel := errors.Expect(err, errors.KindNotFound)
			err = errors.E(op, err, severityLevel)
			lggr.SystemErr(err)
			w.WriteHeader(errors.Kind(err))
			return
		}

		if err = json.NewEncoder(w).Encode(info); err != nil {
			lggr.SystemErr(errors.E(op, err))
		}
	}
	return http.HandlerFunc(f)
}

func OfflineLatestHandler(lggr log.Entry, s storage.Backend) http.Handler {
	const op errors.Op = "download.LatestHandler"
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
		if len(versions) < 1 {
			err := errors.E(op, err, logrus.ErrorLevel)
			lggr.SystemErr(err)
			w.WriteHeader(errors.Kind(err))
			return
		}

		latestVersion := versions[0]
		info, err := s.Info(r.Context(), mod, latestVersion)
		if err != nil {
			err := errors.E(op, err, logrus.ErrorLevel)
			lggr.SystemErr(err)
			w.WriteHeader(errors.Kind(err))
			return
		}

		w.Write(info)
	}
	return http.HandlerFunc(f)

}
