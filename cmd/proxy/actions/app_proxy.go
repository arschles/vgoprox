package actions

import (
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/gomods/athens/pkg/config"
	"github.com/gomods/athens/pkg/download"
	"github.com/gomods/athens/pkg/download/addons"
	"github.com/gomods/athens/pkg/download/mode"
	"github.com/gomods/athens/pkg/index"
	"github.com/gomods/athens/pkg/index/mem"
	"github.com/gomods/athens/pkg/index/mysql"
	"github.com/gomods/athens/pkg/index/nop"
	"github.com/gomods/athens/pkg/index/postgres"
	"github.com/gomods/athens/pkg/log"
	"github.com/gomods/athens/pkg/module"
	"github.com/gomods/athens/pkg/stash"
	"github.com/gomods/athens/pkg/storage"
	"github.com/gorilla/mux"
	"github.com/spf13/afero"
)

func addProxyRoutes(
	r *mux.Router,
	s storage.Backend,
	l *log.Logger,
	c *config.Config,
) error {
	r.HandleFunc("/", proxyHomeHandler)
	r.HandleFunc("/healthz", healthHandler)
	r.HandleFunc("/readyz", getReadinessHandler(s))
	r.HandleFunc("/version", versionHandler)
	r.HandleFunc("/catalog", catalogHandler(s))
	r.HandleFunc("/robots.txt", robotsHandler(c))

	indexer, err := getIndex(c)
	if err != nil {
		return err
	}
	r.HandleFunc("/index", indexHandler(indexer))

	for _, sumdb := range c.SumDBs {
		sumdbURL, err := url.Parse(sumdb)
		if err != nil {
			return err
		}
		if sumdbURL.Scheme != "https" {
			return fmt.Errorf("sumdb: %v must have an https scheme", sumdb)
		}
		supportPath := path.Join("/sumdb", sumdbURL.Host, "/supported")
		r.HandleFunc(supportPath, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
		})
		sumHandler := sumdbProxy(sumdbURL, c.NoSumPatterns)
		pathPrefix := "/sumdb/" + sumdbURL.Host
		r.PathPrefix(pathPrefix + "/").Handler(
			http.StripPrefix(strings.TrimSuffix(c.PathPrefix, "/")+pathPrefix, sumHandler),
		)
	}

	// Download Protocol
	// the download.Protocol and the stash.Stasher interfaces are composable
	// in a middleware fashion. Therefore you can separate concerns
	// by the functionality: a download.Protocol that just takes care
	// of "go getting" things, and another Protocol that just takes care
	// of "pooling" requests etc.

	// In our case, we'd like to compose both interfaces in a particular
	// order to ensure logical ordering of execution.

	// Here's the order of an incoming request to the download.Protocol:

	// 1. The downloadpool gets hit first, and manages concurrent requests
	// 2. The downloadpool passes the request to its parent Protocol: stasher
	// 3. The stasher Protocol checks storage first, and if storage is empty
	// it makes a Stash request to the stash.Stasher interface.

	// Once the stasher picks up an order, here's how the requests go in order:
	// 1. The singleflight picks up the first request and latches duplicate ones.
	// 2. The singleflight passes the stash to its parent: stashpool.
	// 3. The stashpool manages limiting concurrent requests and passes them to stash.
	// 4. The plain stash.New just takes a request from upstream and saves it into storage.
	fs := afero.NewOsFs()

	// TODO: remove before we release v0.7.0
	if c.GoProxy != "direct" && c.GoProxy != "" {
		l.Error("GoProxy is deprecated, please use GoBinaryEnvVars")
	}
	if !c.GoBinaryEnvVars.HasKey("GONOSUMDB") {
		c.GoBinaryEnvVars.Add("GONOSUMDB", strings.Join(c.NoSumPatterns, ","))
	}
	if err := c.GoBinaryEnvVars.Validate(); err != nil {
		return err
	}
	mf, err := module.NewGoGetFetcher(c.GoBinary, c.GoGetDir, c.GoBinaryEnvVars, fs)
	if err != nil {
		return err
	}

	lister := module.NewVCSLister(c.GoBinary, c.GoBinaryEnvVars, fs)
	checker := storage.WithChecker(s)
	withSingleFlight, err := getSingleFlight(c, checker)
	if err != nil {
		return err
	}
	st := stash.New(mf, s, indexer, stash.WithPool(c.GoGetWorkers), withSingleFlight)

	df, err := mode.NewFile(c.DownloadMode, c.DownloadURL)
	if err != nil {
		return err
	}

	dpOpts := &download.Opts{
		Storage:      s,
		Stasher:      st,
		Lister:       lister,
		DownloadFile: df,
		NetworkMode:  c.NetworkMode,
	}

	dp := download.New(dpOpts, addons.WithPool(c.ProtocolWorkers))

	handlerOpts := &download.HandlerOpts{Protocol: dp, Logger: l, DownloadFile: df}
	download.RegisterHandlers(r, handlerOpts)

	return nil
}

func getSingleFlight(c *config.Config, checker storage.Checker) (stash.Wrapper, error) {
	switch c.SingleFlightType {
	case "", "memory":
		return stash.WithSingleflight, nil
	case "etcd":
		if c.SingleFlight == nil || c.SingleFlight.Etcd == nil {
			return nil, fmt.Errorf("Etcd config must be present")
		}
		endpoints := strings.Split(c.SingleFlight.Etcd.Endpoints, ",")
		return stash.WithEtcd(endpoints, checker)
	case "redis":
		if c.SingleFlight == nil || c.SingleFlight.Redis == nil {
			return nil, fmt.Errorf("Redis config must be present")
		}
		return stash.WithRedisLock(c.SingleFlight.Redis.Endpoint, c.SingleFlight.Redis.Password, checker)
	case "redis-sentinel":
		if c.SingleFlight == nil || c.SingleFlight.RedisSentinel == nil {
			return nil, fmt.Errorf("Redis config must be present")
		}
		return stash.WithRedisSentinelLock(
			c.SingleFlight.RedisSentinel.Endpoints,
			c.SingleFlight.RedisSentinel.MasterName,
			c.SingleFlight.RedisSentinel.SentinelPassword,
			checker,
		)
	case "gcp":
		if c.StorageType != "gcp" {
			return nil, fmt.Errorf("gcp SingleFlight only works with a gcp storage type and not: %v", c.StorageType)
		}
		return stash.WithGCSLock, nil
	case "azureblob":
		if c.StorageType != "azureblob" {
			return nil, fmt.Errorf("azureblob SingleFlight only works with a azureblob storage type and not: %v", c.StorageType)
		}
		return stash.WithAzureBlobLock(c.Storage.AzureBlob, c.TimeoutDuration(), checker)
	default:
		return nil, fmt.Errorf("unrecognized single flight type: %v", c.SingleFlightType)
	}
}

func getIndex(c *config.Config) (index.Indexer, error) {
	switch c.IndexType {
	case "", "none":
		return nop.New(), nil
	case "memory":
		return mem.New(), nil
	case "mysql":
		return mysql.New(c.Index.MySQL)
	case "postgres":
		return postgres.New(c.Index.Postgres)
	}
	return nil, fmt.Errorf("unknown index type: %q", c.IndexType)
}
