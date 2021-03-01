package download

import (
	"net/http"
	"net/url"
	"path"

	"github.com/gomods/athens/pkg/download/mode"
	"github.com/gomods/athens/pkg/log"
	"github.com/gomods/athens/pkg/middleware"
	"github.com/gomods/athens/pkg/storage"
	"github.com/gorilla/mux"
)

// ProtocolHandler is a function that takes all that it needs to return
// a ready-to-go http handler that serves up cmd/go's download protocol.
type ProtocolHandler func(dp Protocol, lggr log.Entry, df *mode.DownloadFile) http.Handler

// OfflineProtocolHandler is a function that takes all it needs to return
// a ready-to-go http handler that serves up module information directly
// from storage, not using any sources on the internet
type OfflineProtocolHandler func(lggr log.Entry, s storage.Backend) http.Handler

// HandlerOpts are the generic options
// for a ProtocolHandler
type HandlerOpts struct {
	Protocol     Protocol
	Logger       *log.Logger
	DownloadFile *mode.DownloadFile
}

type OfflineHandlerOpts struct {
	Storage *storage.Backend
}

// LogEntryHandler pulls a log entry from the request context. Thanks to the
// LogEntryMiddleware, we should have a log entry stored in the context for each
// request with request-specific fields. This will grab the entry and pass it to
// the protocol handlers
func LogEntryHandler(ph ProtocolHandler, opts *HandlerOpts) http.Handler {
	f := func(w http.ResponseWriter, r *http.Request) {
		ent := log.EntryFromContext(r.Context())
		handler := ph(opts.Protocol, ent, opts.DownloadFile)
		handler.ServeHTTP(w, r)
	}
	return http.HandlerFunc(f)
}

func OfflineLogEntryHandler(ph OfflineProtocolHandler, opts *OfflineHandlerOpts) http.Handler {
	f := func(w http.ResponseWriter, r *http.Request) {
		ent := log.EntryFromContext(r.Context())
		handler := ph(ent, *opts.Storage)
		handler.ServeHTTP(w, r)
	}
	return http.HandlerFunc(f)
}

// RegisterHandlers is a convenience method that registers
// all the download protocol paths for you.
func RegisterHandlers(r *mux.Router, opts *HandlerOpts) {
	// If true, this would only panic at boot time, static nil checks anyone?
	if opts == nil || opts.Protocol == nil || opts.Logger == nil {
		panic("absolutely unacceptable handler opts")
	}
	noCacheMw := middleware.CacheControl("no-cache, no-store, must-revalidate")

	listHandler := LogEntryHandler(ListHandler, opts)
	r.Handle(PathList, noCacheMw(listHandler))

	latestHandler := LogEntryHandler(LatestHandler, opts)
	r.Handle(PathLatest, noCacheMw(latestHandler)).Methods(http.MethodGet)

	r.Handle(PathVersionInfo, LogEntryHandler(InfoHandler, opts)).Methods(http.MethodGet)
	r.Handle(PathVersionModule, LogEntryHandler(ModuleHandler, opts)).Methods(http.MethodGet)
	r.Handle(PathVersionZip, LogEntryHandler(ZipHandler, opts)).Methods(http.MethodGet, http.MethodHead)
}

func getRedirectURL(base, downloadPath string) (string, error) {
	url, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	url.Path = path.Join(url.Path, downloadPath)
	return url.String(), nil
}

type OfflineHandlerOpts struct {
	Storage storage.Backend
	Logger  *log.Logger
	// DownloadFile *mode.DownloadFile
}

func RegisterOfflineHandlers(r *mux.Router, opts *OfflineHandlerOpts) {
	// If true, this would only panic at boot time, static nil checks anyone?
	if opts == nil || opts.Logger == nil {
		panic("absolutely unacceptable handler opts")
	}
	noCacheMw := middleware.CacheControl("no-cache, no-store, must-revalidate")

	listHandler := OfflineLogEntryHandler(OfflineListHandler, opts)
	r.Handle(PathList, noCacheMw(listHandler))

	latestHandler := OfflineLogEntryHandler(OfflineLatestHandler, opts)
	r.Handle(PathLatest, noCacheMw(latestHandler)).Methods(http.MethodGet)

	r.Handle(PathVersionInfo, OfflineLogEntryHandler(OfflineInfoHandler, opts)).Methods(http.MethodGet)
	r.Handle(PathVersionModule, OfflineLogEntryHandler(OfflineModuleHandler, opts)).Methods(http.MethodGet)
	r.Handle(PathVersionZip, OfflineLogEntryHandler(OfflineZipHandler, opts)).Methods(http.MethodGet)
}
